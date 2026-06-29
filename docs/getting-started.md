# Getting Started

本指南提供一条不依赖外部串口硬件的 PortWeave 上手路径：启动开发应用、加载内置示例工作区、启动串口拓扑图，并观察节点缓冲区、监控帧和运行状态。

## What you will run

你会运行本地 Wails 开发应用，并从全局设置加载一个只读 demo graph tab。推荐从 `serial-observability-demo` 或 `remote-serial-demo` 开始：

- `serial-observability-demo`：脚本生成器 -> 虚拟串口 -> 多个过滤器 -> 监控节点；不需要外部硬件。
- `remote-serial-demo`：图内 raw TCP server/client loopback；不需要外部 ser2net 或额外进程。

这些 demo 只是快照配置。加载后仍需要在拓扑图界面启动 graph runtime，运行时资源不会预先保存在 workspace 文件里。

## Requirements

以仓库当前配置为准：

- Go `1.26.0`，见 `go.mod`。
- Node.js `22`，与 GitHub Actions CI 一致。
- pnpm `10.32.1`，与 CI 的 `pnpm/action-setup` 配置一致。
- Wails CLI v3 alpha。当前项目使用 `github.com/wailsapp/wails/v3 v3.0.0-alpha2.103`。

安装 Wails CLI：

```bash
go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-alpha2.103
```

安装前端依赖：

```bash
cd frontend
pnpm install
```

## Start the development app

从仓库根目录启动桌面应用：

```bash
wails3 dev -config ./build/config.yml -port 9245
```

也可以使用 Taskfile：

```bash
wails3 task dev
```

如果 Wails dev 子进程找不到 `go`、`node`、`pnpm` 或 `wails3`，先确认启动 shell 的 `PATH` 对子进程可见。Windows 相关说明见 [Windows Serial Testing](development/windows-serial-testing.md)。

## Load a demo workspace

当前 UI 的保守路径是：

1. 打开 Settings。
2. 进入 Global / 全局设置面板。
3. 在“示例配置”下拉框中选择一个示例。
4. 点击“加载示例”。

源码依据：`frontend/src/settings/views/global/GlobalSettingsPanel.vue` 使用 `workspaceFile.listDemos()` 生成选项，点击“加载示例”后调用 `workspaceFile.loadDemo()`。加载后的示例会作为只读拓扑图标签页打开；修改后可在拓扑图工具栏另存为文件。

推荐新用户先试：

- `serial-observability-demo` / “串口过滤与日志演示”。
- `remote-serial-demo` / “远端串口演示”。
- `full-workspace-demo` / “完整工作区演示”，用于浏览所有主要节点类型。

完整列表见 [Examples](examples.md)。

## Start a serial graph

加载示例后，切换到 Serial 模块的拓扑图界面。当前 graph store 会在启动前运行 `validateGraph()`；如果图无效，启动会被拒绝并显示 validation errors。

保守操作步骤：

1. 打开加载出来的 graph tab。
2. 检查节点和连线是否存在。
3. 使用拓扑图工具栏上的启动运行时操作启动当前 graph。
4. 等待节点状态进入 `running`；remote client 在允许断线启动时也可能显示 `reconnecting`。
5. 观察节点 inspector、buffer/frame 视图或状态区域。
6. 完成后停止 graph runtime，让 graph-owned serial/virtual/remote resources 被释放。

不要假设 demo 会自动启动。内置 demo 只创建 workspace snapshot；graph runtime 由用户或 MCP 工具显式启动。

## What to inspect first

启动 graph 后，按以下顺序检查：

1. Validation errors：如果存在，先修复节点配置或连接。
2. Node status：`idle`、`running`、`reconnecting`、`error`。
3. Node buffer：serial endpoint、virtual、remote、protocol 节点会记录可查询的 bytes。
4. Monitor/protocol/script analyzer frames：监控节点、协议节点、script analyzer 会产出结构化帧或字段显示。
5. Counters：RX/TX byte counters 和 frame count 可用于判断数据是否到达。
6. Operation log / UI 状态：前端可显示操作过程，但 MCP backend templates 不保存 UI operation-log state。

## No-hardware recipes

### Serial observability

1. 选择 `serial-observability-demo`。
2. 加载后启动 graph。
3. 观察虚拟串口节点的 endpoint buffer。
4. 观察 plain / regex / expression filter 后面的 monitor frames。

该 demo 使用 `serial.script.generator` 自动产生 `TEMP=42 STATUS OK`，再通过 `serial.virtual` 的 `rx` 输出 fan out 到多个过滤器。

### Remote raw TCP loopback

1. 选择 `remote-serial-demo`。
2. 加载后启动 graph。
3. 观察 server/client 两个 `serial.remote` 节点和对应 monitor branch。

该 demo 在图内创建 raw TCP server 和 client，默认使用 `127.0.0.1:3001`。raw TCP 没有认证、加密或串口参数协商；这只是本机可信 loopback 示例。详情见 [Remote Serial Graph Node](remote-serial-node.md)。

## Next steps

- 想理解每个节点：读 [Serial Graph Node Catalog](serial-graph-node-catalog.md)。
- 想理解运行时路由：读 [Serial Graph Runtime Flow](serial-graph-runtime-flow.md)。
- 想按场景选择 demo：读 [Examples](examples.md) 与 [Use Cases](use-cases.md)。
- 想让 agent 或 CI 自动验证 graph：读 [MCP API and Recipes](mcp-api.md) 与 [AI Automation Guide](ai-automation.md)。
- 启动失败或没数据：读 [Troubleshooting](troubleshooting.md)。
