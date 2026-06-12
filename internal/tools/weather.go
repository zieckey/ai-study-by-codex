package tools

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/zieckey/ai-study-by-codex/internal/agent"
)

// NewMockWeather returns mocked weather data for learning tool calls without
// depending on a real weather API.
func NewMockWeather() agent.Tool {
	mock := map[string]string{
		"北京":        "北京：晴，26°C，东北风 2 级，湿度 38%。mock 数据，仅用于学习。",
		"上海":        "上海：多云，28°C，东南风 3 级，湿度 62%。mock 数据，仅用于学习。",
		"深圳":        "深圳：阵雨，30°C，南风 3 级，湿度 78%。mock 数据，仅用于学习。",
		"杭州":        "杭州：小雨，25°C，北风 2 级，湿度 70%。mock 数据，仅用于学习。",
		"guangzhou": "Guangzhou: cloudy, 31°C, south wind, humidity 75%. mock data for learning.",
		"beijing":   "Beijing: sunny, 26°C, northeast wind, humidity 38%. mock data for learning.",
		"shanghai":  "Shanghai: cloudy, 28°C, southeast wind, humidity 62%. mock data for learning.",
		"shenzhen":  "Shenzhen: showers, 30°C, south wind, humidity 78%. mock data for learning.",
	}

	return agent.Tool{
		Name:        "weather",
		Description: "Query mocked city weather. Parameter: city.",
		Parameters:  json.RawMessage(`{"type":"object","properties":{"city":{"type":"string"}},"required":["city"]}`),
		Execute: func(args json.RawMessage) (string, error) {
			var input struct {
				City string `json:"city"`
			}
			if err := json.Unmarshal(args, &input); err != nil {
				return "", fmt.Errorf("invalid weather args: %w", err)
			}
			city := strings.TrimSpace(input.City)
			if city == "" {
				return "", fmt.Errorf("city is empty")
			}

			if value, ok := mock[city]; ok {
				return value, nil
			}
			if value, ok := mock[strings.ToLower(city)]; ok {
				return value, nil
			}

			return fmt.Sprintf("%s：多云，27°C，微风，湿度 55%%。mock 数据：该城市没有专门配置，返回默认天气。", city), nil
		},
	}
}
