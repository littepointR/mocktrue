# MCP API and Recipes

PortWeave exposes local automation through an MCP server. The current source-of-truth is `internal/modules/mcpserver/server.go`; this page documents the current 64 tool names and safe graph recipes.

## What MCP exposes

MCP exposes serial resource operations, read-only protocol template catalog access, serial graph runtime operations, monitor frame access, Modbus operations, and FECbus operations. It is intended for local automation, agent-driven validation, and repeatable no-hardware demos.

Default local config from `internal/core/config/config.go`:

```text
enabled = true
host = 127.0.0.1
port = 39391
path = /mcp
allow_local_origins = true
```

Do not expose MCP to untrusted networks. MCP tools can create resources, send bytes, delete/clear state, and operate physical/remote devices.

## Safety boundary

- MCP is local automation, not a public API gateway.
- Raw TCP serial endpoints have no built-in authentication, encryption, or serial parameter negotiation.
- Do not store passwords, tokens, tunnel credentials, or private connection strings in graph configs.
- Validate graph configs before starting resources.
- Stop graph runtimes after collecting evidence.
- Treat delete/clear/reset/cleanup tools as destructive.
- Treat send/write tools to physical or remote nodes as device-gated actions.

## Tool groups

### Serial resource tools

- `serial_enumerate_ports`
- `serial_open_port`
- `serial_close_port`
- `serial_list_open_ports`
- `serial_send`
- `serial_query_buffer`
- `serial_reset_counters`
- `serial_create_virtual_port`
- `serial_delete_virtual_port`
- `serial_list_virtual_ports`
- `serial_create_virtual_pair`
- `serial_delete_virtual_pair`
- `serial_list_virtual_pairs`
- `serial_create_bridge`
- `serial_delete_bridge`
- `serial_list_bridges`
- `serial_cleanup_virtual`

### Serial graph tools

- `serial_graph_provider_catalog`
- `serial_graph_demo_catalog`
- `serial_graph_demo_template`
- `serial_graph_validate`
- `serial_graph_start`
- `serial_graph_stop`
- `serial_graph_status`
- `serial_graph_send`
- `serial_graph_query_node_buffer`
- `serial_graph_query_node_frames`
- `serial_graph_clear_node_buffer`
- `serial_graph_reset_node_counters`

### Protocol template tools

- `protocol_template_catalog`
- `protocol_template_describe`

### Monitor frame tools

- `serial_start_monitor`
- `serial_stop_monitor`
- `serial_delete_monitor`
- `serial_list_monitors`
- `serial_query_monitor_frames`
- `serial_clear_monitor_frames`

### Modbus tools

- `modbus_open_session`
- `modbus_close_session`
- `modbus_list_sessions`
- `modbus_master_request`
- `modbus_start_slave`
- `modbus_stop_slave`
- `modbus_update_slave_data`
- `modbus_add_slave_unit`
- `modbus_remove_slave_unit`
- `modbus_list_slave_units`
- `modbus_update_slave_unit_data`
- `modbus_read_registers`
- `modbus_scan_unit_ids`
- `modbus_scan_registers`

### FECbus tools

- `fecbus_function_catalog`
- `fecbus_open_session`
- `fecbus_close_session`
- `fecbus_list_sessions`
- `fecbus_send_request`
- `fecbus_start_slave`
- `fecbus_stop_slave`
- `fecbus_update_slave_state`
- `fecbus_list_slave_units`
- `fecbus_add_slave_unit`
- `fecbus_remove_slave_unit`
- `fecbus_query_frames`
- `fecbus_clear_frames`

## Serial graph recipe

Recommended safe flow:

1. `serial_graph_provider_catalog` — inspect current node types, ports, defaults, and resource ownership.
2. `serial_graph_demo_catalog` — choose a read-only backend template.
3. `serial_graph_demo_template` — get nodes, edges, and usage examples.
4. `serial_graph_validate` — validate before start.
5. `serial_graph_start` — start resources.
6. `serial_graph_status` — check runtime and node counters/status.
7. `serial_graph_send` — optionally inject test bytes.
8. `serial_graph_query_node_buffer` — collect endpoint/protocol bytes.
9. `serial_graph_query_node_frames` — collect monitor/protocol/script analyzer frames.
10. `serial_graph_stop` — stop runtime and release resources.

## Demo templates

Current backend template IDs:

- `serial-observability-filter-logging`
- `serial-remote-raw-tcp`

