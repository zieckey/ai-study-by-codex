package tools

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestMockWeather(t *testing.T) {
	tool := NewMockWeather()
	args, _ := json.Marshal(map[string]string{"city": "上海"})

	got, err := tool.Execute(args)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !strings.Contains(got, "上海") || !strings.Contains(got, "mock") {
		t.Fatalf("Execute() = %q, want Shanghai mock weather", got)
	}
}
