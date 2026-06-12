package tools

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestTodoListAddListComplete(t *testing.T) {
	tool := NewTodoList(NewTodoStore())

	addArgs, _ := json.Marshal(map[string]any{"action": "add", "text": "学习 Agent"})
	added, err := tool.Execute(addArgs)
	if err != nil {
		t.Fatalf("add Execute() error = %v", err)
	}
	if !strings.Contains(added, "#1") {
		t.Fatalf("add Execute() = %q", added)
	}

	listArgs, _ := json.Marshal(map[string]any{"action": "list"})
	listed, err := tool.Execute(listArgs)
	if err != nil {
		t.Fatalf("list Execute() error = %v", err)
	}
	if !strings.Contains(listed, "[ ] #1 学习 Agent") {
		t.Fatalf("list Execute() = %q", listed)
	}

	completeArgs, _ := json.Marshal(map[string]any{"action": "complete", "id": 1})
	completed, err := tool.Execute(completeArgs)
	if err != nil {
		t.Fatalf("complete Execute() error = %v", err)
	}
	if !strings.Contains(completed, "completed todo #1") {
		t.Fatalf("complete Execute() = %q", completed)
	}

	listed, err = tool.Execute(listArgs)
	if err != nil {
		t.Fatalf("list Execute() after complete error = %v", err)
	}
	if !strings.Contains(listed, "[x] #1 学习 Agent") {
		t.Fatalf("list Execute() after complete = %q", listed)
	}
}
