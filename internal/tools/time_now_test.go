package tools

import (
	"encoding/json"
	"testing"
	"time"
)

func TestTimeNow(t *testing.T) {
	fixed := time.Date(2026, 6, 12, 10, 30, 0, 0, time.UTC)
	tool := NewTimeNow(func() time.Time { return fixed })

	args, _ := json.Marshal(map[string]string{"timezone": "UTC"})
	got, err := tool.Execute(args)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if got != "2026-06-12T10:30:00Z" {
		t.Fatalf("Execute() = %q", got)
	}
}
