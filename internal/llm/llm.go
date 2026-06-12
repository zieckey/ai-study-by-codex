package llm

import (
	"context"

	"github.com/zieckey/ai-study-by-codex/internal/agent"
)

// Client is the abstraction between the Agent and a model provider.
// You can replace the RuleBased implementation with OpenAI, Ollama, etc.
type Client interface {
	Complete(ctx context.Context, messages []agent.Message, tools []agent.Tool) (agent.Action, error)
}
