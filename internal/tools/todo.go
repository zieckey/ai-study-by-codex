package tools

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/zieckey/ai-study-by-codex/internal/agent"
)

type TodoItem struct {
	ID        int    `json:"id"`
	Text      string `json:"text"`
	Completed bool   `json:"completed"`
}

type TodoStore struct {
	mu     sync.Mutex
	nextID int
	items  []TodoItem
}

func NewTodoStore() *TodoStore {
	return &TodoStore{nextID: 1}
}

func (s *TodoStore) Add(text string) TodoItem {
	s.mu.Lock()
	defer s.mu.Unlock()

	item := TodoItem{ID: s.nextID, Text: text}
	s.nextID++
	s.items = append(s.items, item)
	return item
}

func (s *TodoStore) Complete(id int) (TodoItem, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range s.items {
		if s.items[i].ID == id {
			s.items[i].Completed = true
			return s.items[i], true
		}
	}
	return TodoItem{}, false
}

func (s *TodoStore) List() []TodoItem {
	s.mu.Lock()
	defer s.mu.Unlock()

	items := make([]TodoItem, len(s.items))
	copy(items, s.items)
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	return items
}

// NewTodoList returns a stateful in-memory todo tool.
// action can be add, list, or complete.
func NewTodoList(store *TodoStore) agent.Tool {
	if store == nil {
		store = NewTodoStore()
	}

	return agent.Tool{
		Name:        "todo_list",
		Description: "Manage an in-memory todo list. Actions: add, list, complete.",
		Parameters:  json.RawMessage(`{"type":"object","properties":{"action":{"type":"string","enum":["add","list","complete"]},"text":{"type":"string"},"id":{"type":"integer"}},"required":["action"]}`),
		Execute: func(args json.RawMessage) (string, error) {
			var input struct {
				Action string `json:"action"`
				Text   string `json:"text"`
				ID     int    `json:"id"`
			}
			if err := json.Unmarshal(args, &input); err != nil {
				return "", fmt.Errorf("invalid todo_list args: %w", err)
			}

			switch strings.ToLower(strings.TrimSpace(input.Action)) {
			case "add":
				text := strings.TrimSpace(input.Text)
				if text == "" {
					return "", fmt.Errorf("text is required for add")
				}
				item := store.Add(text)
				return fmt.Sprintf("added todo #%d: %s", item.ID, item.Text), nil
			case "complete":
				if input.ID <= 0 {
					return "", fmt.Errorf("positive id is required for complete")
				}
				item, ok := store.Complete(input.ID)
				if !ok {
					return "", fmt.Errorf("todo #%d not found", input.ID)
				}
				return fmt.Sprintf("completed todo #%d: %s", item.ID, item.Text), nil
			case "list":
				items := store.List()
				if len(items) == 0 {
					return "todo list is empty", nil
				}
				lines := make([]string, 0, len(items))
				for _, item := range items {
					mark := "[ ]"
					if item.Completed {
						mark = "[x]"
					}
					lines = append(lines, fmt.Sprintf("%s #%d %s", mark, item.ID, item.Text))
				}
				return strings.Join(lines, "\n"), nil
			default:
				return "", fmt.Errorf("unsupported todo action %q", input.Action)
			}
		},
	}
}
