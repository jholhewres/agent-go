package agent

import (
	"context"
	"testing"

	"github.com/jholhewres/agent-go/pkg/agentgo/models"
	"github.com/jholhewres/agent-go/pkg/agentgo/skills"
	"github.com/jholhewres/agent-go/pkg/agentgo/types"
)

// Mock model for testing
type mockModelForScripts struct {
	models.BaseModel
	invokeFunc func(ctx context.Context, req *models.InvokeRequest) (*types.ModelResponse, error)
}

func (m *mockModelForScripts) Invoke(ctx context.Context, req *models.InvokeRequest) (*types.ModelResponse, error) {
	if m.invokeFunc != nil {
		return m.invokeFunc(ctx, req)
	}
	return &types.ModelResponse{Content: "test response"}, nil
}

func (m *mockModelForScripts) InvokeStream(ctx context.Context, req *models.InvokeRequest) (<-chan types.ResponseChunk, error) {
	ch := make(chan types.ResponseChunk)
	close(ch)
	return ch, nil
}

func (m *mockModelForScripts) GetProvider() string { return "test" }
func (m *mockModelForScripts) GetID() string       { return "test-model" }
func (m *mockModelForScripts) GetName() string     { return "test-model" }

// Mock skill loader
type mockSkillLoaderForTest struct {
	skills []*skills.Skill
}

func (l *mockSkillLoaderForTest) Load() ([]*skills.Skill, error) {
	return l.skills, nil
}

func (l *mockSkillLoaderForTest) GetType() string {
	return "test"
}

func TestAgentWithDisabledScripts(t *testing.T) {
	model := &mockModelForScripts{}

	// Create skills
	mockLoader := &mockSkillLoaderForTest{
		skills: []*skills.Skill{
			{
				Name:         "test-skill",
				Description:  "Test skill",
				Instructions: "Test instructions",
			},
		},
	}

	agentSkills, err := skills.NewSkills([]skills.SkillLoader{mockLoader})
	if err != nil {
		t.Fatalf("Failed to create skills: %v", err)
	}

	// Create agent (scripts always disabled, no flag needed)
	agent, err := New(Config{
		Name:   "test-agent-no-scripts",
		Model:  model,
		Skills: agentSkills,
	})
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	// Verify agent was created
	if agent == nil {
		t.Fatal("Agent should not be nil")
	}

	// Verify skills toolkit is registered
	if len(agent.Toolkits) == 0 {
		t.Fatal("Agent should have at least one toolkit (skills)")
	}

	// Get functions from skills toolkit
	skillsToolkit := agent.Toolkits[0]
	functions := skillsToolkit.Functions()

	// Should have 2 functions only (instructions + reference, NO script)
	// Scripts are ALWAYS disabled (hardcoded)
	if len(functions) != 2 {
		t.Errorf("Expected 2 functions (scripts permanently disabled), got %d", len(functions))
	}

	// Verify get_skill_instructions is present
	if _, exists := functions["get_skill_instructions"]; !exists {
		t.Error("get_skill_instructions should always be present")
	}

	// Verify get_skill_reference is present
	if _, exists := functions["get_skill_reference"]; !exists {
		t.Error("get_skill_reference should always be present")
	}

	// Verify get_skill_script is NOT present (permanently disabled)
	if _, exists := functions["get_skill_script"]; exists {
		t.Error("get_skill_script should NEVER be present (hardcoded disabled)")
	}
}

func TestAgentWithEnabledScripts(t *testing.T) {
	model := &mockModelForScripts{}

	// Create skills
	mockLoader := &mockSkillLoaderForTest{
		skills: []*skills.Skill{
			{
				Name:         "test-skill",
				Description:  "Test skill",
				Instructions: "Test instructions",
			},
		},
	}

	agentSkills, err := skills.NewSkills([]skills.SkillLoader{mockLoader})
	if err != nil {
		t.Fatalf("Failed to create skills: %v", err)
	}

	// Create agent WITHOUT DisableSkillScripts (default: false, scripts enabled)
	agent, err := New(Config{
		Name:   "test-agent-with-scripts",
		Model:  model,
		Skills: agentSkills,
		// DisableSkillScripts: false (default)
	})
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	// Verify agent was created
	if agent == nil {
		t.Fatal("Agent should not be nil")
	}

	// Verify skills toolkit is registered
	if len(agent.Toolkits) == 0 {
		t.Fatal("Agent should have at least one toolkit (skills)")
	}

	// Get functions from skills toolkit
	skillsToolkit := agent.Toolkits[0]
	functions := skillsToolkit.Functions()

	// Should have 3 functions (instructions + reference + script)
	if len(functions) != 3 {
		t.Errorf("Expected 3 functions (scripts enabled by default), got %d", len(functions))
	}

	// Verify all 3 functions are present
	if _, exists := functions["get_skill_instructions"]; !exists {
		t.Error("get_skill_instructions should be present")
	}

	if _, exists := functions["get_skill_reference"]; !exists {
		t.Error("get_skill_reference should be present")
	}

	if _, exists := functions["get_skill_script"]; !exists {
		t.Error("get_skill_script should be present when scripts are enabled (default)")
	}
}

func TestAgentWithoutSkillsUnaffected(t *testing.T) {
	model := &mockModelForScripts{}

	// Create agent WITHOUT skills
	agent, err := New(Config{
		Name:  "test-agent-no-skills",
		Model: model,
		// No Skills provided
	})
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	// Verify agent was created
	if agent == nil {
		t.Fatal("Agent should not be nil")
	}

	// No toolkits should be registered (no skills provided)
	if len(agent.Toolkits) != 0 {
		t.Errorf("Expected 0 toolkits (no skills provided), got %d", len(agent.Toolkits))
	}
}
