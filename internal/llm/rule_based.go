package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/zieckey/ai-study-by-codex/internal/agent"
)

// RuleBased is a fake model used for learning the Agent control loop without
// paying for or configuring an LLM API key.
type RuleBased struct{}

func NewRuleBased() *RuleBased { return &RuleBased{} }

func (m *RuleBased) Complete(ctx context.Context, messages []agent.Message, tools []agent.Tool) (agent.Action, error) {
	select {
	case <-ctx.Done():
		return agent.Action{}, ctx.Err()
	default:
	}

	last := messages[len(messages)-1]
	if last.Role == agent.RoleTool {
		return m.answerFromObservation(messages)
	}

	question := latestUserQuestion(messages)
	lower := strings.ToLower(question)

	if calls := planMultipleToolCalls(question, lower, tools); len(calls) > 1 {
		return agent.Action{
			Thought: "The user asks for multiple independent things, so I can call multiple tools in one model step.",
			Calls:   calls,
		}, nil
	}

	if expr := findFirstArithmetic(question); expr != "" && hasTool(tools, "calculator") {
		args, _ := json.Marshal(map[string]string{"expression": expr})
		return agent.Action{
			Thought: "The user asks for arithmetic. I should use the calculator tool to avoid mental math mistakes.",
			Tool:    "calculator",
			Args:    args,
		}, nil
	}

	if wantsTime(lower) && hasTool(tools, "time_now") {
		args, _ := json.Marshal(map[string]string{"timezone": guessTimezone(question)})
		return agent.Action{
			Thought: "The user asks for the current time. I should call time_now instead of guessing.",
			Tool:    "time_now",
			Args:    args,
		}, nil
	}

	if wantsWeather(lower) && hasTool(tools, "weather") {
		args, _ := json.Marshal(map[string]string{"city": guessCity(question)})
		return agent.Action{
			Thought: "The user asks about city weather. I should call the mock weather tool.",
			Tool:    "weather",
			Args:    args,
		}, nil
	}

	if action, ok := todoAction(question, lower); ok && hasTool(tools, "todo_list") {
		args, _ := json.Marshal(action.args)
		return agent.Action{
			Thought: action.thought,
			Tool:    "todo_list",
			Args:    args,
		}, nil
	}

	if target := findHTTPSURL(question); target != "" && hasTool(tools, "http_get") {
		args, _ := json.Marshal(map[string]string{"url": target})
		return agent.Action{
			Thought: "The user provided a HTTPS URL. I should use the restricted http_get tool.",
			Tool:    "http_get",
			Args:    args,
		}, nil
	}

	for _, keyword := range []string{"agent", "tool", "rag", "memory", "记忆", "工具", "检索"} {
		if strings.Contains(lower, keyword) && hasTool(tools, "search_notes") {
			args, _ := json.Marshal(map[string]string{"query": keyword})
			return agent.Action{
				Thought: "The question is about Agent concepts. I should retrieve local notes first.",
				Tool:    "search_notes",
				Args:    args,
			}, nil
		}
	}

	return agent.Action{
		Thought: "No tool is required; answer directly.",
		Final:   "这是一个教学版 Agent：它会在需要时选择工具，否则直接回答。你可以试试：'23*7 等于多少？'、'上海天气怎么样？'、'现在几点？'、'添加待办：学习 Agent' 或 '帮我 GET https://example.com'。",
	}, nil
}

