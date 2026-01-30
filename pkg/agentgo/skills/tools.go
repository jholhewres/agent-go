package skills

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/jholhewres/agent-go/pkg/agentgo/tools/toolkit"
)

// SkillTools provides agent tools for interacting with skills
type SkillTools struct {
	skills *Skills
}

// NewSkillTools creates a new SkillTools
func NewSkillTools(skills *Skills) *SkillTools {
	return &SkillTools{
		skills: skills,
	}
}

// AsToolkit converts SkillTools to a toolkit.Toolkit for agent integration
func (st *SkillTools) AsToolkit() toolkit.Toolkit {
	t := toolkit.NewBaseToolkit("skills")

	// get_skill_instructions tool (always available)
	t.RegisterFunction(&toolkit.Function{
		Name:        "get_skill_instructions",
		Description: "Load full instructions for a skill. Use this when you need detailed guidance on how to use a skill.",
		Parameters: map[string]toolkit.Parameter{
			"skill_name": {
				Type:        "string",
				Description: "The name of the skill to load instructions for",
				Required:    true,
			},
		},
		Handler: st.getSkillInstructions,
	})

	// get_skill_reference tool (always available)
	t.RegisterFunction(&toolkit.Function{
		Name:        "get_skill_reference",
		Description: "Load a reference document from a skill. Use this to access supporting documentation.",
		Parameters: map[string]toolkit.Parameter{
			"skill_name": {
				Type:        "string",
				Description: "The name of the skill",
				Required:    true,
			},
			"reference_path": {
				Type:        "string",
				Description: "The path to the reference document (e.g., 'style-guide.md')",
				Required:    true,
			},
		},
		Handler: st.getSkillReference,
	})

	// get_skill_script tool (only if scripts are enabled)
	if st.skills.ScriptsEnabled() {
		t.RegisterFunction(&toolkit.Function{
			Name:        "get_skill_script",
			Description: "Read or execute a script from a skill. Use this to run automated tasks.",
			Parameters: map[string]toolkit.Parameter{
				"skill_name": {
					Type:        "string",
					Description: "The name of the skill",
					Required:    true,
				},
				"script_path": {
					Type:        "string",
					Description: "The path to the script (e.g., 'check_style.py')",
					Required:    true,
				},
				"execute": {
					Type:        "boolean",
					Description: "Whether to execute the script (true) or just return its content (false)",
					Required:    false,
				},
				"args": {
					Type:        "array",
					Description: "Arguments to pass to the script when executing",
					Required:    false,
					Items: &toolkit.Parameter{
						Type: "string",
					},
				},
				"timeout": {
					Type:        "integer",
					Description: "Timeout in seconds for script execution (default: 30)",
					Required:    false,
				},
			},
			Handler: st.getSkillScript,
		})
	}

	return t
}

func (st *SkillTools) getSkillInstructions(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	skillName, ok := args["skill_name"].(string)
	if !ok || skillName == "" {
		return nil, fmt.Errorf("skill_name is required")
	}

	skill, err := st.skills.GetSkill(skillName)
	if err != nil {
		return nil, err
	}

	return skill.Instructions, nil
}

func (st *SkillTools) getSkillReference(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	skillName, ok := args["skill_name"].(string)
	if !ok || skillName == "" {
		return nil, fmt.Errorf("skill_name is required")
	}

	referencePath, ok := args["reference_path"].(string)
	if !ok || referencePath == "" {
		return nil, fmt.Errorf("reference_path is required")
	}

	skill, err := st.skills.GetSkill(skillName)
	if err != nil {
		return nil, err
	}

	reference, exists := skill.References[referencePath]
	if !exists {
		return nil, fmt.Errorf("reference '%s' not found in skill '%s'", referencePath, skillName)
	}

	return string(reference.Content), nil
}

