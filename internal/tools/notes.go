package tools

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/zieckey/ai-study-by-codex/internal/agent"
)

// NewNotesSearch returns a tiny retrieval tool over in-memory notes.
// It demonstrates the RAG idea: the agent can fetch external context before answering.
func NewNotesSearch(notes map[string]string) agent.Tool {
	return agent.Tool{
		Name:        "search_notes",
		Description: "Search local study notes by keyword and return relevant snippets.",
		Parameters:  json.RawMessage(`{"type":"object","properties":{"query":{"type":"string"}},"required":["query"]}`),
		Execute: func(args json.RawMessage) (string, error) {
			var input struct {
				Query string `json:"query"`
			}
			if err := json.Unmarshal(args, &input); err != nil {
				return "", fmt.Errorf("invalid search_notes args: %w", err)
			}

			query := strings.ToLower(strings.TrimSpace(input.Query))
			if query == "" {
				return "", fmt.Errorf("query is empty")
			}

			var matches []string
			for title, body := range notes {
				text := strings.ToLower(title + " " + body)
				if strings.Contains(text, query) {
					matches = append(matches, fmt.Sprintf("[%s] %s", title, body))
				}
			}
			if len(matches) == 0 {
				return "no matching notes", nil
			}
			return strings.Join(matches, "\n"), nil
		},
	}
}