func (m *RuleBased) answerFromObservation(messages []agent.Message) (agent.Action, error) {
	observations := trailingToolObservations(messages)
	if len(observations) > 1 {
		return agent.Action{
			Thought: "Multiple tools returned observations. Now I can combine them into one answer.",
			Final:   summarizeObservations(observations),
		}, nil
	}

	observation := messages[len(messages)-1].Content
	question := latestUserQuestion(messages)

	if strings.HasPrefix(observation, "calculator =>") {
		return agent.Action{
			Thought: "The calculator returned the exact result. Now I can answer the user.",
			Final:   fmt.Sprintf("计算结果：%s", strings.TrimSpace(strings.TrimPrefix(observation, "calculator =>"))),
		}, nil
	}

	if strings.HasPrefix(observation, "time_now =>") {
		return agent.Action{
			Thought: "The time_now tool returned the current time. Now I can answer.",
			Final:   fmt.Sprintf("当前时间：%s", strings.TrimSpace(strings.TrimPrefix(observation, "time_now =>"))),
		}, nil
	}

	if strings.HasPrefix(observation, "weather =>") {
		return agent.Action{
			Thought: "The weather tool returned mock weather data. Now I can answer and mention it is mock data.",
			Final:   fmt.Sprintf("天气查询结果：%s", strings.TrimSpace(strings.TrimPrefix(observation, "weather =>"))),
		}, nil
	}

	if strings.HasPrefix(observation, "todo_list =>") {
		return agent.Action{
			Thought: "The todo tool completed the requested operation. Now I can summarize the result.",
			Final:   fmt.Sprintf("待办事项操作结果：\n%s", strings.TrimSpace(strings.TrimPrefix(observation, "todo_list =>"))),
		}, nil
	}

	if strings.HasPrefix(observation, "http_get =>") {
		return agent.Action{
			Thought: "The restricted HTTP GET tool returned the response. Now I can show a concise result.",
			Final:   fmt.Sprintf("HTTP GET 结果：\n%s", strings.TrimSpace(strings.TrimPrefix(observation, "http_get =>"))),
		}, nil
	}

	if strings.HasPrefix(observation, "search_notes =>") {
		return agent.Action{
			Thought: "The notes provide context. Now I can summarize it for the user.",
			Final:   fmt.Sprintf("结合本地笔记回答：\n%s\n\n针对你的问题“%s”：AI Agent 的核心是一个循环：理解目标，决定是否调用工具，观察工具结果，再继续推理，直到给出最终答案。", strings.TrimSpace(strings.TrimPrefix(observation, "search_notes =>")), question),
		}, nil
	}

	return agent.Action{Thought: "The tool returned an error or unknown observation.", Final: "工具返回了无法处理的结果：" + observation}, nil
}

func latestUserQuestion(messages []agent.Message) string {
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == agent.RoleUser {
			return messages[i].Content
		}
	}
	return ""
}

func hasTool(tools []agent.Tool, name string) bool {
	for _, tool := range tools {
		if tool.Name == name {
			return true
		}
	}
	return false
}

func findFirstArithmetic(text string) string {
	matches := findAllArithmetic(text)
	if len(matches) == 0 {
		return ""
	}
	return matches[0]
}

func findAllArithmetic(text string) []string {
	re := regexp.MustCompile(`[-+]?\d+(?:\.\d+)?\s*[+\-*/]\s*[-+]?\d+(?:\.\d+)?`)
	return re.FindAllString(text, -1)
}

func wantsTime(lower string) bool {
	for _, keyword := range []string{"time", "now", "几点", "时间", "当前时间", "现在"} {
		if strings.Contains(lower, keyword) {
			return true
		}
	}
	return false
}

func guessTimezone(question string) string {
	lower := strings.ToLower(question)
	switch {
	case strings.Contains(question, "上海"), strings.Contains(question, "北京"), strings.Contains(question, "中国"), strings.Contains(lower, "china"):
		return "Asia/Shanghai"
	case strings.Contains(lower, "utc"):
		return "UTC"
	default:
		return "Asia/Shanghai"
	}
}

func wantsWeather(lower string) bool {
	for _, keyword := range []string{"weather", "天气", "气温", "下雨", "晴"} {
		if strings.Contains(lower, keyword) {
			return true
		}
	}
	return false
}

func guessCity(question string) string {
	matches := findWeatherIntentsWithIndex(question)
	if len(matches) > 0 {
		return matches[0].city
	}
	cities := knownCities()
	lower := strings.ToLower(question)
	for _, city := range cities {
		if strings.Contains(question, city) || strings.Contains(lower, city) {
			return city
		}
	}
	return "上海"
}

