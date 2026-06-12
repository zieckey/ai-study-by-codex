package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// Model is implemented by internal/llm.Client. Keeping this interface here
// avoids an import cycle and makes the agent easy to test.
type Model interface {
	Complete(ctx context.Context, messages []Message, tools []Tool) (Action, error)
}

// Agent owns the control loop: think -> act -> observe -> repeat -> final.
type Agent struct {
	model    Model
	tools    map[string]Tool
	toolList []Tool
	maxSteps int
}

func New(model Model, tools []Tool, maxSteps int) (*Agent, error) {
	if model == nil {
		return nil, errors.New("model is nil")
	}
	if maxSteps <= 0 {
		maxSteps = 6
	}

	toolMap := make(map[string]Tool, len(tools))
	for _, tool := range tools {
		if strings.TrimSpace(tool.Name) == "" {
			return nil, errors.New("tool name is empty")
		}
		if tool.Execute == nil {
			return nil, fmt.Errorf("tool %q has nil Execute", tool.Name)
		}
		if _, exists := toolMap[tool.Name]; exists {
			return nil, fmt.Errorf("duplicated tool name %q", tool.Name)
		}
		toolMap[tool.Name] = tool
	}

	return &Agent{model: model, tools: toolMap, toolList: tools, maxSteps: maxSteps}, nil
}

// Run answers a user question. The returned transcript is useful for learning,
// debugging, tracing, and writing tests.
func (a *Agent) Run(ctx context.Context, question string) (answer string, transcript []Message, err error) {
	messages := []Message{
		{
			Role: RoleSystem,
			Content: strings.TrimSpace(`You are a small educational AI Agent.
You can either return a final answer or call one or more tools.
When calling one tool, return JSON with fields: thought, tool, args.
When calling multiple tools in the same step, return JSON with fields: thought, calls, where calls is an array of {tool,args}.
When done, return JSON with fields: thought, final.`),
		},
		{Role: RoleUser, Content: question},
	}

	for step := 1; step <= a.maxSteps; step++ {
		action, err := a.model.Complete(ctx, messages, a.toolList)
		if err != nil {
			return "", messages, fmt.Errorf("model step %d: %w", step, err)
		}

		assistantContent := mustJSON(action)
		messages = append(messages, Message{Role: RoleAssistant, Content: assistantContent})

		if strings.TrimSpace(action.Final) != "" {
			return action.Final, messages, nil
		}

		calls := action.ToolCalls()
		if len(calls) == 0 {
			return "", messages, fmt.Errorf("model step %d returned neither final nor tool calls", step)
		}

		for _, call := range calls {
			toolName := strings.TrimSpace(call.Tool)
			if toolName == "" {
				messages = append(messages, Message{Role: RoleTool, Content: "ERROR: empty tool name"})
				continue
			}

			tool, ok := a.tools[toolName]
			if !ok {
				observation := fmt.Sprintf("ERROR: unknown tool %q", toolName)
				messages = append(messages, Message{Role: RoleTool, Content: fmt.Sprintf("%s => %s", toolName, observation)})
				continue
			}

			observation, err := tool.Execute(call.Args)
			if err != nil {
				observation = "ERROR: " + err.Error()
			}
			messages = append(messages, Message{Role: RoleTool, Content: fmt.Sprintf("%s => %s", toolName, observation)})
		}
	}

	return "", messages, fmt.Errorf("agent reached max steps %d without final answer", a.maxSteps)
}

func mustJSON(v any) string {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Sprintf("%+v", v)
	}
	return string(b)
}
