# Use Cases

PortWeave 适合需要串口拓扑、协议模拟和自动化验证的嵌入式调试场景。它不是通用 telemetry dashboard；它的优势在于把端点、脚本、协议节点和 MCP 自动化放在同一个可验证 graph 中。

## No-hardware embedded UI/protocol development

当硬件还没到位、设备固件不稳定或 CI 环境没有串口设备时，可以使用内置 demos 和 graph-owned virtual resources：

- `serial.script.generator` 生成测试字节流。
- `serial.virtual` 提供 graph-owned 虚拟端点。
- `serial.filter` / `serial.monitor` 观察 expected traffic。
- Modbus/FECbus master/slave 节点模拟协议交互。

推荐起点：[Getting Started](getting-started.md) 与 [Examples](examples.md)。

## Virtual serial pair loopback testing

需要验证串口读写、桥接或端点 buffer 时，可使用虚拟串口节点和桥接节点：

```text
serial.script.generator.out -> serial.virtual.tx
serial.virtual.rx       -> serial.monitor.in
```

或：

```text
serial.virtual.rx -> serial.bridge.a-in
serial.bridge.b-out -> serial.monitor.in
```

内置 `virtual-port-demo`、`bridge-demo` 和 `serial-graph-demo` 都不要求外部硬件。

## Serial gateway / remote raw TCP lab debugging

`serial.remote` 把 raw TCP byte stream 接入 graph，适用于 ser2net、TCP-to-serial adapters、VPN gateway 或 SSH tunnel。当前支持 raw TCP client/server：

- Client：连接 `host:port`。
- Server：在 graph runtime 中监听 `host:port`。
- `tx` 输入写入 TCP connection。
- `rx` 输出将 TCP 收到的 bytes 发到下游。

安全边界很重要：raw TCP 没有认证、授权、加密或串口参数协商。不要公开暴露 raw TCP serial endpoint；优先使用可信 LAN、VPN 或 SSH tunnel。详见 [Remote Serial Graph Node](remote-serial-node.md)。

## Modbus RTU/ASCII master/slave simulation

PortWeave graph 中的 Modbus 节点适合做请求/响应模拟和帧可视化：

- `serial.modbus.master`：生成 RTU/ASCII master 请求，记录请求/响应帧。
- `serial.modbus.slave`：按 Unit ID 响应协议合法的默认/零值数据；完整可编辑数据模型仍在 Modbus session panel。
- `serial.virtual`：承载 master/slave 间的 no-hardware 数据路径。
- `serial.monitor`：旁路观察 raw bytes。

从 `modbus-demo` 或 `full-workspace-demo` 开始。

## FECbus controller/device simulation

FECbus 节点用于电气火灾监控相关帧的请求、应答和状态模拟：

- `serial.fecbus.master`：构建主控请求。
- `serial.fecbus.slave`：模拟从机地址、默认状态和自动状态应答。
- `serial.monitor`：观察应答帧。

从 `fecbus-demo` 开始。若使用真实设备，先在 no-hardware demo 中验证帧形状，再连接物理端口。

## Scripted frame generation/transformation/analysis

脚本节点适合做轻量协议 glue、测试数据生成和字段提取：

- `serial.script.generator`：无输入，按启动或定时器输出 bytes/text/hex。
- `serial.script.transform`：读取 `input.*`，通过 `output.*` 输出转换后的 bytes。
- `serial.script.analyzer`：读取 `input.*`，通过 `field()` 记录字段；不输出 bytes。

脚本运行在受限 Goja runtime 中，有 timeout、output bytes 和 state bytes 限制。不要把秘密、凭据或长期状态放进 graph config。

## Agent-driven MCP validation

PortWeave MCP 工具适合让 agent 或 CI 做 repeatable validation：

1. 读取 provider/template catalog。
2. 取得 no-hardware template。
3. validate before start。
4. 启动 graph。
5. 查询 status、node buffer、frames。
6. 停止 graph，必要时清理虚拟资源。

这种路径适合回归测试、协议变更验证、示例文档自检和 release smoke。详见 [MCP API and Recipes](mcp-api.md) 与 [AI Automation Guide](ai-automation.md)。

## Teaching and demos

PortWeave 也适合课堂、内部培训和 demo：

- 用 `serial-observability-demo` 展示 fan-out、filter 和 monitor frames。
- 用 `remote-serial-demo` 展示 raw TCP client/server loopback 与安全边界。
- 用 `modbus-demo` / `fecbus-demo` 展示协议请求/响应。
- 用 [Workspace and Graph Schema](workspace-graph-schema.md) 解释 workspace、graph tab、graph document 和 runtime snapshot 的区别。

## When not to use PortWeave

PortWeave 不应该被描述成这些东西：

- 高刷新、多 widget 的通用 telemetry dashboard 替代品。
- 未经授权的 raw TCP serial gateway 暴露工具。
- 对未知物理设备自动写入的默认安全工具。
- Qt/C++ 桌面栈迁移项目。
- gRPC-first API 项目。

当前架构方向见 [Architecture](architecture.md) 与 [Architecture Decisions](development/architecture-decisions.md)。