type timeIntent struct {
	pos      int
	timezone string
}

func findTimeIntentsWithIndex(question string) []timeIntent {
	lower := strings.ToLower(question)
	keywords := []string{"当前时间", "time", "now", "几点", "时间", "现在"}
	var raw []timeIntent
	for _, keyword := range keywords {
		searchFrom := 0
		for {
			idx := strings.Index(lower[searchFrom:], keyword)
			if idx < 0 {
				break
			}
			pos := searchFrom + idx
			raw = append(raw, timeIntent{pos: pos, timezone: guessTimezoneAround(question, pos)})
			searchFrom = pos + len(keyword)
			if searchFrom >= len(lower) {
				break
			}
		}
	}
	sort.SliceStable(raw, func(i, j int) bool { return raw[i].pos < raw[j].pos })

	var intents []timeIntent
	for _, candidate := range raw {
		if len(intents) > 0 && candidate.pos-intents[len(intents)-1].pos <= 8 {
			continue
		}
		intents = append(intents, candidate)
	}
	return intents
}

func guessTimezoneAround(question string, pos int) string {
	lower := strings.ToLower(question)
	windowStart := max(0, pos-16)
	windowEnd := min(len(lower), pos+32)
	window := lower[windowStart:windowEnd]
	switch {
	case strings.Contains(window, "utc"):
		return "UTC"
	case strings.Contains(window, "上海"), strings.Contains(window, "北京"), strings.Contains(window, "中国"), strings.Contains(window, "china"):
		return "Asia/Shanghai"
	default:
		return guessTimezone(question)
	}
}

type weatherIntent struct {
	pos  int
	city string
}

func findWeatherIntentsWithIndex(question string) []weatherIntent {
	lower := strings.ToLower(question)
	var intents []weatherIntent
	seenAt := map[int]bool{}

	for _, city := range knownCities() {
		searchFrom := 0
		needle := city
		text := question
		if isASCII(city) {
			needle = strings.ToLower(city)
			text = lower
		}

		for {
			idx := strings.Index(text[searchFrom:], needle)
			if idx < 0 {
				break
			}
			pos := searchFrom + idx
			if looksLikeWeatherAround(lower, pos, pos+len(needle)) && !seenAt[pos] {
				intents = append(intents, weatherIntent{pos: pos, city: city})
				seenAt[pos] = true
			}
			searchFrom = pos + len(needle)
			if searchFrom >= len(text) {
				break
			}
		}
	}

	sort.SliceStable(intents, func(i, j int) bool { return intents[i].pos < intents[j].pos })
	if len(intents) > 0 {
		return intents
	}

	if pos := firstIntentIndex(lower, "weather", "天气", "气温", "下雨", "晴"); pos >= 0 {
		return []weatherIntent{{pos: pos, city: "上海"}}
	}
	return nil
}

func knownCities() []string {
	return []string{"北京", "上海", "深圳", "杭州", "广州", "beijing", "shanghai", "shenzhen", "hangzhou", "guangzhou"}
}

func looksLikeWeatherAround(lower string, start, end int) bool {
	windowStart := max(0, start-12)
	windowEnd := min(len(lower), end+12)
	window := lower[windowStart:windowEnd]
	return containsAny(window, "weather", "天气", "气温", "下雨", "晴")
}

func isASCII(s string) bool {
	for _, r := range s {
		if r > 127 {
			return false
		}
	}
	return true
}

type todoDecision struct {
	thought string
	args    map[string]any
}

