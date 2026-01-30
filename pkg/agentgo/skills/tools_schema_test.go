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

	// get_skill_script should NOT exist (permanently disabled)
	if _, exists := functions["get_skill_script"]; exists {
		t.Fatal("get_skill_script should NOT be present (hardcoded disabled)")
	}

	// Verify that only instructions and reference are present
	if len(functions) != 2 {
		t.Errorf("Expected 2 functions, got %d", len(functions))
	}

	if _, exists := functions["get_skill_instructions"]; !exists {
		t.Error("get_skill_instructions should be present")
	}

	if _, exists := functions["get_skill_reference"]; !exists {
		t.Error("get_skill_reference should be present")
	}

	t.Logf("✅ Skills toolkit has only 2 tools (scripts permanently disabled)")
}

func TestAllSkillToolsHaveValidParameters(t *testing.T) {
	// Test only available tools (scripts permanently disabled)
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
		// get_skill_script is permanently disabled
	}

	for _, expectedFunc := range expectedFunctions {
		if _, exists := functions[expectedFunc]; !exists {
			t.Errorf("Expected function '%s' not found", expectedFunc)
		}
	}

	// Count should be exactly 2 (scripts permanently disabled)
	if len(functions) != 2 {
		t.Errorf("Expected 2 functions, got %d", len(functions))
	}

	// Ensure get_skill_script is NOT present
	if _, exists := functions["get_skill_script"]; exists {
		t.Error("get_skill_script should NOT be present (hardcoded disabled)")
	}
}
