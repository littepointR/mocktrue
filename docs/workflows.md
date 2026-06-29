# Workflows

PortWeave 的核心工作流不是“打开一个串口终端”或“做一个通用仪表盘”，而是把串口端点、协议模拟、脚本处理和自动化验证组织成可重复的 topology。可以按四层理解：observe、demo、build、automate。

## 1. Observe: inspect bytes and runtime evidence

Observe 层用于回答“有没有数据、数据从哪里来、在哪里丢了”。

常见检查面：

- Serial endpoint buffers：物理、虚拟、远端 raw TCP、协议节点可以保存可查询的字节缓冲。
- Monitor frames：`serial.monitor` 把一路 bytes 记录成监控帧，便于查看 text/hex/dec/oct/bin 等显示形式。
- Protocol frames：Modbus/FECbus 节点记录请求、响应、解析错误和帧显示。
- Script analyzer fields：`serial.script.analyzer` 通过 `field(name, value)` 产出字段。
- Node status/counters：节点状态、RX/TX counters、frame count 能定位运行时问题。

Observe 不一定需要外部硬件。内置 no-hardware demos 会用 `serial.script.generator`、`serial.virtual` 或 graph-owned raw TCP loopback 产生数据。

## 2. Demo: load built-in workspaces

Demo 层用于快速看到可运行拓扑，而不用先手工创建节点。

保守 UI 路径：Settings -> Global / 全局设置 -> 示例配置 -> 选择 demo -> 加载示例。加载后的 demo 会作为只读拓扑图标签页打开；它不会自动启动 runtime，也不会预先打开真实串口资源。

推荐顺序：

1. `serial-observability-demo`：过滤、fan-out、monitor frames 和 endpoint buffer。
2. `remote-serial-demo`：图内 raw TCP server/client loopback。
3. `modbus-demo` 或 `fecbus-demo`：协议 master/slave 模拟。
4. `full-workspace-demo`：浏览完整工作区状态和主要节点类型。

完整说明见 [Examples](examples.md)。

## 3. Build: compose custom serial topologies

Build 层用于把节点按真实调试意图连起来。

当前 graph 连接模型：

- Output ports may fan out: 一个输出可以连接多个下游输入。
- Input ports are single-source unless marked `multiple`: 当前 provider catalog 中的普通输入不能被多个输出同时连接。
- Directed cycles are rejected: 直接有向环会被拒绝。
- Protocol response handling has a runtime exception pattern: Modbus/FECbus master 的 `rx` 终端输入可用于响应返回，不必画出会造成普通 cycle 的直接环。
- Resource owners must be unique inside one graph: 物理/虚拟串口按 `portName` 去重，remote raw TCP 按 normalized endpoint + role 去重。

常见构图模式：

```text
serial.script.generator.out -> serial.virtual.tx
serial.virtual.rx       -> serial.monitor.in
```

```text
serial.script.generator.out -> serial.virtual.tx
serial.virtual.rx       -> serial.filter.in -> serial.monitor.in
                   \----> serial.filter.in -> serial.monitor.in
```

```text
serial.modbus.master.tx -> serial.virtual.tx
serial.virtual.rx       -> serial.modbus.slave.rx
serial.modbus.slave.tx  -> serial.modbus.master.rx
serial.modbus.slave.tx  -> serial.monitor.in
```

详见 [Serial Graph Node Catalog](serial-graph-node-catalog.md) 与 [Serial Graph Runtime Flow](serial-graph-runtime-flow.md)。

## 4. Automate: validate/start/query/stop through MCP

Automate 层用于让 agent、脚本或 CI 对 graph 做可重复验证。

推荐 MCP 顺序：

1. `serial_graph_provider_catalog` — 读取当前节点 provider 和连接规则。
2. `serial_graph_demo_catalog` — 选择 read-only backend template。
3. `serial_graph_demo_template` — 取得 nodes/edges。
4. `serial_graph_validate` — 在启动任何资源前验证拓扑。
5. `serial_graph_start` — 启动 graph runtime。
6. `serial_graph_status` — 查询运行状态和节点 counters。
7. `serial_graph_send` — 如需，向可写节点注入测试数据。
8. `serial_graph_query_node_buffer` / `serial_graph_query_node_frames` — 收集证据。
9. `serial_graph_stop` — 停止并释放 graph-owned resources。
10. 必要时使用 `serial_cleanup_virtual` 清理虚拟资源，但这是 destructive/cleanup 操作。

MCP 默认是本地 HTTP MCP server（默认 `127.0.0.1:39391/mcp`），不应暴露到不可信网络。详见 [MCP API and Recipes](mcp-api.md) 与 [AI Automation Guide](ai-automation.md)。

## Suggested paths by scenario

### I have a serial device and need to check output

1. 先用 `serial_enumerate_ports` 或 UI 串口列表确认设备可见。
2. 用 `serial.physical` 或串口终端打开真实端口。
3. 用 monitor branch 或 node buffer 观察字节。
4. 如果要自动化，先 validate graph，再启动，并避免对未知设备发送写操作。

### I need to simulate a device

1. 从 `serial-observability-demo`、`modbus-demo` 或 `fecbus-demo` 开始。
2. 用 `serial.virtual` 创建 graph-owned 虚拟端点。
3. 用 script/protocol slave 节点产生响应。
4. 用 monitor/protocol frames 验证模拟行为。

### I need to debug Modbus or FECbus

1. 用对应 demo 加载 master/slave topology。
2. 检查 master `tx`、slave `rx`、slave `tx`、master `rx` 的路径。
3. 用 protocol frames 查看 function code、address、quantity、status 或 parse error。
4. 不要为了“看起来活跃”添加孤立 loop branch；真实请求/响应路径应可解释。

### I need an agent or CI to validate behavior

1. 从 MCP catalog/template 开始，不手写未知 tool name。
2. validate before start。
3. 使用 localhost/virtual/no-hardware template。
4. 查询 status/buffer/frames 作为验收证据。
5. 停止 graph 并清理资源。

## What PortWeave is not optimizing for

PortWeave 不是一个 dashboard widget 克隆项目。它鼓励 topology-native debugging surfaces：node inspectors、edge traffic overlays、protocol frame tables、scenario timelines、operation logs、lightweight metrics 和 validation reports。架构方向见 [Architecture Decisions](development/architecture-decisions.md)。
