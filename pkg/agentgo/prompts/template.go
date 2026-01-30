package prompts

import (
	"bytes"
	"fmt"
	"text/template"
)

// PromptTemplate provides template rendering functionality
type PromptTemplate interface {
	Render(vars map[string]interface{}) (string, error)
	Validate(vars map[string]interface{}) error
	AddExample(example Example) error
}

// Template implements PromptTemplate using Go's text/template
type Template struct {
	prompt   *Prompt
	template *template.Template
}

// NewTemplate creates a new prompt template
func NewTemplate(prompt *Prompt) (*Template, error) {
	if err := prompt.Validate(); err != nil {
		return nil, fmt.Errorf("invalid prompt: %w", err)
	}

	tmpl, err := template.New(prompt.Name).Parse(prompt.Template)
	if err != nil {
		return nil, fmt.Errorf("failed to parse template: %w", err)
	}

	return &Template{
		prompt:   prompt,
		template: tmpl,
	}, nil
}

// Render renders the template with the given variables
func (t *Template) Render(vars map[string]interface{}) (string, error) {
	// Validate variables
	if err := t.Validate(vars); err != nil {
		return "", err
	}

	// Apply defaults
	varsWithDefaults := t.applyDefaults(vars)

	// Render template
	var buf bytes.Buffer
	if err := t.template.Execute(&buf, varsWithDefaults); err != nil {
		return "", fmt.Errorf("failed to render template: %w", err)
	}

	result := buf.String()

	// Add examples if present
	if len(t.prompt.Examples) > 0 {
		result = t.injectExamples(result, varsWithDefaults)
	}

	return result, nil
}

// Validate validates the variables against the prompt definition
func (t *Template) Validate(vars map[string]interface{}) error {
	for _, variable := range t.prompt.Variables {
		if variable.Required {
			if _, exists := vars[variable.Name]; !exists {
				if variable.Default == nil {
					return fmt.Errorf("required variable '%s' is missing", variable.Name)
				}
			}
		}

		// Type validation (basic)
		if val, exists := vars[variable.Name]; exists {
			if err := validateType(variable.Name, variable.Type, val); err != nil {
				return err
			}
		}
	}

	return nil
}

// AddExample adds a few-shot example to the prompt
func (t *Template) AddExample(example Example) error {
	t.prompt.Examples = append(t.prompt.Examples, example)
	return nil
}

// applyDefaults applies default values to variables
func (t *Template) applyDefaults(vars map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	// Copy provided vars
	for k, v := range vars {
		result[k] = v
	}

	// Apply defaults for missing vars
	for _, variable := range t.prompt.Variables {
		if _, exists := result[variable.Name]; !exists && variable.Default != nil {
			result[variable.Name] = variable.Default
		}
	}

	return result
}

// injectExamples injects few-shot examples into the rendered prompt
func (t *Template) injectExamples(rendered string, vars map[string]interface{}) string {
	// Check if examples should be injected
	if injectExamples, ok := vars["inject_examples"].(bool); ok && !injectExamples {
		return rendered
	}

	var examplesText string
	examplesText += "\n\nHere are some examples:\n\n"

	for i, example := range t.prompt.Examples {
		examplesText += fmt.Sprintf("Example %d:\n", i+1)
		examplesText += fmt.Sprintf("Input: %v\n", example.Input)
		examplesText += fmt.Sprintf("Output: %s\n\n", example.Output)
	}

	return rendered + examplesText
}

func validateType(name, expectedType string, value interface{}) error {
	switch expectedType {
	case "string":
		if _, ok := value.(string); !ok {
			return fmt.Errorf("variable '%s' must be a string", name)
		}
	case "integer", "number":
		switch value.(type) {
		case int, int64, float64:
			// OK
		default:
			return fmt.Errorf("variable '%s' must be a number", name)
		}
	case "boolean":
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("variable '%s' must be a boolean", name)
		}
	case "array":
		switch value.(type) {
		case []interface{}, []string, []int:
			// OK
		default:
			return fmt.Errorf("variable '%s' must be an array", name)
		}
	case "object":
		if _, ok := value.(map[string]interface{}); !ok {
			return fmt.Errorf("variable '%s' must be an object", name)
		}
	}

	return nil
}
