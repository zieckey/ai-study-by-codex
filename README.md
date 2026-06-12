# ai-study-by-codex

Prompt: 我想创建一个golang的简单项目，帮我学习如何编写AI Agent，把原理和实现都做详细讲解
一个用于学习 **Go 语言 AI Agent 原理与实现** 的最小可运行项目。

特点：

- 只使用 Go 标准库，方便理解核心概念；
- 内置一个 `RuleBased` 假模型，不需要 API Key 就能运行；
- 演示 Agent 最重要的控制循环：**Think → Act → Observe → Repeat → Final**；
- 内置多个工具：`calculator`、`search_notes`、`weather`、`time_now`、`todo_list`、`http_get`；
- 后续可以把 `internal/llm` 替换成真实 LLM API 客户端。

## 快速开始

```bash
go run ./cmd/agent
```

试试计算工具：

```bash
go run ./cmd/agent --trace "23*7 等于多少？"
```

试试检索工具：

```bash
go run ./cmd/agent --trace "什么是 Agent 的工具调用？"
```

试试 mock 天气工具：

```bash
go run ./cmd/agent --trace "上海天气怎么样？"
```

试试当前时间工具：

```bash
go run ./cmd/agent --trace "现在几点？"
```

试试待办事项工具：

```bash
go run ./cmd/agent --trace "添加待办：学习 Agent 工具调用"
go run ./cmd/agent --trace "查看待办"
```

注意：当前 CLI 每次启动都会创建新的内存 todo store，所以 todo 数据只在单次进程内保存。这个设计是为了教学简单。

试试受限 HTTP GET 工具：

```bash
go run ./cmd/agent --trace "帮我 GET https://example.com"
```

`http_get` 只允许 HTTPS，并且只允许访问白名单 host，默认是 `example.com` 和 `httpbin.org`。

运行测试：

```bash
make test
```

如果你直接运行 Go 命令，建议带上纯 Go build tags，避免部分 macOS + Go 1.22 环境里的 `missing LC_UUID load command`：

```bash
go test -tags=netgo,osusergo ./...
go run -tags=netgo,osusergo ./cmd/agent --trace "上海天气怎么样？"
```

## 项目结构

```text
cmd/agent/              CLI 入口
internal/agent/         Agent 控制循环与通用类型
internal/llm/           模型抽象，以及教学用 RuleBased 假模型
internal/tools/         工具实现：计算器、本地笔记检索、mock 天气、当前时间、todo、受限 HTTP GET
docs/agent-principles.md 原理与代码讲解
```

## 学习路线

1. 先读 `docs/agent-principles.md` 理解 Agent 的组成；
2. 再运行 `go run ./cmd/agent --trace "23*7 等于多少？"` 观察完整轨迹；
3. 阅读 `internal/agent/agent.go`，重点看 `Run` 方法；
4. 阅读 `internal/tools` 里的工具实现，理解工具参数校验、状态管理和安全限制；
5. 新增一个自己的工具，例如 `file_search`、`json_extract` 或 `calendar`；
6. 把 `internal/llm.RuleBased` 替换成真实 LLM API 客户端。


## macOS 上 missing LC_UUID load command 的处理

在部分 macOS 26 + Go 1.22.x 环境里，普通 `go test ./...` 或 `go run ./cmd/agent` 可能报：

```text
dyld: missing LC_UUID load command
```

本项目的根因是：`net/http` 等包在 macOS 默认会把 `runtime/cgo` 链进测试/运行二进制；当前系统 dyld 对 Go 1.22 内部链接生成的 cgo Mach-O 二进制更严格，发现缺少 `LC_UUID` 后拒绝运行。

本项目不需要 cgo，所以默认推荐使用纯 Go 构建：

```bash
make test
make run q="现在几点？"
```

等价的原生命令是：

```bash
go test -tags=netgo,osusergo ./...
go run -tags=netgo,osusergo ./cmd/agent --trace "现在几点？"
```

另一个可行但更重的绕过方式是外部链接：

```bash
go test -ldflags=-linkmode=external ./...
```

但对这个教学项目来说，`netgo,osusergo` 更合适，因为它直接避免引入 cgo。
