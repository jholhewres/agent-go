package skills

import (
	"testing"
)

func TestSkillToolsSchemaValid(t *testing.T) {
	// Create mock skills
	mockSkill := &Skill{
		Name:         "test-skill",
		Description:  "Test skill for schema validation",
		Instructions: "Test instructions",
	}

	skills := &Skills{
		skills: map[string]*Skill{
			"test-skill": mockSkill,
		},
	}

	// Create SkillTools and convert to toolkit
	skillTools := NewSkillTools(skills)
	tk := skillTools.AsToolkit()

	// Get all functions
	functions := tk.Functions()

	// Validate each function's parameters
	for funcName, fn := range functions {
		t.Run(funcName, func(t *testing.T) {
			for paramName, param := range fn.Parameters {
				// If parameter is array type, it MUST have Items defined
				if param.Type == "array" {
					if param.Items == nil {
						t.Errorf("Function %s: parameter %s is type 'array' but missing 'items' field (OpenAI will reject this schema)", 
							funcName, paramName)
					} else if param.Items.Type == "" {
						t.Errorf("Function %s: parameter %s has 'items' but items.type is empty", 
							funcName, paramName)
					}
				}
			}
		})
	}
}

func TestGetSkillScriptParametersValid(t *testing.T) {
	// Specifically test get_skill_script
	mockSkill := &Skill{
		Name:         "test",
		Description:  "Test",
		Instructions: "Test",
	}

	skills := &Skills{
		skills: map[string]*Skill{
			"test": mockSkill,
		},
	}

	skillTools := NewSkillTools(skills)
	tk := skillTools.AsToolkit()
	functions := tk.Functions()

	getSkillScript, exists := functions["get_skill_script"]
	if !exists {
		t.Fatal("get_skill_script function not found")
	}

	// Check that 'args' parameter has Items
	argsParam, exists := getSkillScript.Parameters["args"]
	if !exists {
		t.Fatal("'args' parameter not found in get_skill_script")
	}

	if argsParam.Type != "array" {
		t.Errorf("'args' parameter should be type 'array', got '%s'", argsParam.Type)
	}

	if argsParam.Items == nil {
		t.Fatal("'args' parameter is type 'array' but 'Items' is nil - OpenAI will reject this schema")
	}

	if argsParam.Items.Type != "string" {
		t.Errorf("'args' parameter Items.Type should be 'string', got '%s'", argsParam.Items.Type)
	}

	t.Logf("✅ get_skill_script schema is valid: args.items.type = %s", argsParam.Items.Type)
}

func TestAllSkillToolsHaveValidParameters(t *testing.T) {
	// Test all three tools
	mockSkill := &Skill{
		Name:         "validation-test",
		Description:  "Test skill",
		Instructions: "Instructions",
	}

	skills := &Skills{
		skills: map[string]*Skill{
			"validation-test": mockSkill,
		},
	}

	skillTools := NewSkillTools(skills)
	tk := skillTools.AsToolkit()
	functions := tk.Functions()

	expectedFunctions := []string{
		"get_skill_instructions",
		"get_skill_reference",
		"get_skill_script",
	}

	for _, expectedFunc := range expectedFunctions {
		if _, exists := functions[expectedFunc]; !exists {
			t.Errorf("Expected function '%s' not found", expectedFunc)
		}
	}

	// Count should be exactly 3
	if len(functions) != 3 {
		t.Errorf("Expected 3 functions, got %d", len(functions))
	}
}
