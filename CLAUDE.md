# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 项目概述

godis 是一个用 Go 实现的类 Redis 内存键值数据库，处于早期开发阶段。当前所有代码在单一的 `main` 包下，文件按模块命名。

详细的 agent 工作规则见 `AGENTS.md`（面向 AI agent 的行为约定、任务优先级、学习导向要求）。本文件聚焦建筑、构建、测试等实际操作信息。

## 构建与测试

```bash
# 构建
go build .

# 运行全部测试
go test ./...

# 运行特定测试（目前只有网络层 echo 测试）
go test -run TestNet -v

# 带 race detector
go test -race ./...
```

没有 Makefile 或其他构建工具，直接使用 `go` 命令。

## 架构

### 网络层 (`net.go`)

使用 `golang.org/x/sys/unix` 直接做 POSIX socket 系统调用，而不是 Go 标准库 `net`。设计目标是贴近系统层以便理解 Redis 的网络模型。

- `TcpServer(port)` — `socket() → setsockopt(SO_REUSEPORT) → bind() → listen()`，返回 listen fd
- `Accept(fd)` — 封装 `unix.Accept`，返回 client fd
- `Connect(host, port)` — `socket() → connect()`，返回 client fd
- `Read/Write/Close` — 对 `unix.Read/Write/Close` 的薄封装

`SO_REUSEPORT` 被开启是为了后续多进程/多线程共用同一端口的场景。

### 数据结构 (`list.go`)

`Node` 结构体为双向链表节点（`val int`, `next`, `prev`），目前是占位实现，后续将被事件循环和命令处理使用。

### 待实现的模块（占位文件）

| 文件 | 对应 Redis 概念 |
|------|----------------|
| `ae.go` | 事件循环 / IO 多路复用 |
| `dict.go` | 哈希表 / 字典 |
| `obj.go` | 数据对象模型 |
| `zset.go` | 有序集合 |
| `conf.go` | 配置管理 |
| `godis.go` | 入口 / 主程序 |

### 开发顺序

按照 `AGENTS.md` 的规划，推荐次序为：网络层 → 事件循环 → 命令解析 → 对象模型 → 数据结构 → 核心命令 → 持久化/复制等扩展。
