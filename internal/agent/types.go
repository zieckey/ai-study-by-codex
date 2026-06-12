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

// Action is the structured decision emitted by the model.
// It is intentionally simple so that the agent loop is easy to study.
type Action struct {
	Thought string          `json:"thought"`
	Tool    string          `json:"tool"`
	Args    json.RawMessage `json:"args"`
	Final   string          `json:"final"`
}
