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

func TestRuleBasedCanCallSameToolMultipleTimes(t *testing.T) {
	ag, err := agent.New(
		llm.NewRuleBased(),
		[]agent.Tool{tools.NewCalculator(), tools.NewMockWeather(), tools.NewTimeNow(nil)},
		4,
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	answer, transcript, err := ag.Run(context.Background(), "time，北京天气，计算5*7，计算9/3")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	for _, want := range []string{"time_now =>", "weather =>", "calculator => 35", "calculator => 3"} {
		if !strings.Contains(answer, want) {
			t.Fatalf("answer = %q, want it to contain %q", answer, want)
		}
	}

	var calculatorCount int
	var toolCount int
	for _, msg := range transcript {
		if msg.Role != agent.RoleTool {
			continue
		}
		toolCount++
		if strings.Contains(msg.Content, "calculator =>") {
			calculatorCount++
		}
	}
	if toolCount != 4 {
		t.Fatalf("toolCount = %d, want 4; transcript=%+v", toolCount, transcript)
	}
	if calculatorCount != 2 {
		t.Fatalf("calculatorCount = %d, want 2; transcript=%+v", calculatorCount, transcript)
	}
}

func TestRuleBasedCanCallWeatherMultipleTimes(t *testing.T) {
	ag, err := agent.New(
		llm.NewRuleBased(),
		[]agent.Tool{tools.NewCalculator(), tools.NewMockWeather(), tools.NewTimeNow(nil)},
		4,
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	answer, transcript, err := ag.Run(context.Background(), "time，北京天气，计算5*7，计算9/3，上海天气")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	for _, want := range []string{"time_now =>", "北京", "上海", "calculator => 35", "calculator => 3"} {
		if !strings.Contains(answer, want) {
			t.Fatalf("answer = %q, want it to contain %q", answer, want)
		}
	}

	var weatherCount int
	var calculatorCount int
	var toolCount int
	for _, msg := range transcript {
		if msg.Role != agent.RoleTool {
			continue
		}
		toolCount++
		if strings.Contains(msg.Content, "weather =>") {
			weatherCount++
		}
		if strings.Contains(msg.Content, "calculator =>") {
			calculatorCount++
		}
	}
	if toolCount != 5 {
		t.Fatalf("toolCount = %d, want 5; transcript=%+v", toolCount, transcript)
	}
	if weatherCount != 2 {
		t.Fatalf("weatherCount = %d, want 2; transcript=%+v", weatherCount, transcript)
	}
	if calculatorCount != 2 {
		t.Fatalf("calculatorCount = %d, want 2; transcript=%+v", calculatorCount, transcript)
	}
}

func TestRuleBasedCanRepeatEveryToolKind(t *testing.T) {
	notes := map[string]string{
		"agent": "Agent note",
		"tool":  "Tool note",
		"rag":   "RAG note",
	}

	tests := []struct {
		name           string
		question       string
		tools          []agent.Tool
		wantTool       string
		wantToolCount  int
		wantAnswerText []string
	}{
		{
			name:          "calculator twice",
			question:      "计算5*7，计算9/3",
			tools:         []agent.Tool{tools.NewCalculator()},
			wantTool:      "calculator =>",
			wantToolCount: 2,
			wantAnswerText: []string{
				"calculator => 35",
				"calculator => 3",
			},
		},
		{
			name:          "weather twice",
			question:      "北京天气，上海天气",
			tools:         []agent.Tool{tools.NewMockWeather()},
			wantTool:      "weather =>",
			wantToolCount: 2,
			wantAnswerText: []string{
				"北京",
				"上海",
			},
		},
		{
			name:          "time twice",
			question:      "time，UTC time",
			tools:         []agent.Tool{tools.NewTimeNow(nil)},
			wantTool:      "time_now =>",
			wantToolCount: 2,
			wantAnswerText: []string{
				"time_now =>",
			},
		},
		{
			name:          "todo twice",
			question:      "添加待办：A，添加待办：B",
			tools:         []agent.Tool{tools.NewTodoList(nil)},
			wantTool:      "todo_list =>",
			wantToolCount: 2,
			wantAnswerText: []string{
				"added todo #1: A",
				"added todo #2: B",
			},
		},
		{
			name:          "http twice",
			question:      "GET https://example.com，GET https://httpbin.org/get",
			tools:         []agent.Tool{tools.NewHTTPGet()},
			wantTool:      "http_get =>",
			wantToolCount: 2,
			wantAnswerText: []string{
				"https://example.com",
				"https://httpbin.org/get",
			},
		},
		{
			name:          "search notes three times",
			question:      "agent tool rag",
			tools:         []agent.Tool{tools.NewNotesSearch(notes)},
			wantTool:      "search_notes =>",
			wantToolCount: 3,
			wantAnswerText: []string{
				"Agent note",
				"Tool note",
				"RAG note",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ag, err := agent.New(llm.NewRuleBased(), tt.tools, 4)
			if err != nil {
				t.Fatalf("New() error = %v", err)
			}

			answer, transcript, err := ag.Run(context.Background(), tt.question)
			if err != nil {
				t.Fatalf("Run() error = %v", err)
			}

			for _, want := range tt.wantAnswerText {
				if !strings.Contains(answer, want) {
					t.Fatalf("answer = %q, want it to contain %q", answer, want)
				}
			}

			var count int
			for _, msg := range transcript {
				if msg.Role == agent.RoleTool && strings.Contains(msg.Content, tt.wantTool) {
					count++
				}
			}
			if count != tt.wantToolCount {
				t.Fatalf("%s count = %d, want %d; transcript=%+v", tt.wantTool, count, tt.wantToolCount, transcript)
			}
		})
	}
}
