# macOS missing LC_UUID load command 排查记录

## 现象

在当前环境中运行：

```bash
go test ./...
go run ./cmd/agent --trace "上海天气怎么样？"
```

会出现类似错误：

```text
dyld: missing LC_UUID load command
signal: abort trap
```

## 已验证事实

### 1. 环境

```text
GOOS=darwin
GOARCH=arm64
GOVERSION=go1.22.4
CGO_ENABLED=1
macOS=26.5
```

### 2. 最小纯 Go 程序可运行，但缺少 LC_UUID

用 Go 内部链接器构建一个最小 `main`，`otool -l` 看不到 `LC_UUID`，但它可以运行。

这说明“缺少 LC_UUID”本身不一定导致所有 Go 程序失败。

### 3. 引入 runtime/cgo 后失败

本项目使用 `net/http`，在 macOS 默认配置下会引入 `runtime/cgo`：

```bash
go list -deps ./cmd/agent | grep '^runtime/cgo$'
```

测试二进制和 `go run` 二进制在 dyld 加载阶段失败。

### 4. 外部链接可通过

```bash
go test -ldflags=-linkmode=external ./...
```

可以通过，因为外部链接器会生成包含 `LC_UUID` 的 Mach-O。

### 5. 禁用 cgo / 使用纯 Go net,user 实现可通过

```bash
CGO_ENABLED=0 go test ./...
go test -tags=netgo,osusergo ./...
```

都可以通过。

## 根因

当前失败路径不是业务代码 bug，而是 macOS 26 dyld、Go 1.22 内部链接器、cgo Mach-O 输出之间的兼容性问题。

本项目并不依赖 cgo。`runtime/cgo` 是通过标准库的系统解析路径被默认引入的。因此最小修复是在本项目的默认开发命令中使用纯 Go build tags：

```text
netgo,osusergo
```

## 项目修复

新增 `Makefile`：

```makefile
GOFLAGS ?= -tags=netgo,osusergo

test:
	go test ./...

run:
	go run ./cmd/agent --trace "$(q)"
```

因此推荐使用：

```bash
make test
make run q="上海天气怎么样？"
```

## 其他可选方案

### 方案 A：升级 Go

如果本机可升级 Go，建议升级到更新版本。较新的 Go 工具链可能改善 macOS 新版本上的 Mach-O 兼容性。

### 方案 B：外部链接

```bash
go test -ldflags=-linkmode=external ./...
```

缺点是更慢，并且依赖系统 C toolchain。

### 方案 C：全局关闭 cgo

```bash
go env -w CGO_ENABLED=0
```

不建议随便全局设置，因为其他项目可能真的需要 cgo。
