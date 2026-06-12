package tools

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/zieckey/ai-study-by-codex/internal/agent"
)

// NewTimeNow returns a tool that reports the current time.
// The clock dependency makes the tool deterministic in tests.
func NewTimeNow(clock func() time.Time) agent.Tool {
	if clock == nil {
		clock = time.Now
	}

	return agent.Tool{
		Name:        "time_now",
		Description: "Get the current local time. Optional parameter: timezone, for example Asia/Shanghai or UTC.",
		Parameters:  json.RawMessage(`{"type":"object","properties":{"timezone":{"type":"string"}}}`),
		Execute: func(args json.RawMessage) (string, error) {
			var input struct {
				Timezone string `json:"timezone"`
			}
			if len(args) > 0 && string(args) != "null" {
				if err := json.Unmarshal(args, &input); err != nil {
					return "", fmt.Errorf("invalid time_now args: %w", err)
				}
			}

			now := clock()
			zoneName := input.Timezone
			if zoneName != "" {
				loc, err := time.LoadLocation(zoneName)
				if err != nil {
					return "", fmt.Errorf("unknown timezone %q", zoneName)
				}
				now = now.In(loc)
			}

			return now.Format(time.RFC3339), nil
		},
	}
}
