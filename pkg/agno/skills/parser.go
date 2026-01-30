package skills

import (
	"bufio"
	"bytes"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// ParseSkillMD parses a SKILL.md file with YAML frontmatter
func ParseSkillMD(content []byte) (*Skill, error) {
	scanner := bufio.NewScanner(bytes.NewReader(content))
	
	// Check for frontmatter start
	if !scanner.Scan() || strings.TrimSpace(scanner.Text()) != "---" {
		return nil, fmt.Errorf("SKILL.md must start with YAML frontmatter (---)")
	}

	// Read frontmatter
	var frontmatterLines []string
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "---" {
			break
		}
		frontmatterLines = append(frontmatterLines, line)
	}

	if len(frontmatterLines) == 0 {
		return nil, fmt.Errorf("empty frontmatter in SKILL.md")
	}

	// Parse YAML frontmatter
	var skill Skill
	frontmatterYAML := strings.Join(frontmatterLines, "\n")
	if err := yaml.Unmarshal([]byte(frontmatterYAML), &skill); err != nil {
		return nil, fmt.Errorf("failed to parse YAML frontmatter: %w", err)
	}

	// Read markdown content (instructions)
	var instructionLines []string
	for scanner.Scan() {
		instructionLines = append(instructionLines, scanner.Text())
	}

	skill.Instructions = strings.TrimSpace(strings.Join(instructionLines, "\n"))

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading SKILL.md: %w", err)
	}

	return &skill, nil
}

// ExtractShebang extracts the shebang from script content
func ExtractShebang(content []byte) string {
	if len(content) < 2 {
		return ""
	}

	if content[0] != '#' || content[1] != '!' {
		return ""
	}

	// Find end of first line
	end := bytes.IndexByte(content, '\n')
	if end == -1 {
		end = len(content)
	}

	return strings.TrimSpace(string(content[:end]))
}