func todoAction(question, lower string) (todoDecision, bool) {
	if !strings.Contains(lower, "todo") && !strings.Contains(question, "待办") && !strings.Contains(question, "任务") {
		return todoDecision{}, false
	}

	if containsAny(lower, "list", "show", "查看", "列出", "有什么") || containsAny(question, "查看", "列出") {
		return todoDecision{thought: "The user wants to list todos.", args: map[string]any{"action": "list"}}, true
	}

	if containsAny(lower, "complete", "done", "finish", "完成") || strings.Contains(question, "完成") {
		id := findFirstInt(question)
		return todoDecision{thought: "The user wants to complete a todo item.", args: map[string]any{"action": "complete", "id": id}}, true
	}

	if containsAny(lower, "add", "create", "添加", "新增", "加入") || containsAny(question, "添加", "新增", "加入") {
		text := extractTodoText(question)
		return todoDecision{thought: "The user wants to add a todo item.", args: map[string]any{"action": "add", "text": text}}, true
	}

	return todoDecision{thought: "The user mentioned todos without a specific action, so listing todos is safest.", args: map[string]any{"action": "list"}}, true
}

type todoIntent struct {
	pos  int
	args map[string]any
}

func findTodoIntentsWithIndex(question string) []todoIntent {
	segments := splitIntentSegments(question)
	var intents []todoIntent
	for _, segment := range segments {
		lower := strings.ToLower(segment.text)
		action, ok := todoAction(segment.text, lower)
		if !ok {
			continue
		}
		pos := firstIntentIndex(lower, "todo", "待办", "任务", "add", "create", "添加", "新增", "加入", "list", "show", "查看", "列出", "complete", "done", "finish", "完成")
		if pos < 0 {
			pos = 0
		}
		intents = append(intents, todoIntent{pos: segment.start + pos, args: action.args})
	}
	return intents
}

type textSegment struct {
	start int
	text  string
}

func splitIntentSegments(text string) []textSegment {
	var segments []textSegment
	start := 0
	for i, r := range text {
		if r == '，' || r == ',' || r == '；' || r == ';' || r == '\n' {
			if trimmed := strings.TrimSpace(text[start:i]); trimmed != "" {
				segments = append(segments, textSegment{start: start + strings.Index(text[start:i], trimmed), text: trimmed})
			}
			start = i + len(string(r))
		}
	}
	if start <= len(text) {
		if trimmed := strings.TrimSpace(text[start:]); trimmed != "" {
			segments = append(segments, textSegment{start: start + strings.Index(text[start:], trimmed), text: trimmed})
		}
	}
	if len(segments) == 0 && strings.TrimSpace(text) != "" {
		segments = append(segments, textSegment{start: strings.Index(text, strings.TrimSpace(text)), text: strings.TrimSpace(text)})
	}
	return segments
}

type searchNotesIntent struct {
	pos   int
	query string
}

func findSearchNotesIntentsWithIndex(question string) []searchNotesIntent {
	lower := strings.ToLower(question)
	keywords := []string{"agent", "tool", "rag", "memory", "记忆", "工具", "检索"}
	var intents []searchNotesIntent
	seenQuery := map[string]bool{}
	for _, keyword := range keywords {
		searchFrom := 0
		for {
			idx := strings.Index(lower[searchFrom:], keyword)
			if idx < 0 {
				break
			}
			pos := searchFrom + idx
			if !seenQuery[keyword] {
				intents = append(intents, searchNotesIntent{pos: pos, query: keyword})
				seenQuery[keyword] = true
			}
			searchFrom = pos + len(keyword)
			if searchFrom >= len(lower) {
				break
			}
		}
	}
	sort.SliceStable(intents, func(i, j int) bool { return intents[i].pos < intents[j].pos })
	return intents
}

func containsAny(s string, keywords ...string) bool {
	for _, keyword := range keywords {
		if strings.Contains(s, keyword) {
			return true
		}
	}
	return false
}

func findFirstInt(text string) int {
	re := regexp.MustCompile(`\d+`)
	match := re.FindString(text)
	if match == "" {
		return 0
	}
	value, _ := strconv.Atoi(match)
	return value
}