func (st *SkillTools) getSkillScript(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	skillName, ok := args["skill_name"].(string)
	if !ok || skillName == "" {
		return nil, fmt.Errorf("skill_name is required")
	}

	scriptPath, ok := args["script_path"].(string)
	if !ok || scriptPath == "" {
		return nil, fmt.Errorf("script_path is required")
	}

	execute := false
	if execVal, ok := args["execute"].(bool); ok {
		execute = execVal
	}

	skill, err := st.skills.GetSkill(skillName)
	if err != nil {
		return nil, err
	}

	script, exists := skill.Scripts[scriptPath]
	if !exists {
		return nil, fmt.Errorf("script '%s' not found in skill '%s'", scriptPath, skillName)
	}

	// If not executing, just return content
	if !execute {
		return string(script.Content), nil
	}

	// Execute script
	timeout := 30
	if timeoutVal, ok := args["timeout"].(float64); ok {
		timeout = int(timeoutVal)
	} else if timeoutVal, ok := args["timeout"].(int); ok {
		timeout = timeoutVal
	}

	var scriptArgs []string
	if argsVal, ok := args["args"].([]interface{}); ok {
		for _, arg := range argsVal {
			if argStr, ok := arg.(string); ok {
				scriptArgs = append(scriptArgs, argStr)
			}
		}
	}

	return st.executeScript(skill, script, scriptArgs, timeout)
}

func (st *SkillTools) executeScript(skill *Skill, script *Script, args []string, timeoutSec int) (string, error) {
	if script.Shebang == "" {
		return "", fmt.Errorf("script must have a shebang line")
	}

	// Extract interpreter from shebang
	interpreter := st.extractInterpreter(script.Shebang)
	if interpreter == "" {
		return "", fmt.Errorf("invalid shebang: %s", script.Shebang)
	}

	// Create temp file for script
	tempFile := filepath.Join(os.TempDir(), fmt.Sprintf("skill-%s-%s", skill.Name, script.Name))
	if err := os.WriteFile(tempFile, script.Content, 0755); err != nil {
		return "", fmt.Errorf("failed to write temp script: %w", err)
	}
	defer os.Remove(tempFile)

	// Execute script with timeout
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutSec)*time.Second)
	defer cancel()

	cmdArgs := append([]string{tempFile}, args...)
	cmd := exec.CommandContext(ctx, interpreter, cmdArgs...)
	cmd.Dir = skill.Path // Execute in skill directory

	output, err := cmd.CombinedOutput()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("script execution timed out after %d seconds", timeoutSec)
		}
		return "", fmt.Errorf("script execution failed: %w\nOutput: %s", err, string(output))
	}

	return string(output), nil
}

// allowedInterpreters whitelist for safe script execution
var allowedInterpreters = map[string]bool{
	"sh":      true,
	"bash":    true,
	"python":  true,
	"python3": true,
	"node":    true,
	"ruby":    true,
}

func (st *SkillTools) extractInterpreter(shebang string) string {
	// Remove #! prefix
	shebang = strings.TrimPrefix(shebang, "#!")
	shebang = strings.TrimSpace(shebang)

	var interpreterName string

	// Handle /usr/bin/env python3, /usr/bin/env bash, etc
	if strings.HasPrefix(shebang, "/usr/bin/env ") {
		parts := strings.Fields(shebang)
		if len(parts) >= 2 {
			interpreterName = parts[1]
		}
	} else {
		// Handle direct paths like /bin/bash, /usr/bin/python3
		parts := strings.Fields(shebang)
		if len(parts) > 0 {
			interpreterName = filepath.Base(parts[0])
		}
	}

	if interpreterName == "" {
		return ""
	}

	// Security: Validate interpreter is in whitelist
	if !allowedInterpreters[interpreterName] {
		return "" // Triggers error in caller
	}

	// Security: Find interpreter in system PATH (don't trust shebang paths)
	validPath, err := exec.LookPath(interpreterName)
	if err != nil {
		return "" // Interpreter not found in PATH
	}

	return validPath
}
