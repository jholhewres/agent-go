// Package agenttool wraps an Agent as a Toolkit, allowing one agent
// to be invoked as a tool by another agent.
package agenttool

import (
	"context"
	"fmt"
	"strings"
	"unicode"

	"github.com/jholhewres/agent-go/pkg/agentgo/agent"
	"github.com/jholhewres/agent-go/pkg/agentgo/tools/toolkit"
)

// AgentToolkit wraps an Agent as a single-function toolkit.
// The registered function is named "ask_<agent_name>" and accepts
// an "input" string parameter. When invoked, it spawns the wrapped
// agent using agent.Spawn and returns its output content.
type AgentToolkit struct {
	*toolkit.BaseToolkit
	agent       *agent.Agent
	description string
}

// sanitizeName converts a name into a safe identifier for function names.
func sanitizeName(name string) string {
	s := strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
			return unicode.ToLower(r)
		}
		return '_'
	}, name)
	if s == "" {
		s = "agent"
	}
	return s
}

// New creates a new AgentToolkit that wraps the given agent.
// The description is used as the tool function's description,
// helping the parent agent understand when to delegate to this agent.
// Panics if ag is nil.
func New(ag *agent.Agent, description string) *AgentToolkit {
	if ag == nil {
		panic("agenttool.New: agent must not be nil")
	}

	safeName := sanitizeName(ag.Name)
	funcName := fmt.Sprintf("ask_%s", safeName)

	t := &AgentToolkit{
		BaseToolkit: toolkit.NewBaseToolkit(fmt.Sprintf("agent_%s", safeName)),
		agent:       ag,
		description: description,
	}

	t.RegisterFunction(&toolkit.Function{
		Name:        funcName,
		Description: description,
		Parameters: map[string]toolkit.Parameter{
			"input": {
				Type:        "string",
				Description: "The input or question to send to the agent",
				Required:    true,
			},
		},
		Handler: t.handleAsk,
	})

	return t
}

func (t *AgentToolkit) handleAsk(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	input, ok := args["input"].(string)
	if !ok || input == "" {
		return nil, fmt.Errorf("input parameter is required and must be a non-empty string")
	}

	output, err := agent.Spawn(ctx, t.agent, input)
	if err != nil {
		return nil, fmt.Errorf("agent %s failed: %w", t.agent.Name, err)
	}

	return output.Content, nil
}