func extractTodoText(question string) string {
	separators := []string{"：", ":"}
	for _, sep := range separators {
		if idx := strings.Index(question, sep); idx >= 0 && idx+len(sep) < len(question) {
			return strings.TrimSpace(question[idx+len(sep):])
		}
	}

	replacer := strings.NewReplacer("添加", "", "新增", "", "加入", "", "add", "", "todo", "", "待办", "", "任务", "")
	text := strings.TrimSpace(replacer.Replace(question))
	if text == "" {
		return question
	}
	return text
}

func findHTTPSURL(text string) string {
	re := regexp.MustCompile(`https://[^\s，。]+`)
	return re.FindString(text)
}

func planMultipleToolCalls(question, lower string, tools []agent.Tool) []agent.ToolCall {
	var candidates []plannedToolCall

	add := func(pos int, name string, args any) {
		if pos < 0 || !hasTool(tools, name) {
			return
		}
		payload, _ := json.Marshal(args)
		candidates = append(candidates, plannedToolCall{pos: pos, call: agent.ToolCall{Tool: name, Args: payload}})
	}

	for _, match := range findAllArithmeticWithIndex(question) {
		add(match.pos, "calculator", map[string]string{"expression": match.expr})
	}
	for _, match := range findTimeIntentsWithIndex(question) {
		add(match.pos, "time_now", map[string]string{"timezone": match.timezone})
	}
	for _, match := range findWeatherIntentsWithIndex(question) {
		add(match.pos, "weather", map[string]string{"city": match.city})
	}
	for _, match := range findTodoIntentsWithIndex(question) {
		add(match.pos, "todo_list", match.args)
	}
	for _, match := range findSearchNotesIntentsWithIndex(question) {
		add(match.pos, "search_notes", map[string]string{"query": match.query})
	}
	for _, match := range findAllHTTPSURLsWithIndex(question) {
		add(match.pos, "http_get", map[string]string{"url": match.url})
	}

	sort.SliceStable(candidates, func(i, j int) bool { return candidates[i].pos < candidates[j].pos })

	calls := make([]agent.ToolCall, 0, len(candidates))
	for _, candidate := range candidates {
		calls = append(calls, candidate.call)
	}
	return calls
}

type plannedToolCall struct {
	pos  int
	call agent.ToolCall
}

type arithmeticMatch struct {
	pos  int
	expr string
}

func findAllArithmeticWithIndex(text string) []arithmeticMatch {
	re := regexp.MustCompile(`[-+]?\d+(?:\.\d+)?\s*[+\-*/]\s*[-+]?\d+(?:\.\d+)?`)
	indexes := re.FindAllStringIndex(text, -1)
	matches := make([]arithmeticMatch, 0, len(indexes))
	for _, index := range indexes {
		matches = append(matches, arithmeticMatch{pos: index[0], expr: text[index[0]:index[1]]})
	}
	return matches
}

type urlMatch struct {
	pos int
	url string
}

func findAllHTTPSURLsWithIndex(text string) []urlMatch {
	re := regexp.MustCompile(`https://[^\s，。]+`)
	indexes := re.FindAllStringIndex(text, -1)
	matches := make([]urlMatch, 0, len(indexes))
	for _, index := range indexes {
		matches = append(matches, urlMatch{pos: index[0], url: text[index[0]:index[1]]})
	}
	return matches
}

func firstIntentIndex(text string, keywords ...string) int {
	best := -1
	for _, keyword := range keywords {
		idx := strings.Index(text, strings.ToLower(keyword))
		if idx >= 0 && (best == -1 || idx < best) {
			best = idx
		}
	}
	return best
}

func trailingToolObservations(messages []agent.Message) []string {
	var reversed []string
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role != agent.RoleTool {
			break
		}
		reversed = append(reversed, messages[i].Content)
	}

	observations := make([]string, len(reversed))
	for i := range reversed {
		observations[len(reversed)-1-i] = reversed[i]
	}
	return observations
}

func summarizeObservations(observations []string) string {
	lines := []string{"多工具调用结果："}
	for _, observation := range observations {
		lines = append(lines, "- "+strings.TrimSpace(observation))
	}
	return strings.Join(lines, "\n")
}
