package skills

import (
	"fmt"
	"strings"
	"sync"

	"github.com/jholhewres/agent-go/pkg/agentgo/tools/toolkit"
)

// Skills is the main orchestrator that manages skills through loaders
type Skills struct {
	loaders       []SkillLoader
	skills        map[string]*Skill
	systemPrompt  string
	enableScripts bool // Controls whether script execution tools are available
	mu            sync.RWMutex
}

// NewSkills creates a new Skills orchestrator with the given loaders
func NewSkills(loaders []SkillLoader) (*Skills, error) {
	if len(loaders) == 0 {
		return nil, fmt.Errorf("at least one skill loader is required")
	}

	s := &Skills{
		loaders:       loaders,
		skills:        make(map[string]*Skill),
		enableScripts: true, // Default: scripts enabled (current behavior)
	}

	if err := s.LoadAll(); err != nil {
		return nil, err
	}

	return s, nil
}

// LoadAll loads skills from all loaders
func (s *Skills) LoadAll() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Clear existing skills
	s.skills = make(map[string]*Skill)

	// Load from each loader (first loader wins if duplicate names)
	for _, loader := range s.loaders {
		skills, err := loader.Load()
		if err != nil {
			return fmt.Errorf("failed to load skills from %s loader: %w", loader.GetType(), err)
		}

		for _, skill := range skills {
			// Validate skill
			if err := skill.Validate(); err != nil {
				return &SkillValidationError{
					SkillName: skill.Name,
					Errors:    []string{err.Error()},
				}
			}

			// Only add if not already present (first loader wins)
			if _, exists := s.skills[skill.Name]; !exists {
				s.skills[skill.Name] = skill
			}
		}
	}

	// Generate system prompt
	s.generateSystemPrompt()

	return nil
}

// GetSkill retrieves a skill by name
func (s *Skills) GetSkill(name string) (*Skill, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	skill, exists := s.skills[name]
	if !exists {
		return nil, fmt.Errorf("skill '%s' not found", name)
	}

	return skill, nil
}

// ListSkills returns all loaded skills
func (s *Skills) ListSkills() []*Skill {
	s.mu.RLock()
	defer s.mu.RUnlock()

	skills := make([]*Skill, 0, len(s.skills))
	for _, skill := range s.skills {
		skills = append(skills, skill)
	}

	return skills
}

// GetSystemPrompt returns the system prompt with skill metadata
func (s *Skills) GetSystemPrompt() string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.systemPrompt
}

// Reload reloads all skills from their loaders
func (s *Skills) Reload() error {
	return s.LoadAll()
}

// generateSystemPrompt generates the system prompt with skill summaries
func (s *Skills) generateSystemPrompt() {
	if len(s.skills) == 0 {
		s.systemPrompt = ""
		return
	}

	var sb strings.Builder
	sb.WriteString("\nYou have access to the following skills:\n\n")

	for _, skill := range s.skills {
		sb.WriteString(skill.GetSummary())
		sb.WriteString("\n")
	}

	sb.WriteString("\nUse get_skill_instructions(skill_name) to load full instructions when needed.\n")
	sb.WriteString("Use get_skill_reference(skill_name, reference_path) to load documentation.\n")
	sb.WriteString("Use get_skill_script(skill_name, script_path, execute, args, timeout) to run scripts.\n")

	s.systemPrompt = sb.String()
}

// HasSkill checks if a skill with the given name exists
func (s *Skills) HasSkill(name string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	_, exists := s.skills[name]
	return exists
}

// Count returns the number of loaded skills
func (s *Skills) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return len(s.skills)
}

// GetToolkit returns a toolkit.Toolkit for agent integration
// This is a convenience method that creates SkillTools and converts to Toolkit
func (s *Skills) GetToolkit() toolkit.Toolkit {
	if s == nil {
		return nil
	}

	skillTools := NewSkillTools(s)
	return skillTools.AsToolkit()
}

// DisableScripts disables script execution tools for this Skills instance.
// After calling this method, get_skill_script will not be registered as a tool.
// This is useful when you want to use skills only for instructions/references
// without allowing script execution.
func (s *Skills) DisableScripts() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.enableScripts = false
}

// EnableScripts enables script execution tools for this Skills instance.
// This is the default behavior.
func (s *Skills) EnableScripts() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.enableScripts = true
}

// ScriptsEnabled returns whether script execution is enabled.
func (s *Skills) ScriptsEnabled() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.enableScripts
}
