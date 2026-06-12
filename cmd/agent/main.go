package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/zieckey/ai-study-by-codex/internal/agent"
	"github.com/zieckey/ai-study-by-codex/internal/llm"
	"github.com/zieckey/ai-study-by-codex/internal/tools"
)

func main() {
	trace := flag.Bool("trace", false, "print the full agent transcript")
	flag.Parse()

	question := strings.Join(flag.Args(), " ")
	if strings.TrimSpace(question) == "" {
		question = "什么是 AI Agent 的工具调用？"
	}

	notes := map[string]string{
		"agent": "Agent = LLM + state/memory + tools + planning loop. It repeatedly decides what to do next.",
		"tool":  "Tools let an Agent affect or inspect the outside world, for example calculator, search, database, shell, HTTP API.",
		"rag":   "RAG means retrieval augmented generation: search relevant knowledge first, then answer with that context.",
	}

	ag, err := agent.New(
		llm.NewRuleBased(),
		[]agent.Tool{
			tools.NewCalculator(),
			tools.NewNotesSearch(notes),
			tools.NewMockWeather(),
			tools.NewTimeNow(nil),
			tools.NewTodoList(nil),
			tools.NewHTTPGet(),
		},
		6,
	)
	if err != nil {
		fmt.Fprintln(os.Stderr, "create agent:", err)
		os.Exit(1)
	}

	answer, transcript, err := ag.Run(context.Background(), question)
	if err != nil {
		fmt.Fprintln(os.Stderr, "run agent:", err)
		os.Exit(1)
	}

	fmt.Println(answer)

	if *trace {
		fmt.Println("\n--- transcript ---")
		for i, msg := range transcript {
			fmt.Printf("[%02d] %s: %s\n", i, msg.Role, msg.Content)
		}
	}
}
