package toolkit

import (
	"encoding/json"
	"testing"
)

func TestToModelToolDefinitions_ArrayParameterWithItems(t *testing.T) {
	// Create a toolkit with an array parameter
	tk := NewBaseToolkit("test-toolkit")
	
	tk.RegisterFunction(&Function{
		Name:        "test_function",
		Description: "Test function with array parameter",
		Parameters: map[string]Parameter{
			"items": {
				Type:        "array",
				Description: "List of items",
				Required:    true,
				Items: &Parameter{
					Type: "string",
				},
			},
		},
		Handler: nil,
	})

	// Convert to model tool definitions
	definitions := ToModelToolDefinitions([]Toolkit{tk})

	if len(definitions) != 1 {
		t.Fatalf("Expected 1 definition, got %d", len(definitions))
	}

	def := definitions[0]

	// Get parameters (already a map)
	params := def.Function.Parameters

	// Get properties
	properties, ok := params["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("Properties is not a map")
	}

	// Get items parameter
	itemsParam, ok := properties["items"].(map[string]interface{})
	if !ok {
		t.Fatal("items parameter not found")
	}

	// Check that items field exists
	itemsField, hasItems := itemsParam["items"]
	if !hasItems {
		t.Fatal("CRITICAL: 'items' field missing in array parameter - OpenAI will reject this schema!")
	}

	// Check items structure
	itemsMap, ok := itemsField.(map[string]interface{})
	if !ok {
		t.Fatalf("items field is not a map, got type %T", itemsField)
	}

	// Check items.type
	itemsType, ok := itemsMap["type"].(string)
	if !ok {
		t.Fatal("items.type not found")
	}

	if itemsType != "string" {
		t.Errorf("Expected items.type = 'string', got '%s'", itemsType)
	}

	// Serialize to JSON to verify OpenAI compatibility
	jsonBytes, err := json.MarshalIndent(def, "", "  ")
	if err != nil {
		t.Fatalf("Failed to serialize to JSON: %v", err)
	}

	t.Logf("✅ Generated OpenAI-compatible schema:\n%s", string(jsonBytes))

	// Parse back to verify structure
	var parsed map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &parsed); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	// Verify critical path: function.parameters.properties.items.items
	function := parsed["function"].(map[string]interface{})
	funcParams := function["parameters"].(map[string]interface{})
	funcProps := funcParams["properties"].(map[string]interface{})
	funcItems := funcProps["items"].(map[string]interface{})
	funcItemsItems, hasItemsItems := funcItems["items"]
	
	if !hasItemsItems {
		t.Fatal("CRITICAL: After JSON serialization, 'items.items' field is missing!")
	}

	t.Logf("✅ items.items field present after serialization: %+v", funcItemsItems)
}

func TestToModelToolDefinitions_SkillsGetScriptSchema(t *testing.T) {
	// Simulate the exact get_skill_script function
	tk := NewBaseToolkit("skills")
	
	tk.RegisterFunction(&Function{
		Name:        "get_skill_script",
		Description: "Read or execute a script from a skill",
		Parameters: map[string]Parameter{
			"skill_name": {
				Type:        "string",
				Description: "The name of the skill",
				Required:    true,
			},
			"script_path": {
				Type:        "string",
				Description: "The path to the script",
				Required:    true,
			},
			"execute": {
				Type:        "boolean",
				Description: "Whether to execute the script",
				Required:    false,
			},
			"args": {
				Type:        "array",
				Description: "Arguments to pass to the script when executing",
				Required:    false,
				Items: &Parameter{
					Type: "string",
				},
			},
			"timeout": {
				Type:        "integer",
				Description: "Timeout in seconds for script execution",
				Required:    false,
			},
		},
		Handler: nil,
	})

	definitions := ToModelToolDefinitions([]Toolkit{tk})

	if len(definitions) != 1 {
		t.Fatalf("Expected 1 definition, got %d", len(definitions))
	}

	def := definitions[0]

	// Serialize to JSON
	jsonBytes, err := json.MarshalIndent(def, "", "  ")
	if err != nil {
		t.Fatalf("Failed to serialize: %v", err)
	}

	t.Logf("✅ get_skill_script OpenAI schema:\n%s", string(jsonBytes))

	// Verify structure
	var parsed map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &parsed); err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	// Navigate to args parameter
	function := parsed["function"].(map[string]interface{})
	funcParams := function["parameters"].(map[string]interface{})
	funcProps := funcParams["properties"].(map[string]interface{})
	argsParam := funcProps["args"].(map[string]interface{})

	// Check args.type
	if argsParam["type"] != "array" {
		t.Errorf("args.type should be 'array', got '%v'", argsParam["type"])
	}

	// Check args.items (CRITICAL)
	argsItems, hasItems := argsParam["items"]
	if !hasItems {
		t.Fatal("CRITICAL: args parameter missing 'items' field - OpenAI will return HTTP 400!")
	}

	argsItemsMap := argsItems.(map[string]interface{})
	if argsItemsMap["type"] != "string" {
		t.Errorf("args.items.type should be 'string', got '%v'", argsItemsMap["type"])
	}

	t.Logf("✅ get_skill_script has valid OpenAI schema with args.items.type = %s", argsItemsMap["type"])
}

func TestToModelToolDefinitions_NonArrayParameter(t *testing.T) {
	// Test that non-array parameters don't get items field
	tk := NewBaseToolkit("test")
	
	tk.RegisterFunction(&Function{
		Name:        "test_func",
		Description: "Test",
		Parameters: map[string]Parameter{
			"name": {
				Type:        "string",
				Description: "A string parameter",
				Required:    true,
			},
		},
		Handler: nil,
	})

	definitions := ToModelToolDefinitions([]Toolkit{tk})
	def := definitions[0]

	params := def.Function.Parameters
	properties := params["properties"].(map[string]interface{})
	nameParam := properties["name"].(map[string]interface{})

	// Non-array parameters should NOT have items
	if _, hasItems := nameParam["items"]; hasItems {
		t.Error("Non-array parameter should not have 'items' field")
	}

	t.Log("✅ Non-array parameters correctly don't have 'items' field")
}
