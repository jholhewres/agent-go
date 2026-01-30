package prompts

import (
	"fmt"
	"time"
)

// Prompt represents a prompt template
type Prompt struct {
	ID          string                 `json:"id" yaml:"id"`
	Name        string                 `json:"name" yaml:"name"`
	Template    string                 `json:"template" yaml:"template"`
	Variables   []Variable             `json:"variables,omitempty" yaml:"variables,omitempty"`
	Examples    []Example              `json:"examples,omitempty" yaml:"examples,omitempty"`
	Version     string                 `json:"version,omitempty" yaml:"version,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	CreatedAt   time.Time              `json:"created_at,omitempty" yaml:"created_at,omitempty"`
}

// Variable represents a template variable
type Variable struct {
	Name        string      `json:"name" yaml:"name"`
	Type        string      `json:"type" yaml:"type"` // "string", "integer", "boolean", "array", "object"
	Required    bool        `json:"required,omitempty" yaml:"required,omitempty"`
	Default     interface{} `json:"default,omitempty" yaml:"default,omitempty"`
	Description string      `json:"description,omitempty" yaml:"description,omitempty"`
}

// Example represents a few-shot example
type Example struct {
	Input  map[string]interface{} `json:"input" yaml:"input"`
	Output string                 `json:"output" yaml:"output"`
}

// Validate validates a prompt
func (p *Prompt) Validate() error {
	if p.Name == "" {
		return fmt.Errorf("prompt name is required")
	}

	if p.Template == "" {
		return fmt.Errorf("prompt template is required")
	}

	// Validate variables
	for _, v := range p.Variables {
		if v.Name == "" {
			return fmt.Errorf("variable name is required")
		}
		if v.Type == "" {
			return fmt.Errorf("variable type is required for '%s'", v.Name)
		}
		if !isValidType(v.Type) {
			return fmt.Errorf("invalid variable type '%s' for '%s'", v.Type, v.Name)
		}
	}

	return nil
}

func isValidType(t string) bool {
	validTypes := []string{"string", "integer", "boolean", "array", "object", "number"}
	for _, valid := range validTypes {
		if t == valid {
			return true
		}
	}
	return false
}
