package agent

import "encoding/json"

// Role describes who produced a message in the conversation.
type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
)

// Message is the minimal chat message shape used by the agent.
type Message struct {
	Role    Role   `json:"role"`
	Content string `json:"content"`
}

// Tool describes one capability the agent can invoke.
// In a production LLM integration this metadata is usually sent to the model
// as function/tool schema.
type Tool struct {
	Name        string
	Description string
	Parameters  json.RawMessage
	Execute     func(args json.RawMessage) (string, error)
}

// ToolCall is one tool invocation requested by the model.
type ToolCall struct {
	Tool string          `json:"tool"`
	Args json.RawMessage `json:"args"`
}

// Action is the structured decision emitted by the model.
// It supports both the legacy single-tool fields (Tool/Args) and the newer
// multi-tool Calls field so the tutorial can demonstrate both patterns.
type Action struct {
	Thought string `json:"thought"`

	// Legacy single-tool call. Kept for backward compatibility and for simple
	// examples where one action only needs one tool.
	Tool string          `json:"tool,omitempty"`
	Args json.RawMessage `json:"args,omitempty"`

	// Calls lets one model step request multiple tool invocations. The Agent
	// executes them in order and appends one tool observation per call.
	Calls []ToolCall `json:"calls,omitempty"`

	Final string `json:"final,omitempty"`
}

// ToolCalls normalizes Action into a list of tool calls.
func (a Action) ToolCalls() []ToolCall {
	if len(a.Calls) > 0 {
		return a.Calls
	}
	if a.Tool != "" {
		return []ToolCall{{Tool: a.Tool, Args: a.Args}}
	}
	return nil
}
