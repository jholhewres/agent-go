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
		skills:        map[string]*Skill{"test-skill": mockSkill},
		enableScripts: true, // Default
	}

	// Check default state
	if !skills.ScriptsEnabled() {
		t.Error("Scripts should be enabled by default")
	}

	// Disable scripts
	skills.DisableScripts()

	// Verify scripts are disabled
	if skills.ScriptsEnabled() {
		t.Error("Scripts should be disabled after calling DisableScripts()")
	}

	// Re-enable scripts
	skills.EnableScripts()

	// Verify scripts are enabled again
	if !skills.ScriptsEnabled() {
		t.Error("Scripts should be enabled after calling EnableScripts()")
	}
}

func TestAsToolkitWithScriptsDisabled(t *testing.T) {
	mockSkill := &Skill{
		Name:         "test-skill",
		Description:  "Test skill",
		Instructions: "Test instructions",
	}

	skills := &Skills{
		skills:        map[string]*Skill{"test-skill": mockSkill},
		enableScripts: true,
	}

	// Create toolkit with scripts enabled
	skillTools := NewSkillTools(skills)
	tkWithScripts := skillTools.AsToolkit()

	// Count functions (should be 3: instructions, reference, script)
	functionsWithScripts := tkWithScripts.Functions()
	if len(functionsWithScripts) != 3 {
		t.Errorf("Expected 3 functions with scripts enabled, got %d", len(functionsWithScripts))
	}

	// Verify get_skill_script is present
	if _, exists := functionsWithScripts["get_skill_script"]; !exists {
		t.Error("get_skill_script should be present when scripts are enabled")
	}

	// Now disable scripts
	skills.DisableScripts()

	// Create toolkit with scripts disabled
	skillToolsNoScripts := NewSkillTools(skills)
	tkWithoutScripts := skillToolsNoScripts.AsToolkit()

	// Count functions (should be 2: instructions, reference only)
	functionsWithoutScripts := tkWithoutScripts.Functions()
	if len(functionsWithoutScripts) != 2 {
		t.Errorf("Expected 2 functions with scripts disabled, got %d", len(functionsWithoutScripts))
	}

	// Verify get_skill_script is NOT present
	if _, exists := functionsWithoutScripts["get_skill_script"]; exists {
		t.Error("get_skill_script should NOT be present when scripts are disabled")
	}

	// Verify instructions and reference are still present
	if _, exists := functionsWithoutScripts["get_skill_instructions"]; !exists {
		t.Error("get_skill_instructions should always be present")
	}

	if _, exists := functionsWithoutScripts["get_skill_reference"]; !exists {
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

	// By default, scripts should be enabled
	if !skills.ScriptsEnabled() {
		t.Error("Scripts should be enabled by default when creating new Skills")
	}
}
