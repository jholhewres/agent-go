package agent

import (
	"context"
	"fmt"
	"sync"

	"github.com/jholhewres/agent-go/pkg/agentgo/run"
)

// SpawnConfig configures a sub-agent spawn.
type SpawnConfig struct {
	Agent *Agent
	Input string
}

// SpawnResult contains the result of a sub-agent execution.
type SpawnResult struct {
	AgentID string
	Output  *RunOutput
	Error   error
}

// Spawn runs a child agent with a linked RunContext.
// The child's RunContext.ParentRunID is set to the current RunID from ctx.
// Context cancellation propagates from parent to child.
// Returns an error if child is nil.
func Spawn(ctx context.Context, child *Agent, input string) (*RunOutput, error) {
	if child == nil {
		return nil, fmt.Errorf("Spawn: child agent must not be nil")
	}
	childCtx := buildChildContext(ctx)
	return child.Run(childCtx, input)
}

// SpawnAll runs multiple child agents concurrently and waits for all to complete.
// Results are returned in the same order as the input configs.
// Context cancellation propagates to all children.
func SpawnAll(ctx context.Context, children []SpawnConfig) []SpawnResult {
	results := make([]SpawnResult, len(children))
	var wg sync.WaitGroup

	for i, cfg := range children {
		wg.Add(1)
		go func(idx int, c SpawnConfig) {
			defer wg.Done()
			output, err := Spawn(ctx, c.Agent, c.Input)
			results[idx] = SpawnResult{
				AgentID: c.Agent.ID,
				Output:  output,
				Error:   err,
			}
		}(i, cfg)
	}

	wg.Wait()
	return results
}

func buildChildContext(parentCtx context.Context) context.Context {
	parentRC, ok := run.FromContext(parentCtx)

	childRC := run.NewContext()
	childRC.EnsureRunID()

	if ok && parentRC != nil {
		childRC.ParentRunID = parentRC.RunID
		childRC.SessionID = parentRC.SessionID
		childRC.UserID = parentRC.UserID
	}

	return run.WithContext(parentCtx, childRC)
}
