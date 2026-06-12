package agent_test

import (
	"context"
	"encoding/json"
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

type scriptedModel struct {
	actions []agent.Action
	idx     int
}

func (m *scriptedModel) Complete(ctx context.Context, messages []agent.Message, tools []agent.Tool) (agent.Action, error) {
	if m.idx >= len(m.actions) {
		return agent.Action{Final: "done"}, nil
	}
	action := m.actions[m.idx]
	m.idx++
	return action, nil
}

func TestAgentExecutesMultipleToolCallsInOneStep(t *testing.T) {
	model := &scriptedModel{actions: []agent.Action{
		{
			Thought: "Need two tools.",
			Calls: []agent.ToolCall{
				{Tool: "first", Args: []byte(`{"value":"a"}`)},
				{Tool: "second", Args: []byte(`{"value":"b"}`)},
			},
		},
		{Thought: "Observed both tools.", Final: "done"},
	}}

	ag, err := agent.New(model, []agent.Tool{
		{
			Name: "first",
			Execute: func(args json.RawMessage) (string, error) {
				return "first result", nil
			},
		},
		{
			Name: "second",
			Execute: func(args json.RawMessage) (string, error) {
				return "second result", nil
			},
		},
	}, 4)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	answer, transcript, err := ag.Run(context.Background(), "call two tools")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if answer != "done" {
		t.Fatalf("answer = %q, want done", answer)
	}

	var observations []string
	for _, msg := range transcript {
		if msg.Role == agent.RoleTool {
			observations = append(observations, msg.Content)
		}
	}
	if len(observations) != 2 {
		t.Fatalf("tool observation count = %d, want 2; transcript=%+v", len(observations), transcript)
	}
	if !strings.Contains(observations[0], "first => first result") || !strings.Contains(observations[1], "second => second result") {
		t.Fatalf("unexpected observations: %+v", observations)
	}
}

func TestRuleBasedCanPlanMultipleToolCalls(t *testing.T) {
	tests := []struct {
		name     string
		question string
	}{
		{name: "explicit conjunction", question: "同时告诉我上海天气和现在几点？"},
		{name: "implicit comma separated intents", question: "time，北京天气"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ag, err := agent.New(
				llm.NewRuleBased(),
				[]agent.Tool{tools.NewMockWeather(), tools.NewTimeNow(nil)},
				4,
			)
			if err != nil {
				t.Fatalf("New() error = %v", err)
			}

			answer, transcript, err := ag.Run(context.Background(), tt.question)
			if err != nil {
				t.Fatalf("Run() error = %v", err)
			}
			if !strings.Contains(answer, "多工具调用结果") || !strings.Contains(answer, "weather =>") || !strings.Contains(answer, "time_now =>") {
				t.Fatalf("answer = %q, want multi-tool summary", answer)
			}

			var toolCount int
			for _, msg := range transcript {
				if msg.Role == agent.RoleTool {
					toolCount++
				}
			}
			if toolCount != 2 {
				t.Fatalf("toolCount = %d, want 2; transcript=%+v", toolCount, transcript)
			}
		})
	}
}