These are MCP templates, not frontend workspace demo IDs. Frontend demo IDs such as `serial-observability-demo` and `remote-serial-demo` are documented in [Examples](examples.md).

MCP templates cover graph config plus backend runtime buffers/status. They do not persist or synthesize frontend UI operation-log state.

## Protocol template catalog tools

Protocol template tools are read-only. They expose the current parser template registry without starting resources or changing parser state.

Use `protocol_template_catalog` to list template names, descriptions, and kinds:

```json
{
  "tool": "protocol_template_catalog",
  "arguments": {}
}
```

Current template names are documented in [Protocol and Script Templates](protocol-script-templates.md). Use returned names rather than hard-coding future guesses.

Use `protocol_template_describe` to fetch a single template config by name:

```json
{
  "tool": "protocol_template_describe",
  "arguments": { "name": "AA55 自定义帧" }
}
```

Visual templates return a summarized config including `header_hex`, optional length/checksum details, fields, and frame length bounds. Script templates return their parser script text. These are export/inspection helpers; they do not register new templates and do not replace `serial_graph_validate` for runtime graph configs.

## Example: safe observability template

1. Call `serial_graph_demo_template`:

```json
{
  "id": "serial-observability-filter-logging",
  "graph_id": "serial-observability-mcp-demo",
  "port_name": "portweave-mcp-demo"
}
```

2. Call `serial_graph_validate` with the returned `nodes` and `edges`.
3. Call `serial_graph_start` with the returned start arguments.
4. Query status and monitor frames.
5. Stop the graph.

Expected shape:

```text
serial.script.generator.out -> serial.virtual.tx
serial.virtual.rx -> serial.filter.in -> serial.monitor.in
serial.virtual.rx -> serial.filter.in -> serial.monitor.in
serial.virtual.rx -> serial.filter.in -> serial.monitor.in
```

## Example: remote raw TCP loopback template

1. Call `serial_graph_demo_template`:

```json
{
  "id": "serial-remote-raw-tcp",
  "graph_id": "serial-remote-raw-tcp-mcp-demo",
  "remote_host": "127.0.0.1",
  "remote_port": 3001,
  "allow_start_disconnected": false
}
```

2. Validate with `serial_graph_validate`.
3. Start with `serial_graph_start`.
4. Query `serial_graph_status` and node buffers/frames.
5. Stop with `serial_graph_stop`.

Expected shape:

```text
serial.script.generator.out -> serial.remote(client).tx
serial.remote(server).rx -> serial.monitor.in
serial.remote(client).rx -> serial.monitor.in
```

Security: raw TCP has no authentication or encryption. Keep this on localhost/trusted networks.

## Example: query evidence

After start, query runtime state:

```json
{
  "tool": "serial_graph_status",
  "arguments": { "graph_id": "serial-remote-raw-tcp-mcp-demo" }
}
```

Query node buffer:

```json
{
  "tool": "serial_graph_query_node_buffer",
  "arguments": {
    "graph_id": "serial-remote-raw-tcp-mcp-demo",
    "node_id": "b-remote-client",
    "offset": 0,
    "length": 256
  }
}
```

Query node frames:

```json
{
  "tool": "serial_graph_query_node_frames",
  "arguments": {
    "graph_id": "serial-remote-raw-tcp-mcp-demo",
    "node_id": "client-monitor",
    "offset": 0,
    "limit": 20
  }
}
```

Exact node IDs depend on the selected template response; use returned IDs rather than guessing.

## Troubleshooting MCP calls

### Cannot connect

- Confirm PortWeave is running and MCP is enabled.
- Confirm endpoint is local host/port/path.
- Confirm the MCP client sends correct HTTP/MCP headers.
- Check origin guard behavior if calling from a browser-like client.

### Tool name not found

- Re-read `serial_graph_provider_catalog`, `protocol_template_catalog`, and source docs for current tool names.
- Do not assume future or invented tool names.

### Graph start fails

- Run `serial_graph_validate` first.
- Check duplicate resources, invalid remote config, and missing handles.
- Start from a backend demo template before custom nodes.

### No data after send/start

- Query upstream node buffers first.
- Query monitor/protocol frames downstream.
- Check node status and counters.
- Stop and restart only after capturing errors.

## Related docs

- [AI Automation Guide](ai-automation.md)
- [Serial Graph Runtime Flow](serial-graph-runtime-flow.md)
- [Remote Serial Graph Node](remote-serial-node.md)
- [Troubleshooting](troubleshooting.md)
