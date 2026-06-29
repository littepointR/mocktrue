# Serial Graph Node Catalog

This catalog documents the current PortWeave serial graph provider catalog. Source-of-truth files are `frontend/src/serial/graph/serialGraph.ts`, `internal/modules/mcpserver/server.go`, and `internal/modules/serial/graph_runtime.go`.

## Reading this catalog

For each node, check:

- Type string: stable provider ID used by graph documents and MCP.
- Category/title: frontend provider grouping.
- Ports: input/output handle IDs and kind.
- Resource ownership: whether graph start opens or claims an external resource.
- Key config: documented defaults that appear in current provider metadata.
- Runtime observations: buffers, frames, counters, status, and common evidence.

## Connection model

Current port kinds are `bytes`, `frame`, `registers`, `status`, and `control`. The current provider catalog uses byte ports for the documented nodes.

Rules:

- Output ports can fan out to multiple downstream inputs.
- Input ports are single-source unless the input marks `multiple`.
- Edge port kinds must match.
- Ordinary directed cycles are rejected.
- A protocol master `rx` terminal input can receive its response path without invalidating the request/response graph pattern.
- Resource-owning nodes cannot duplicate their resource identity in one graph.

## Current provider list

- `serial.physical`
- `serial.virtual`
- `serial.remote`
- `serial.bridge`
- `serial.monitor`
- `serial.filter`
- `serial.script.transform`
- `serial.script.generator`
- `serial.script.analyzer`
- `serial.modbus.master`
- `serial.modbus.slave`
- `serial.fecbus.master`
- `serial.fecbus.slave`

## Nodes

### `serial.physical`

- Title/category: 物理串口 / 串口.
- Purpose: open and use a real system serial port.
- Inputs: `tx` (`bytes`) writes to the port.
- Outputs: `rx` (`bytes`) emits bytes read from the port.
- Resource owner: yes, resource key `portName`.
- Key config defaults: `portName`, `baudRate = 115200`, `dataBits = 8`, `stopBits = 1`, `parity = none`, `flowMode = none`, `readBufKB = 32`, endpoint display fields.
- Runtime observations: status, resource ID, RX/TX counters, endpoint buffer, errors.
- Common mistakes: using a port already owned by another node/session, wrong OS port name, wrong line parameters, or writing to a production device without operator intent.

### `serial.virtual`

- Title/category: 虚拟串口 / 串口.
- Purpose: create and manage a graph-owned virtual serial endpoint.
- Inputs: `tx` (`bytes`).
- Outputs: `rx` (`bytes`).
- Resource owner: yes, resource key `portName`.
- Key config defaults: same serial defaults as `serial.physical`, with `portName = portweave-vport` before demos override it.
- Runtime observations: generated endpoint/resource, buffer, counters, status.
- Recommended demos: `serial-open-demo`, `virtual-port-demo`, `serial-observability-demo`, `full-workspace-demo`.
- Common mistakes: reusing the same `portName` in one graph or expecting demo snapshots to pre-create OS virtual ports before runtime start.

### `serial.remote`

- Title/category: 远端串口 / 串口.
- Purpose: connect a graph to a raw TCP serial endpoint or create an in-graph raw TCP server/client loopback.
- Inputs: `tx` (`bytes`) writes to TCP connection.
- Outputs: `rx` (`bytes`) emits bytes read from TCP.
- Resource owner: yes, endpoint identity + role.
- Key config defaults: `protocol = raw-tcp`, `role = client`, `host`, `port = 3001`, `connectTimeoutMs = 3000`, `writeTimeoutMs = 3000`, `reconnect = true`, `reconnectIntervalMs = 1000`, `allowStartDisconnected = false`, `readBufKB = 32`, serial metadata/display fields.
- Runtime observations: `running`, `reconnecting`, or `error`; resource ID such as `raw-tcp://127.0.0.1:3001`; endpoint buffer; counters.
- Recommended demo: `remote-serial-demo`.
- Common mistakes: putting `tcp://` in `host`, exposing raw TCP publicly, assuming raw TCP negotiates baud/parity, or duplicating the same endpoint + role in one graph.
- More: [Remote Serial Graph Node](remote-serial-node.md).

### `serial.bridge`

- Title/category: 串口桥接 / 串口.
- Purpose: route two byte streams bidirectionally.
- Inputs: `a-in` (`bytes`), `b-in` (`bytes`).
- Outputs: `a-out` (`bytes`), `b-out` (`bytes`).
- Resource owner: no.
- Key config defaults: none.
- Runtime behavior: data entering `a-in` exits `b-out`; data entering `b-in` exits `a-out`.
- Recommended demos: `bridge-demo`, `full-workspace-demo`.
- Common mistakes: treating bridge as an OS port resource or adding serial config fields that runtime ignores.

### `serial.monitor`

- Title/category: 串口监控 / 工具.
- Purpose: observe one byte stream and record monitor frames.
- Inputs: `in` (`bytes`).
- Outputs: none.
- Resource owner: no.
- Key config defaults: `displayMode = hex`.
- Runtime observations: monitor frames, frame count, status.
- Recommended demos: most demos include monitor branches.
- Common mistakes: expecting monitor to forward bytes downstream; it is an input-only sink.

### `serial.filter`

