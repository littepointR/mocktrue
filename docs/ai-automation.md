# AI Automation Guide

This guide describes safe PortWeave automation for agents and scripts. It focuses on current MCP tools, no-hardware validation paths, evidence collection, and resource cleanup.

## Recommended agent workflow

1. Inspect catalogs before acting:
   - `serial_graph_provider_catalog`
   - `serial_graph_demo_catalog`
   - `protocol_template_catalog` when protocol parser templates are involved
2. Prefer safe no-hardware templates first:
   - `serial-observability-filter-logging`
   - `serial-remote-raw-tcp` on localhost
3. Generate/load template nodes and edges.
4. Validate before start with `serial_graph_validate`.
5. Start only after validation succeeds.
6. Query status, buffers, and frames before making claims.
7. Stop the graph with `serial_graph_stop`.
8. Use cleanup/delete/clear/reset only when the operator intends it.

Evidence before claims: a graph is not “working” until status/buffer/frame output proves expected behavior.

## Safety tiers

### Safe/read-only

These inspect state or validate config without opening resources or transmitting bytes:

- `serial_graph_provider_catalog`
- `serial_graph_demo_catalog`
- `serial_graph_demo_template`
- `serial_graph_validate`
- `serial_graph_status`
- `serial_graph_query_node_buffer`
- `serial_graph_query_node_frames`
- `protocol_template_catalog`
- `protocol_template_describe`
- `serial_enumerate_ports`
- list/query tools such as `serial_list_open_ports`, `serial_list_virtual_ports`, `serial_list_bridges`, `serial_list_monitors`, `modbus_list_sessions`, `fecbus_list_sessions`, and frame/query tools.

### Requires operator intent

These create live resources or start runtime behavior:

- `serial_open_port`
- `serial_create_virtual_port`
- `serial_create_virtual_pair`
- `serial_create_bridge`
- `serial_start_monitor`
- `serial_graph_start`
- `modbus_open_session`
- `modbus_start_slave`
- `fecbus_open_session`
- `fecbus_start_slave`

Use safe demos and localhost first. For physical hardware, confirm the device and intended operation.

### Device-gated writes

These can transmit bytes or protocol writes:

- `serial_send`
- `serial_graph_send`
- `modbus_master_request` when it writes to a real device.
- `modbus_read_registers`, `modbus_scan_unit_ids`, and `modbus_scan_registers` when targeting a real bus.
- `fecbus_send_request`.

Do not send to unknown physical or remote devices without explicit operator intent.

### Destructive/cleanup

These delete resources, clear data, reset counters, or remove state:

- `serial_close_port`
- `serial_delete_virtual_port`
- `serial_delete_virtual_pair`
- `serial_delete_bridge`
- `serial_cleanup_virtual`
- `serial_stop_monitor`
- `serial_delete_monitor`
- `serial_clear_monitor_frames`
- `serial_reset_counters`
- `serial_graph_stop`
- `serial_graph_clear_node_buffer`
- `serial_graph_reset_node_counters`
- Modbus/FECbus close/stop/remove/clear/update tools when they affect active sessions.

Stopping a graph you started is expected cleanup. Broad cleanup tools should be used more cautiously.

## Safe no-hardware recipe

Use the observability template:

1. `serial_graph_demo_template` with `id = serial-observability-filter-logging`.
2. `serial_graph_validate` with returned nodes/edges.
3. `serial_graph_start` with returned start arguments.
4. Wait briefly or query status until counters move.
5. `serial_graph_query_node_buffer` on the virtual node.
6. `serial_graph_query_node_frames` on monitor nodes.
7. `serial_graph_stop`.

Expected evidence:

- Runtime status is running while active.
- Generator/virtual/filter/monitor counters change.
- Monitor frames contain generated test traffic.

## Safe remote loopback recipe

Use the remote template on localhost:

1. `serial_graph_demo_template` with `id = serial-remote-raw-tcp`, `remote_host = 127.0.0.1`, `remote_port = 3001`.
2. Validate.
3. Start.
4. Query status for server/client remote nodes.
5. Query server/client monitor frames.
6. Stop.

This does not require ser2net or an external TCP server because the graph contains both server and client nodes.

## Physical-device cautious recipe

When a real serial device is involved:

1. Use `serial_enumerate_ports` to identify the port.
2. Confirm operator intent to open or send to that specific port.
3. Prefer read-only monitor/open behavior before writes.
4. Use conservative baud/parity/flow settings matching the device.
5. Add a monitor branch before protocol parsers.
6. Query evidence before and after any write.
7. Stop/close resources after the test.

Avoid scan/write tools unless the bus owner expects that traffic.

## Raw TCP safety

`serial.remote` raw TCP has no built-in authentication, authorization, encryption, or serial parameter negotiation.

Agents should not:

- Expose raw TCP serial endpoints to the public Internet.
- Bind PortWeave MCP or raw TCP endpoints to broad/untrusted interfaces.
- Store SSH/VPN/TLS credentials in graph configs.
- Treat raw TCP as a secure tunnel.

Use trusted LAN, VPN, SSH tunnel, or TLS wrapper outside PortWeave.

## Tool safety matrix

| Category | Examples | Default automation stance |
| --- | --- | --- |
| Catalog/validate | `serial_graph_provider_catalog`, `protocol_template_catalog`, `serial_graph_validate` | Safe |
| Status/query/list | `serial_graph_status`, list/query tools | Safe |
| Start/open/create | `serial_graph_start`, `serial_open_port`, create virtual/bridge | Requires operator intent, safe for no-hardware demos |
| Send/write/scan | `serial_send`, `serial_graph_send`, protocol requests/scans | Device-gated |
| Stop own runtime | `serial_graph_stop` for a graph started by the recipe | Expected cleanup |
| Delete/clear/reset/global cleanup | `*_delete_*`, `*_clear_*`, `*_reset_*`, `serial_cleanup_virtual` | Destructive/cleanup |

## What agents should not do

- Do not invent tool names or transports.
- Do not skip provider catalog inspection before constructing graphs.
- Do not start a graph that failed validation.
- Do not claim a graph works without status/buffer/frame evidence.
- Do not write to physical/remote devices without explicit operator intent.
- Do not expose MCP or raw TCP to untrusted networks.
- Do not store secrets in graph configs, screenshots, docs, or handoffs.
- Do not leave started graph runtimes running after validation recipes finish.

## Handoff checklist

When reporting automation results, include:

- Template/tool IDs used.
- Graph runtime ID.
- Validation result.
- Start/stop result.
- Status summary.
- Buffer/frame evidence.
- Any skipped device-gated actions.
- Cleanup performed.

Use `[REDACTED]` for any credentials or private connection material if they appear in operator-provided context.
