// Package structured provides utilities for generating JSON Schema from Go structs
// and parsing model responses into typed values.
package structured

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/jholhewres/agent-go/pkg/agentgo/models"
	"github.com/jholhewres/agent-go/pkg/agentgo/types"
)

// OutputSchema describes a JSON Schema derived from a Go struct.
type OutputSchema struct {
	Name        string                 // Name of the schema (derived from struct name)
	Description string                 // Optional description
	Schema      map[string]interface{} // JSON Schema as map
}

// SchemaFromType generates a JSON Schema from a Go struct using reflection.
// The value must be a struct or pointer to struct.
// Self-referencing types are detected and produce a generic object schema to avoid infinite recursion.
func SchemaFromType(v interface{}) (*OutputSchema, error) {
	t := reflect.TypeOf(v)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return nil, fmt.Errorf("SchemaFromType requires a struct, got %s", t.Kind())
	}

	seen := make(map[reflect.Type]bool)
	schema := typeToSchemaGuarded(t, seen)

	return &OutputSchema{
		Name:   t.Name(),
		Schema: schema,
	}, nil
}

// ToResponseFormat converts an OutputSchema to a models.ResponseFormat.
// The resulting JSONSchema map includes "name" and "description" metadata
// alongside the schema properties, ready for provider consumption.
func (s *OutputSchema) ToResponseFormat() *models.ResponseFormat {
	schema := make(map[string]interface{})
	for k, v := range s.Schema {
		schema[k] = v
	}
	schema["name"] = s.Name
	if s.Description != "" {
		schema["description"] = s.Description
	}
	return &models.ResponseFormat{
		Type:       "json_schema",
		JSONSchema: schema,
	}
}

// ParseResponse unmarshals a ModelResponse.Content into the target struct.
// The target must be a non-nil pointer.
func ParseResponse(resp *types.ModelResponse, target interface{}) error {
	if resp == nil {
		return fmt.Errorf("response is nil")
	}
	if target == nil {
		return fmt.Errorf("target is nil")
	}
	rv := reflect.ValueOf(target)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return fmt.Errorf("target must be a non-nil pointer, got %T", target)
	}
	content := strings.TrimSpace(resp.Content)
	if content == "" {
		return fmt.Errorf("response content is empty")
	}
	if err := json.Unmarshal([]byte(content), target); err != nil {
		return fmt.Errorf("failed to parse structured output: %w", err)
	}
	return nil
}

var timeType = reflect.TypeOf(time.Time{})

func typeToSchemaGuarded(t reflect.Type, seen map[reflect.Type]bool) map[string]interface{} {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	// Handle time.Time as a date-time string.
	if t == timeType {
		return map[string]interface{}{"type": "string", "format": "date-time"}
	}

	switch t.Kind() {
	case reflect.Struct:
		if seen[t] {
			return map[string]interface{}{"type": "object"} // break cycle
		}
		seen[t] = true
		return structToSchemaGuarded(t, seen)
	case reflect.Slice, reflect.Array:
		return map[string]interface{}{
			"type":  "array",
			"items": typeToSchemaGuarded(t.Elem(), seen),
		}
	case reflect.Map:
		return map[string]interface{}{
			"type":                 "object",
			"additionalProperties": typeToSchemaGuarded(t.Elem(), seen),
		}
	case reflect.String:
		return map[string]interface{}{"type": "string"}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return map[string]interface{}{"type": "integer"}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return map[string]interface{}{"type": "integer"}
	case reflect.Float32, reflect.Float64:
		return map[string]interface{}{"type": "number"}
	case reflect.Bool:
		return map[string]interface{}{"type": "boolean"}
	case reflect.Interface:
		return map[string]interface{}{} // any value
	default:
		return map[string]interface{}{"type": "string"}
	}
}

func structToSchemaGuarded(t reflect.Type, seen map[reflect.Type]bool) map[string]interface{} {
	properties := make(map[string]interface{})
	var required []string

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if !field.IsExported() {
			continue
		}

		name := field.Name
		jsonTag := field.Tag.Get("json")
		if jsonTag != "" {
			parts := strings.Split(jsonTag, ",")
			if parts[0] == "-" {
				continue
			}
			if parts[0] != "" {
				name = parts[0]
			}
			// Fields with omitempty are optional
			isOmitempty := false
			for _, p := range parts[1:] {
				if p == "omitempty" {
					isOmitempty = true
				}
			}
			if !isOmitempty {
				required = append(required, name)
			}
		} else {
			required = append(required, name)
		}

		fieldSchema := typeToSchemaGuarded(field.Type, seen)

		// Add description from struct tag if present
		if desc := field.Tag.Get("description"); desc != "" {
			fieldSchema["description"] = desc
		}

		properties[name] = fieldSchema
	}

	schema := map[string]interface{}{
		"type":       "object",
		"properties": properties,
	}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}
