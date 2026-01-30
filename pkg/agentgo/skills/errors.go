package skills

import "strings"

// SkillValidationError represents validation errors for a skill
type SkillValidationError struct {
	SkillName string
	Errors    []string
}

// Error implements the error interface
func (e *SkillValidationError) Error() string {
	if len(e.Errors) == 0 {
		return "skill validation failed"
	}

	if e.SkillName != "" {
		return "skill '" + e.SkillName + "' validation failed: " + strings.Join(e.Errors, "; ")
	}

	return "skill validation failed: " + strings.Join(e.Errors, "; ")
}
