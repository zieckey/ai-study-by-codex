package tools

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/zieckey/ai-study-by-codex/internal/agent"
)

// NewCalculator returns a deliberately tiny calculator tool.
// It supports expressions such as: 1+2, 10 - 3, 6*7, 8/2.
func NewCalculator() agent.Tool {
	return agent.Tool{
		Name:        "calculator",
		Description: "Evaluate a simple binary arithmetic expression. Supported operators: +, -, *, /.",
		Parameters:  json.RawMessage(`{"type":"object","properties":{"expression":{"type":"string"}},"required":["expression"]}`),
		Execute: func(args json.RawMessage) (string, error) {
			var input struct {
				Expression string `json:"expression"`
			}
			if err := json.Unmarshal(args, &input); err != nil {
				return "", fmt.Errorf("invalid calculator args: %w", err)
			}

			result, err := evalSimple(input.Expression)
			if err != nil {
				return "", err
			}
			return strconv.FormatFloat(result, 'f', -1, 64), nil
		},
	}
}

func evalSimple(expression string) (float64, error) {
	expr := strings.ReplaceAll(expression, " ", "")
	if expr == "" {
		return 0, fmt.Errorf("empty expression")
	}

	for _, op := range []string{"+", "-", "*", "/"} {
		idx := strings.LastIndex(expr, op)
		if idx <= 0 {
			continue
		}
		left, err := strconv.ParseFloat(expr[:idx], 64)
		if err != nil {
			return 0, fmt.Errorf("invalid left operand %q", expr[:idx])
		}
		right, err := strconv.ParseFloat(expr[idx+1:], 64)
		if err != nil {
			return 0, fmt.Errorf("invalid right operand %q", expr[idx+1:])
		}

		switch op {
		case "+":
			return left + right, nil
		case "-":
			return left - right, nil
		case "*":
			return left * right, nil
		case "/":
			if right == 0 {
				return 0, fmt.Errorf("division by zero")
			}
			return left / right, nil
		}
	}

	return 0, fmt.Errorf("unsupported expression %q; use a simple form like 6*7", expression)
}
