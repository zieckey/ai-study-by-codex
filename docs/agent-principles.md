# Go AI Agent 原理与实现详解

本文配合本仓库代码阅读。这个项目不是为了做一个强大的 Agent 框架，而是为了把核心机制拆到最小、最清楚。

## 1. 什么是 AI Agent？

普通聊天机器人通常是：

```text
用户问题 -> LLM -> 回答
```

AI Agent 多了一个“行动循环”：

```text
用户目标
  -> 模型思考下一步
  -> 需要工具吗？
      -> 是：调用工具，拿到观察结果，再回到模型
      -> 否：输出最终答案
```

所以，一个简单 Agent 可以写成：

```text
Agent = Model + Tools + State + Control Loop
```

- **Model**：大语言模型，负责理解问题、规划下一步、生成最终回答。
- **Tools**：工具，例如搜索、数据库、计算器、HTTP API、文件系统等。
- **State**：状态或记忆，也就是对话历史、工具结果、中间推理轨迹。
- **Control Loop**：控制循环，决定什么时候问模型、什么时候调用工具、什么时候结束。

本项目中：

- `internal/agent` 实现 Agent 通用循环；
- `internal/llm` 定义模型接口，并提供一个教学用 `RuleBased` 假模型；
- `internal/tools` 实现可调用工具，包括计算器、检索、mock 天气、当前时间、todo、受限 HTTP GET；
- `cmd/agent` 把它们组装成命令行程序。

## 2. 为什么先用假模型？

真实 LLM API 会带来 API Key、网络、价格、模型返回格式不稳定等额外问题。初学时，最重要的是先理解 Agent 的执行机制。

所以本项目用 `RuleBased` 模拟模型决策：

- 看到算术表达式时，选择 `calculator`；
- 看到 Agent、tool、RAG 等关键词时，选择 `search_notes`；
- 否则直接回答。

等你理解控制循环后，只需要实现同一个接口：

```go
type Client interface {
    Complete(ctx context.Context, messages []agent.Message, tools []agent.Tool) (agent.Action, error)
}
```

就可以接入真实 LLM。

## 3. Agent 的核心数据结构

看 `internal/agent/types.go`。

### Message

```go
type Message struct {
    Role    Role   `json:"role"`
    Content string `json:"content"`
}
```

它表示对话历史。Agent 每一步都会把模型回复、工具结果追加进去。这样下一轮模型就能“看见”之前发生了什么。

### Tool

```go
type Tool struct {
    Name        string
    Description string
    Parameters  json.RawMessage
    Execute     func(args json.RawMessage) (string, error)
}
```

工具由四部分组成：

- `Name`：模型用这个名字选择工具；
- `Description`：告诉模型工具能做什么；
- `Parameters`：参数 schema，告诉模型如何构造参数；
- `Execute`：Go 代码里真正执行工具的函数。

### Action

```go
type Action struct {
    Thought string          `json:"thought"`
    Tool    string          `json:"tool"`
    Args    json.RawMessage `json:"args"`
    Final   string          `json:"final"`
}
```

模型每一轮必须做两类决策之一：

1. 调工具：填写 `Tool` 和 `Args`；
2. 结束：填写 `Final`。

## 4. 控制循环详解

核心代码在 `internal/agent/agent.go` 的 `Run` 方法。简化后是：

```go
messages := []Message{systemPrompt, userQuestion}

for step := 1; step <= maxSteps; step++ {
    action := model.Complete(messages, tools)
    messages = append(messages, assistantAction)

    if action.Final != "" {
        return action.Final
    }

    tool := tools[action.Tool]
    observation := tool.Execute(action.Args)
    messages = append(messages, toolObservation)
}

return error("too many steps")
```

这里有几个关键点：

### 4.1 为什么需要 maxSteps？

真实模型可能陷入循环，例如一直调用同一个工具。`maxSteps` 是安全阀，避免程序无限运行。

### 4.2 为什么工具错误也放回 messages？

因为真实 Agent 不应该一遇到工具错误就崩溃。更好的做法是把错误作为观察结果交给模型，让模型决定重试、换工具或向用户解释。

本项目中：

```go
if err != nil {
    observation = "ERROR: " + err.Error()
}
```

### 4.3 为什么 transcript 很重要？

`transcript` 是完整执行轨迹。学习、调试、测试 Agent 时，它比最终答案更重要。你可以用：

```bash
go run ./cmd/agent --trace "23*7 等于多少？"
```

看到每一步发生了什么。

## 5. 工具调用如何工作？

以计算器为例，代码在 `internal/tools/calculator.go`。

用户输入：

```text
23*7 等于多少？
```

模型输出动作：

```json
{
  "thought": "The user asks for arithmetic...",
  "tool": "calculator",
  "args": {"expression":"23*7"}
}
```

Agent 找到名为 `calculator` 的工具，执行：

```go
result, err := tool.Execute(action.Args)
```

工具返回：

```text
161
```

Agent 把结果追加成 tool message。模型下一轮看到结果后输出最终答案：

```text
计算结果：161
```

## 6. RAG / 检索增强生成

`search_notes` 是一个极简 RAG 示例。真实 RAG 通常是：

```text
用户问题 -> 向量检索/关键词检索 -> 取回相关资料 -> LLM 基于资料回答
```

