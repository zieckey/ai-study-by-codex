package agent_test

import (
	"context"
	"strings"
	"testing"

	"github.com/zieckey/ai-study-by-codex/internal/agent"
	"github.com/zieckey/ai-study-by-codex/internal/llm"
	"github.com/zieckey/ai-study-by-codex/internal/tools"
)

func TestAgentUsesCalculator(t *testing.T) {
	ag, err := agent.New(llm.NewRuleBased(), []agent.Tool{tools.NewCalculator()}, 4)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	answer, transcript, err := ag.Run(context.Background(), "23*7 等于多少？")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !strings.Contains(answer, "161") {
		t.Fatalf("answer = %q, want it to contain 161", answer)
	}

	var usedCalculator bool
	for _, msg := range transcript {
		if msg.Role == agent.RoleTool && strings.Contains(msg.Content, "calculator =>") {
			usedCalculator = true
		}
	}
	if !usedCalculator {
		t.Fatalf("transcript did not include calculator observation: %+v", transcript)
	}
}
