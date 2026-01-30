package skills

import (
	"testing"
)

func TestDisableScripts(t *testing.T) {
	mockSkill := &Skill{
		Name:         "test-skill",
		Description:  "Test skill for script disabling",
		Instructions: "Test instructions",
	}

	skills := &Skills{
		skills: map[string]*Skill{"test-skill": mockSkill},
	}

	// Scripts should ALWAYS be disabled (hardcoded)
	if skills.ScriptsEnabled() {
		t.Error("Scripts should be permanently disabled (hardcoded)")
	}

	// These methods should be no-ops but not crash
	skills.DisableScripts()
	skills.EnableScripts()

	// Scripts should STILL be disabled (hardcoded)
	if skills.ScriptsEnabled() {
		t.Error("Scripts should remain permanently disabled even after Enable/Disable calls")
	}
}

func TestAsToolkitWithScriptsDisabled(t *testing.T) {
	mockSkill := &Skill{
		Name:         "test-skill",
		Description:  "Test skill",
		Instructions: "Test instructions",
	}

	skills := &Skills{
		skills: map[string]*Skill{"test-skill": mockSkill},
	}

	// Create toolkit (scripts always disabled)
	skillTools := NewSkillTools(skills)
	tk := skillTools.AsToolkit()

	// Count functions (should be 2: instructions, reference only)
	functions := tk.Functions()
	if len(functions) != 2 {
		t.Errorf("Expected 2 functions (scripts permanently disabled), got %d", len(functions))
	}

	// Verify get_skill_script is NEVER present
	if _, exists := functions["get_skill_script"]; exists {
		t.Error("get_skill_script should NEVER be present (hardcoded disabled)")
	}

	// Verify instructions and reference are present
	if _, exists := functions["get_skill_instructions"]; !exists {
		t.Error("get_skill_instructions should always be present")
	}

	if _, exists := functions["get_skill_reference"]; !exists {
		t.Error("get_skill_reference should always be present")
	}
}

// Mock loader for testing
type testSkillLoader struct {
	skills []*Skill
}

func (l *testSkillLoader) Load() ([]*Skill, error) {
	return l.skills, nil
}

func (l *testSkillLoader) GetType() string {
	return "test"
}

func TestNewSkillsDefaultScriptState(t *testing.T) {
	mockLoader := &testSkillLoader{
		skills: []*Skill{
			{Name: "test", Description: "test", Instructions: "test"},
		},
	}

	skills, err := NewSkills([]SkillLoader{mockLoader})
	if err != nil {
		t.Fatalf("Failed to create skills: %v", err)
	}

	// Scripts should ALWAYS be disabled (hardcoded)
	if skills.ScriptsEnabled() {
		t.Error("Scripts should be permanently disabled by default (hardcoded)")
	}
}