- Title/category: 过滤器 / 工具.
- Purpose: pass/drop bytes by plain text, regex, or expression matching.
- Inputs: `in` (`bytes`).
- Outputs: `out` (`bytes`).
- Resource owner: no.
- Key config defaults: `mode = plain`, `expression = ""`, `caseSensitive = false`, `wholeWord = false`.
- Runtime observations: matching bytes on `out`, counters/status, downstream monitor frames.
- Recommended demos: `serial-observability-demo`, `full-workspace-demo`.
- Common mistakes: regex syntax mismatch, case sensitivity surprises, or missing a downstream monitor to prove pass/drop behavior.

### `serial.script.transform`

- Title/category: 脚本转换 / 脚本.
- Purpose: process input bytes with a safe script and emit transformed bytes.
- Inputs: `in` (`bytes`).
- Outputs: `out` (`bytes`).
- Resource owner: no.
- Key config defaults: `script = output.bytes(input.bytes())`, `timeoutMs = 50`, `maxOutputBytes = 65536`, `maxStateBytes = 262144`, `onError = mark-error-and-drop`, `encoding = utf-8`, `autoRun = false`, `intervalMs = 1000`, `displayMode = hex`.
- Runtime observations: transformed output, script errors/timeouts, counters.
- Recommended demos: `script-transform-demo`, `full-workspace-demo`.
- Common mistakes: empty script, unbounded output, relying on unavailable global APIs, or forgetting that transform needs input bytes.

### `serial.script.generator`

- Title/category: 脚本生成 / 脚本.
- Purpose: emit generated bytes on graph start or interval.
- Inputs: none.
- Outputs: `out` (`bytes`).
- Resource owner: no.
- Key config defaults: script defaults plus `script = output.text("tick", "utf-8")`, `autoRun = true`.
- Runtime observations: generated output, interval behavior, script errors/timeouts.
- Recommended demos: all no-hardware demos use generator nodes.
- Common mistakes: assuming generator needs an input edge, using too short intervals, or producing more than `maxOutputBytes`.

### `serial.script.analyzer`

- Title/category: 脚本分析 / 脚本.
- Purpose: parse input bytes and record fields/errors.
- Inputs: `in` (`bytes`).
- Outputs: none.
- Resource owner: no.
- Key config defaults: script defaults plus `script = field("length", input.bytes().length)`.
- Runtime observations: analyzer fields and errors; no downstream bytes.
- Recommended demos: `script-analyzer-demo`, `full-workspace-demo`.
- Common mistakes: expecting analyzer to forward bytes or omitting `field()` calls.

### `serial.modbus.master`

- Title/category: Modbus 主站 / 协议.
- Purpose: build Modbus RTU/ASCII master requests and observe responses.
- Inputs: `rx` (`bytes`) receives responses.
- Outputs: `tx` (`bytes`) sends requests.
- Resource owner: no.
- Key config defaults: `mode = rtu`, `unitIds = 1`, `addressMode = zero-based`, `functionCode = 3`, `address = 0`, `quantity = 1`, `value = 0`, plus coil/register value helpers.
- Runtime observations: request/response frames, parse errors, counters.
- Recommended demos: `modbus-demo`, `serial-graph-demo`, `full-workspace-demo`.
- Common mistakes: drawing an ordinary cycle instead of using the documented response path, mismatching Unit ID/mode, or expecting the graph node to replace all editable Modbus session panel features.

### `serial.modbus.slave`

- Title/category: Modbus 从站 / 协议.
- Purpose: respond to Modbus RTU/ASCII requests with protocol-valid default/zero values.
- Inputs: `rx` (`bytes`) receives requests.
- Outputs: `tx` (`bytes`) sends responses.
- Resource owner: no.
- Key config defaults: `mode = rtu`, `unitIds = 1`.
- Runtime observations: request/response frames and errors.
- Recommended demo: `modbus-demo`.
- Common mistakes: adding unsupported `addressMode` to slave graph config, or expecting full editable multi-Unit data model behavior in the graph node rather than the Modbus session panel.

### `serial.fecbus.master`

- Title/category: FECbus 主控 / 协议.
- Purpose: build FECbus controller requests and parse responses.
- Inputs: `rx` (`bytes`).
- Outputs: `tx` (`bytes`).
- Resource owner: no.
- Key config defaults: `sourceAddress = 1`, `targetAddress = 2`, `priority = 3`, `messageNumber = 1`, `groupNumber = 0`, `functionCode = 44`, `dataHex = ""`, display fields.
- Runtime observations: FECbus frames and parse errors.
- Recommended demos: `fecbus-demo`, `full-workspace-demo`.
- Common mistakes: invalid hex data, mismatched address, or no downstream path to a slave/monitor.

### `serial.fecbus.slave`

- Title/category: FECbus 从机 / 协议.
- Purpose: simulate FECbus device status and responses.
- Inputs: `rx` (`bytes`).
- Outputs: `tx` (`bytes`).
- Resource owner: no.
- Key config defaults: `address = 2`, `defaultStatus = 10`, `autoStatusAnswer = true`, display fields.
- Runtime observations: response frames, status-related output, parse errors.
- Recommended demos: `fecbus-demo`, `full-workspace-demo`.
- Common mistakes: address mismatch or expecting output without a request path.

## Removed providers

The current provider catalog does not include the removed early-planning helper nodes from the old remote examples. If you see old diagrams that rely on separate sender, receiver, tap, or tee providers, treat them as stale and rebuild them with current nodes such as `serial.script.generator`, `serial.monitor`, direct output fan-out, and `serial.filter`.
