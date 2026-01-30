package skills

import (
	"fmt"
	"strings"
)

// Skill represents an agent skill
type Skill struct {
	Name         string                 `json:"name" yaml:"name"`
	Description  string                 `json:"description" yaml:"description"`
	License      string                 `json:"license,omitempty" yaml:"license,omitempty"`
	Instructions string                 `json:"instructions"`
	Scripts      map[string]*Script     `json:"scripts,omitempty"`
	References   map[string]*Reference  `json:"references,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	Path         string                 `json:"path,omitempty"`
}

// Script represents an executable script in a skill
type Script struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	Content []byte `json:"content"`
	Shebang string `json:"shebang,omitempty"`
}

// Reference represents a reference document in a skill
type Reference struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	Content []byte `json:"content"`
}

// Validate validates a skill
func (s *Skill) Validate() error {
	// Name validation
	if s.Name == "" {
		return fmt.Errorf("skill name is required")
	}
	if len(s.Name) > 64 {
		return fmt.Errorf("skill name must be 64 characters or less, got %d", len(s.Name))
	}
	if !isValidSkillName(s.Name) {
		return fmt.Errorf("skill name must be lowercase alphanumeric with hyphens only, no leading/trailing hyphens")
	}

	// Description validation
	if s.Description == "" {
		return fmt.Errorf("skill description is required")
	}
	if len(s.Description) > 1024 {
		return fmt.Errorf("skill description must be 1024 characters or less, got %d", len(s.Description))
	}

	// Instructions validation
	if s.Instructions == "" {
		return fmt.Errorf("skill instructions are required")
	}

	// Script validation
	for name, script := range s.Scripts {
		if script.Content != nil && len(script.Content) > 0 {
			// Check for shebang
			content := string(script.Content)
			if !strings.HasPrefix(content, "#!") {
				return fmt.Errorf("script '%s' must have a shebang line (#!/usr/bin/env ...)", name)
			}
		}
	}

	return nil
}

// isValidSkillName checks if a skill name follows the naming rules
func isValidSkillName(name string) bool {
	if name == "" {
		return false
	}

	// Cannot start or end with hyphen
	if name[0] == '-' || name[len(name)-1] == '-' {
		return false
	}

	// Must be lowercase alphanumeric with hyphens
	for i, char := range name {
		if (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') {
			continue
		}
		if char == '-' {
			// No consecutive hyphens
			if i > 0 && name[i-1] == '-' {
				return false
			}
			continue
		}
		return false
	}

	return true
}

// GetSummary returns a summary of the skill for system prompts
func (s *Skill) GetSummary() string {
	return fmt.Sprintf("- %s: %s", s.Name, s.Description)
}
