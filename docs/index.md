# PortWeave Documentation

PortWeave 是面向嵌入式调试和自动化验证的开源串口拓扑、协议模拟与 MCP 自动化工作台。本索引按“先跑起来、再理解拓扑、最后自动化与贡献”的顺序组织文档。

## Start here

- [Getting Started](getting-started.md) — 5 分钟无硬件路径：安装依赖、启动开发应用、加载内置示例、启动拓扑。
- [Workflows](workflows.md) — observe / demo / build / automate 四层工作流。
- [Use Cases](use-cases.md) — 按真实场景选择 PortWeave 能力。
- [Examples](examples.md) — 当前 12 个内置 demo workspace 的 ID、拓扑和硬件要求。

## Serial graph

- [Serial Graph Node Catalog](serial-graph-node-catalog.md) — 当前 serial graph provider catalog 中的所有节点类型、端口和常见错误。
- [Serial Graph Runtime Flow](serial-graph-runtime-flow.md) — 校验、启动、路由、缓冲、帧、状态与停止清理流程。
- [Workspace and Graph Schema](workspace-graph-schema.md) — `portweave.workspace.v1` 与 `portweave.graph.v1` 快照边界。
- [Identity Model](identity-model.md) — workspace、graph、node、edge、resource、demo 和 MCP template ID 的差异。
- [Remote Serial Graph Node](remote-serial-node.md) — `serial.remote` raw TCP client/server 节点、安全边界与故障排查。
- [Serial Cross-Platform Inventory](serial-cross-platform-inventory.md) — 当前真实串口边界和跨平台抽象清单。

## Automation

- [MCP API and Recipes](mcp-api.md) — 当前 64 个 MCP 工具、工具分组、protocol template catalog 与 graph validate/start/query/stop 配方。
- [AI Automation Guide](ai-automation.md) — agent 操作安全分层、无硬件优先配方、设备写入防护。

## Development and architecture

- [Architecture](architecture.md) — Go + Wails + Vue/TypeScript + MCP 的当前架构。
- [Architecture Decisions](development/architecture-decisions.md) — 保持现有技术栈、MCP-first、Wails v3 alpha guardrails 等 ADR。
- [Testing](development/testing.md) — Go、frontend、coverage、typecheck、build、E2E、Makefile 与 CI gate 命令。
- [Headless Integration](development/headless-integration.md) — server-mode 启动、readiness、teardown 与 MCP smoke harness 前置条件。
- [Benchmarks](development/benchmarks.md) — graph routing、buffer、script、protocol、remote TCP 的 benchmark 分类与初始阈值。
- [Security Tests](development/security-tests.md) — MCP/raw TCP/script/runtime/release artifact 的安全测试 taxonomy。
- [Release](development/release.md) — release gate、artifact smoke、checksum 与矩阵扩展纪律。
- [Windows Serial Testing](development/windows-serial-testing.md) — Windows COM 与 POSIX socat 集成测试策略。

## Troubleshooting

- [Troubleshooting](troubleshooting.md) — 串口、虚拟端口、graph validation、remote TCP、script、protocol 与 MCP 的问题导向排查。

## Source-of-truth map

这些文档刻意从源码事实出发，避免手写文档漂移：

- Node provider catalog: `frontend/src/serial/graph/serialGraph.ts` 与 `internal/modules/mcpserver/server.go`。
- Demo workspace IDs: `frontend/src/workspace/demoWorkspaces.ts`。
- Runtime behavior: `internal/modules/serial/graph_runtime.go`。
- Workspace snapshot kinds: `frontend/src/workspace/workspaceSnapshot.ts`。
- MCP tools and protocol template export shape: `internal/modules/mcpserver/server.go`。
- Test/CI commands: `Makefile`、`frontend/package.json`、`.github/workflows/ci.yml`、`scripts/check-go-coverage.sh`。
