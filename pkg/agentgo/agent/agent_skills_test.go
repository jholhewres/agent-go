package agent

import (
	"context"
	"strings"
	"testing"

	"github.com/jholhewres/agent-go/pkg/agentgo/models"
	"github.com/jholhewres/agent-go/pkg/agentgo/skills"
	"github.com/jholhewres/agent-go/pkg/agentgo/tools/toolkit"
	"github.com/jholhewres/agent-go/pkg/agentgo/types"
)

// mockSkillLoader for testing
type mockSkillLoader struct {
	skillsToLoad []*skills.Skill
}

func (m *mockSkillLoader) Load() ([]*skills.Skill, error) {
	return m.skillsToLoad, nil
}

func (m *mockSkillLoader) GetType() string {
	return "mock"
}

// mockModelForSkills for testing
type mockModelForSkills struct{}

func (m *mockModelForSkills) GetID() string       { return "mock-model" }
func (m *mockModelForSkills) GetName() string     { return "Mock Model" }
func (m *mockModelForSkills) GetProvider() string { return "mock" }
func (m *mockModelForSkills) Invoke(ctx context.Context, req *models.InvokeRequest) (*types.ModelResponse, error) {
	return &types.ModelResponse{
		Content: "Mock response",
	}, nil
}
func (m *mockModelForSkills) InvokeStream(ctx context.Context, req *models.InvokeRequest) (<-chan types.ResponseChunk, error) {
	ch := make(chan types.ResponseChunk)
	close(ch)
	return ch, nil
}

func TestAgentWithSkills(t *testing.T) {
	// Create mock skill
	mockLoader := &mockSkillLoader{
		skillsToLoad: []*skills.Skill{
			{
				Name:         "test-skill",
				Description:  "A test skill for validation",
				Instructions: "Detailed test instructions for the skill.\nMultiple lines of guidance.",
			},
		},
	}

	// Create Skills orchestrator
	agentSkills, err := skills.NewSkills([]skills.SkillLoader{mockLoader})
	if err != nil {
		t.Fatalf("Failed to create skills: %v", err)
	}

	// Create agent with skills
	agent, err := New(Config{
		Model:        &mockModelForSkills{},
		Instructions: "You are a helpful assistant",
		Skills:       agentSkills,
	})
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	// Test 1: Verify skills snippet is in instructions
	if !strings.Contains(agent.Instructions, "test-skill") {
		t.Error("Agent instructions should contain skill name")
	}
	if !strings.Contains(agent.Instructions, "A test skill for validation") {
		t.Error("Agent instructions should contain skill description")
	}
	if !strings.Contains(agent.Instructions, "get_skill_instructions") {
		t.Error("Agent instructions should mention get_skill_instructions tool")
	}

	// Test 2: Verify base instructions are preserved
	if !strings.Contains(agent.Instructions, "You are a helpful assistant") {
		t.Error("Agent instructions should contain base instructions")
	}

	// Test 3: Verify skill tools are registered
	foundGetInstructions := false
	foundGetReference := false
	foundGetScript := false

	for _, tk := range agent.Toolkits {
		funcs := tk.Functions()
		for name := range funcs {
			switch name {
			case "get_skill_instructions":
				foundGetInstructions = true
			case "get_skill_reference":
				foundGetReference = true
			case "get_skill_script":
				foundGetScript = true
			}
		}
	}

	if !foundGetInstructions {
		t.Error("Agent should have get_skill_instructions tool")
	}
	if !foundGetReference {
		t.Error("Agent should have get_skill_reference tool")
	}
	if !foundGetScript {
		t.Error("Agent should have get_skill_script tool")
	}

	// Test 4: Verify toolkit count
	expectedToolkits := 1 // Skills toolkit only
	if len(agent.Toolkits) != expectedToolkits {
		t.Errorf("Expected %d toolkit, got %d", expectedToolkits, len(agent.Toolkits))
	}
}

