# Serial Graph Runtime Flow

This page explains how a PortWeave serial graph moves from saved document state to a running runtime graph, how bytes flow through nodes, and what evidence the UI and MCP tools can observe.

## Document vs runtime

A `SerialGraphDocument` is authoring state. It has:

- `id` and `name`.
- `nodes` with `id`, `type`, position, config, and optional display name.
- `edges` with source/target node IDs and handle IDs.
- selected node/edge and node-tab UI state.

A runtime graph is created only when the user or MCP calls graph start. It owns resources, goroutines, TCP connections, buffers, frames, counters, and status. Stopping the runtime releases graph-owned resources; the document remains editable.

## Start sequence

```text
SerialGraphDocument
  -> normalize/clone document state
  -> validate nodes, edges, handles, resources, and remote config
  -> open resource-owning nodes
  -> start remote raw TCP listeners/dialers
  -> start protocol autoSend loops and script generators
  -> route bytes along edges
  -> record node buffers, frames, counters, status, and errors
  -> expose evidence to frontend stores and MCP queries
```

The frontend store validates with `validateGraph()` before calling `StartSerialGraph`. The backend validates again in `validateSerialGraphRuntimeRequest()` before creating resources.

## Validation rules

Current shared rules:

- Node IDs must be non-empty and unique.
- Node types must exist in the provider catalog.
- Edge IDs must be non-empty and unique.
- Edge source/target node IDs must exist.
- Edge source handle must be an output on the source provider.
- Edge target handle must be an input on the target provider.
- Port kinds must match.
- Output ports can fan out.
- Input ports are single-source unless the provider marks them `multiple`.
- Directed cycles are rejected.
- Edges that target terminal protocol master `rx` inputs are allowed as the supported request/response pattern for Modbus/FECbus master responses.
- Resource owners cannot duplicate the same resource key inside one graph.

## Resource ownership

Resource-owning nodes are responsible for live OS/network resources:

- `serial.physical`: resource key is `portName`.
- `serial.virtual`: resource key is `portName`.
- `serial.remote`: resource identity is normalized endpoint + role, for example `remote:127.0.0.1:3001:client` internally and `raw-tcp://127.0.0.1:3001` as resource ID.

`serial.bridge`, monitors, filters, scripts, and protocol nodes do not own external serial/remote endpoints directly.

## Routing model

Bytes move from an output handle to all connected input handles:

```text
source node output handle -> edge(s) -> target node input handle
```

Fan-out means one output can feed multiple downstream inputs:

```text
serial.virtual.rx -> serial.filter.in -> serial.monitor.in
                 \-> serial.filter.in -> serial.monitor.in
```

Inputs are single-source by default. If two different outputs target the same node input, validation reports `input already connected` unless that input explicitly supports multiple sources.

## Node processing

### Endpoint nodes

`serial.physical`, `serial.virtual`, and `serial.remote` expose:

- `tx` input: bytes written to the endpoint.
- `rx` output: bytes read from the endpoint.
- Endpoint buffer, counters, status, and resource ID.

### Bridge node

`serial.bridge` exposes two input/output pairs:

- `a-in` -> `b-out`.
- `b-in` -> `a-out`.

It routes bytes between two sides but does not open a physical resource itself.

### Monitor and filter nodes

`serial.monitor` records incoming bytes as monitor frames and has no output.

`serial.filter` reads bytes from `in`, evaluates `mode`/`expression`, and emits matching bytes from `out`.

### Script nodes

`serial.script.transform` reads `input.*` and emits bytes through `output.*`.

`serial.script.generator` has no input and emits bytes on startup or interval when configured.

`serial.script.analyzer` reads bytes and records fields/errors; it does not emit bytes.

Script runtime applies timeout, maximum output bytes, and maximum state bytes limits.

### Protocol nodes

`serial.modbus.master` and `serial.fecbus.master` emit requests from `tx` and receive responses on `rx`.

`serial.modbus.slave` and `serial.fecbus.slave` receive requests on `rx` and emit responses from `tx`.

Protocol master response paths use the terminal-input cycle exception so a realistic request/response flow can be modeled without an invalid ordinary directed cycle.

## Buffers, frames, counters, and status

Runtime evidence is intentionally visible:

- Node status: `idle`, `running`, `reconnecting`, `error`.
- Node counters: RX/TX bytes.
- Node buffer: endpoint/protocol bytes queryable by offset/length.
- Monitor frames: frame records created by monitor nodes.
- Protocol frames: parsed request/response/error records for Modbus/FECbus nodes.
- Script analyzer fields/errors.

MCP tools can query status, node buffers, node frames, clear buffers, and reset counters.

## Error handling

### Validation errors

Graph start should not proceed. Fix node type, handles, duplicate IDs, duplicate resource keys, remote config, or cycles first.

### Resource open failure

If a physical/virtual/remote resource cannot start, graph start fails unless a remote node is explicitly allowed to start disconnected.

### Remote disconnect

`serial.remote` can enter `reconnecting` when reconnect is enabled. Writes while disconnected fail instead of creating an unbounded offline queue.

### Script timeout/error

Script nodes mark error/drop behavior according to config and runtime policy. Check node status, node frames/fields, and operation logs.

### Protocol parse error

Protocol nodes can record parse errors in frame evidence. Inspect raw monitor bytes upstream when a parser reports unexpected data.

## Stop and cleanup

Stopping a graph cancels loops, closes open serial/TCP resources, and stops protocol autoSend/script generator loops. It does not delete the saved graph document.

Cleanup tools such as `serial_cleanup_virtual` are broader cleanup operations. Use them intentionally and avoid running them while another graph still owns related virtual resources.

## MCP observation flow

Recommended automation sequence:

1. `serial_graph_provider_catalog`.
2. `serial_graph_validate`.
3. `serial_graph_start`.
4. `serial_graph_status`.
5. `serial_graph_send` if needed.
6. `serial_graph_query_node_buffer` and/or `serial_graph_query_node_frames`.
7. `serial_graph_stop`.

See [MCP API and Recipes](mcp-api.md) for exact tool lists and examples.