本项目为了简单，用内存 map 作为知识库：

```go
notes := map[string]string{
    "agent": "Agent = LLM + state/memory + tools + planning loop...",
    "tool":  "Tools let an Agent affect or inspect the outside world...",
}
```

当问题包含 Agent、tool、RAG 等关键词时，假模型会调用 `search_notes`，再基于检索结果回答。

## 7. 如何接入真实 LLM？

你需要新增一个结构体，实现 `llm.Client` 接口：

```go
type OpenAIClient struct {
    APIKey string
}

func (c *OpenAIClient) Complete(ctx context.Context, messages []agent.Message, tools []agent.Tool) (agent.Action, error) {
    // 1. 把 messages 转成模型 API 的 messages
    // 2. 把 tools 转成模型 API 的 tool/function schema
    // 3. 调用模型
    // 4. 如果模型选择工具，返回 agent.Action{Tool: ..., Args: ...}
    // 5. 如果模型直接回答，返回 agent.Action{Final: ...}
}
```

真实工程里还要处理：

- API 超时与重试；
- 模型返回 JSON 不合法；
- 工具参数校验；
- 日志与 tracing；
- prompt injection 防护；
- 工具权限控制；
- 成本与 token 限制。

## 8. 可以继续练习的扩展

### 练习 1：新增 time_now 工具

实现一个返回当前时间的工具：

```text
用户：现在几点？
Agent：调用 time_now -> 回答
```

### 练习 2：新增 todo 工具

实现内存版待办事项：

- `add_todo`
- `list_todos`
- `complete_todo`

这会帮助你理解 Agent 如何修改状态。

### 练习 3：新增 HTTP 工具

实现 `http_get`，让 Agent 能访问网页 API。注意一定要加安全限制，例如只允许访问白名单域名。

### 练习 4：替换成真实模型

把 `llm.NewRuleBased()` 替换成真实模型客户端。此时 Agent 控制循环无需改变，这就是接口抽象的价值。


## 9. 新增工具说明

本项目现在内置 6 个工具，覆盖 Agent 学习里最常见的几类能力。

### 9.1 weather：查询城市天气，当前返回 mock 数据

文件：`internal/tools/weather.go`

这个工具演示“外部 API 查询”的形态，但目前不访问真实天气服务，而是返回 mock 数据。

参数：

```json
{"city":"上海"}
```

示例：

```bash
go run ./cmd/agent --trace "上海天气怎么样？"
```

为什么先用 mock？因为学习 Agent 时，先稳定理解工具调用链路更重要：

```text
用户问天气 -> 模型选择 weather -> Agent 执行工具 -> 工具返回天气 -> 模型总结回答
```

以后要接真实天气 API，只需要替换 `weather.go` 内部实现，Agent 主循环不用改。

### 9.2 time_now：获取当前时间

文件：`internal/tools/time_now.go`

这个工具演示“实时信息查询”。模型自身不知道当前准确时间，所以应该调用工具。

参数可选：

```json
{"timezone":"Asia/Shanghai"}
```

示例：

```bash
go run ./cmd/agent --trace "现在几点？"
```

代码里 `NewTimeNow(clock func() time.Time)` 接收一个 clock 函数，是为了测试时可以传入固定时间，让测试稳定可重复。

### 9.3 todo_list：管理待办事项

文件：`internal/tools/todo.go`

这个工具演示“有状态工具”。支持三个 action：

```json
{"action":"add","text":"学习 Agent"}
{"action":"list"}
{"action":"complete","id":1}
```

它使用 `TodoStore` 在内存中保存数据，并用 `sync.Mutex` 保护并发访问。

注意：当前 CLI 每次启动是一个新进程，所以待办事项不会跨命令保存。如果想持久化，可以把 `TodoStore` 改成读写 JSON 文件或 SQLite。

### 9.4 http_get：访问受限 HTTP API

文件：`internal/tools/http_get.go`

这个工具演示“高风险外部访问工具”的安全边界。它默认只允许：

- HTTPS；
- host 在白名单中；
- 响应体最多读取 4096 字节；
- 请求有 5 秒超时。

默认白名单：

```text
example.com
httpbin.org
```

示例：

```bash
go run ./cmd/agent --trace "帮我 GET https://example.com"
```

为什么要限制？因为如果让 Agent 任意访问 URL，可能产生 SSRF、访问内网资源、下载超大内容、泄露敏感信息等风险。真实项目里，所有能影响外部世界的工具都应该有权限边界。

## 10. 工具设计经验

新增工具时建议遵循这个模板：

```go
func NewMyTool() agent.Tool {
    return agent.Tool{
        Name:        "my_tool",
        Description: "Tell the model when to use this tool.",
        Parameters:  json.RawMessage(`{"type":"object", ...}`),
        Execute: func(args json.RawMessage) (string, error) {
            // 1. 解析参数
            // 2. 校验参数
            // 3. 执行动作
            // 4. 返回简短、结构清楚的 observation
        },
    }
}
```

关键原则：

- 参数必须校验；
- 错误要清楚；
- 返回给模型的 observation 不要太长；
- 涉及网络、文件、数据库、shell 的工具必须做权限限制；
- 有状态工具要考虑并发安全和持久化策略。