func TestAgentWithoutSkills(t *testing.T) {
	// Create agent without skills (backward compatibility test)
	agent, err := New(Config{
		Model:        &mockModelForSkills{},
		Instructions: "You are a helpful assistant",
		Skills:       nil, // No skills
	})
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	// Verify no skills snippet
	if strings.Contains(agent.Instructions, "You have access to the following skills") {
		t.Error("Agent without skills should not have skills snippet")
	}

	// Verify no skills toolkit
	for _, tk := range agent.Toolkits {
		if tk.Name() == "skills" {
			t.Error("Agent without skills should not have skills toolkit")
		}
	}

	// Verify base instructions are preserved
	if agent.Instructions != "You are a helpful assistant" {
		t.Errorf("Expected instructions to be unchanged, got: %s", agent.Instructions)
	}

	// Verify no toolkits
	if len(agent.Toolkits) != 0 {
		t.Errorf("Expected 0 toolkits, got %d", len(agent.Toolkits))
	}
}

func TestAgentSkillsToolkitPrepended(t *testing.T) {
	// Create mock skill
	mockLoader := &mockSkillLoader{
		skillsToLoad: []*skills.Skill{
			{
				Name:         "prepend-test",
				Description:  "Test skill",
				Instructions: "Instructions",
			},
		},
	}

	agentSkills, err := skills.NewSkills([]skills.SkillLoader{mockLoader})
	if err != nil {
		t.Fatalf("Failed to create skills: %v", err)
	}

	// Create mock user toolkit
	userToolkit := &mockToolkit{name: "user-toolkit"}

	// Create agent with both skills and user toolkit
	agent, err := New(Config{
		Model:        &mockModelForSkills{},
		Instructions: "Base",
		Skills:       agentSkills,
		Toolkits:     []toolkit.Toolkit{userToolkit},
	})
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	// Verify toolkit order: skills first, then user toolkits
	if len(agent.Toolkits) != 2 {
		t.Errorf("Expected 2 toolkits, got %d", len(agent.Toolkits))
	}

	if agent.Toolkits[0].Name() != "skills" {
		t.Errorf("First toolkit should be 'skills', got '%s'", agent.Toolkits[0].Name())
	}

	if agent.Toolkits[1].Name() != "user-toolkit" {
		t.Errorf("Second toolkit should be 'user-toolkit', got '%s'", agent.Toolkits[1].Name())
	}
}

func TestAgentSkillsWithMultipleSkills(t *testing.T) {
	// Create mock loader with multiple skills
	mockLoader := &mockSkillLoader{
		skillsToLoad: []*skills.Skill{
			{
				Name:         "skill-one",
				Description:  "First skill",
				Instructions: "Instructions for skill one",
			},
			{
				Name:         "skill-two",
				Description:  "Second skill",
				Instructions: "Instructions for skill two",
			},
		},
	}

	agentSkills, err := skills.NewSkills([]skills.SkillLoader{mockLoader})
	if err != nil {
		t.Fatalf("Failed to create skills: %v", err)
	}

	// Create agent
	agent, err := New(Config{
		Model:        &mockModelForSkills{},
		Instructions: "Base instructions",
		Skills:       agentSkills,
	})
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	// Verify both skills are in instructions
	if !strings.Contains(agent.Instructions, "skill-one") {
		t.Error("Instructions should contain skill-one")
	}
	if !strings.Contains(agent.Instructions, "skill-two") {
		t.Error("Instructions should contain skill-two")
	}
	if !strings.Contains(agent.Instructions, "First skill") {
		t.Error("Instructions should contain skill-one description")
	}
	if !strings.Contains(agent.Instructions, "Second skill") {
		t.Error("Instructions should contain skill-two description")
	}

	// Verify skills toolkit is present
	hasSkillsToolkit := false
	for _, tk := range agent.Toolkits {
		if tk.Name() == "skills" {
			hasSkillsToolkit = true
			// Should have 3 functions
			funcs := tk.Functions()
			if len(funcs) != 3 {
				t.Errorf("Skills toolkit should have 3 functions, got %d", len(funcs))
			}
			break
		}
	}

	if !hasSkillsToolkit {
		t.Error("Agent should have skills toolkit")
	}
}

// mockToolkit for testing toolkit ordering
type mockToolkit struct {
	name string
}

func (m *mockToolkit) Name() string {
	return m.name
}

func (m *mockToolkit) Functions() map[string]*toolkit.Function {
	return map[string]*toolkit.Function{}
}

func (m *mockToolkit) Execute(ctx context.Context, functionName string, args map[string]interface{}) (interface{}, error) {
	return nil, nil
}
