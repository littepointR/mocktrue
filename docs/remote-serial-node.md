# Remote Serial Graph Node

PortWeave serial graphs support a `serial.remote` node for connecting a topology to a remote serial server over raw TCP. It is intended for transparent byte streams exposed by tools such as ser2net, TCP-to-serial adapters, VPN-connected gateways, or an SSH tunnel that forwards a remote serial endpoint to a trusted host and port.

## Scope

The current graph runtime implements raw TCP **client** and **server** roles:

- Client nodes connect to `host:port` with `protocol: raw-tcp` and `role: client`.
- Server nodes listen on `host:port` with `protocol: raw-tcp` and `role: server`.
- Exposes a byte input port `tx`; bytes received on this graph input are written to the TCP connection.
- Exposes a byte output port `rx`; bytes read from TCP are emitted downstream and stored in the node buffer.
- Reports runtime status, resource ID, error text, RX/TX counters, and the inbound buffer through the same graph runtime APIs as physical/virtual serial nodes.
- Supports reconnect loops when `reconnect` is enabled, and can optionally start in `reconnecting` state when `allowStartDisconnected` is true.

## Security boundary

`serial.remote` raw TCP does **not** provide authentication, authorization, encryption, or serial parameter negotiation. Treat it like a direct serial cable stretched across the network:

- Use it only on trusted LANs, VPNs, SSH tunnels, lab networks, or other access-controlled networks.
- Do not expose raw TCP serial ports directly to the public Internet.
- Keep passwords, tokens, and private tunnel credentials out of graph configs and documentation.
- Configure baud rate, parity, stop bits, flow control, and line discipline on the remote serial server or adapter. Phase 1 stores those fields as PortWeave-side intent/notes only.

## ser2net raw TCP example

A minimal ser2net v4 YAML endpoint for `/dev/ttyUSB0` on TCP port `3001`:

```yaml
connection: &portweave-usb0
  accepter: tcp,3001
  connector: serialdev,/dev/ttyUSB0,115200N81
```

Then configure PortWeave to connect to the host running ser2net. If the server is not on a trusted network, prefer an SSH tunnel instead of exposing the TCP listener:

```bash
ssh -N -L 3001:127.0.0.1:3001 user@serial-gateway.local
```

With the tunnel above, set `host = 127.0.0.1` and `port = 3001` in PortWeave.

## Built-in examples

PortWeave includes a frontend demo workspace named `remote-serial-demo` / `远端串口演示`. It builds a safe example topology without pre-opening runtime resources:

```text
serial.script.generator -> serial.remote(client)
serial.remote(server).rx -> serial.monitor
serial.remote(client).rx -> serial.monitor
```

The demo starts a graph-owned raw TCP server and client on `host = 127.0.0.1`, `port = 3001`, `protocol = raw-tcp`. It does not require an external ser2net endpoint or a process outside PortWeave; the client generator sends startup traffic into the client endpoint, and both remote endpoint buffers plus monitor branches can be inspected.

MCP automation can generate the same shape through the read-only template catalog:

```json
{
  "tool": "serial_graph_demo_template",
  "arguments": {
    "id": "serial-remote-raw-tcp",
    "graph_id": "remote-demo",
    "remote_host": "127.0.0.1",
    "remote_port": 3001,
    "allow_start_disconnected": true
  }
}
```

Use the returned `nodes` and `edges` directly with `serial_graph_validate`, then `serial_graph_start`. The template response also includes `serial_graph_status`, `serial_graph_send`, and `serial_graph_query_node_buffer` usage examples. MCP templates do not persist or synthesize frontend UI operation-log state; they cover graph config plus backend runtime buffers/status.

## PortWeave node configuration

Recommended Phase 1 configuration:

```text
protocol = raw-tcp
role = client
host = serial-gateway.local
port = 3001
connectTimeoutMs = 3000
writeTimeoutMs = 3000
reconnect = true
reconnectIntervalMs = 1000
allowStartDisconnected = false
readBufKB = 32

# Raw TCP does not negotiate these values; keep them aligned with ser2net/device config.
baudRate = 115200
dataBits = 8
stopBits = 1
parity = none
flowMode = none
```

`serial.remote` intentionally does **not** use a `mode` config key. Other graph node types already use `mode` for payload encoding, filter mode, and protocol framing; remote serial uses `protocol` and `role` to avoid that collision.

## Configuration reference

| Key | Default | Notes |
| --- | --- | --- |
| `protocol` | `raw-tcp` | Only `raw-tcp` is supported in Phase 1. |
| `role` | `client` | `client` connects to an existing endpoint; `server` listens in the graph runtime. |
| `host` | empty | Required before validation/start succeeds; use a hostname or IP without URL scheme. |
| `port` | `3001` | TCP port, `1..65535`. |
| `connectTimeoutMs` | `3000` | TCP dial timeout, `100..60000` ms. |
| `writeTimeoutMs` | `3000` | Per-write deadline, `100..60000` ms. |
| `reconnect` | `true` | Reconnect after a connected session drops. |
| `reconnectIntervalMs` | `1000` | Delay between reconnect attempts, `100..60000` ms. |
| `allowStartDisconnected` | `false` | If false, graph start fails when the TCP endpoint is unavailable; if true, the node starts as `reconnecting`. |
| `readBufKB` | `32` | TCP read buffer size, `1..1024` KiB; runtime still clamps defensively. |
| `baudRate`, `dataBits`, `stopBits`, `parity`, `flowMode` | serial metadata defaults | Kept for future RFC2217/serial metadata compatibility; Phase 1 raw TCP does not negotiate them. |
| `viewMode`, `autoScroll`, `showTimestamp` | endpoint buffer display defaults | Controls node buffer rendering in the frontend. |

## Suggested topologies

Use `serial.remote` as the graph-owned resource node that bridges a remote TCP byte stream with ordinary graph nodes:

```text
serial.script.generator -> serial.remote.tx
serial.remote.rx -> serial.monitor
```

For inspection and fan-out, connect the `rx` output directly to multiple downstream inputs; endpoint buffers are also queryable directly:

```text
serial.remote.rx -> serial.monitor
                 \-> serial.monitor
```

For protocol flows, connect protocol-node transmit output into `serial.remote.tx`, then route `serial.remote.rx` directly to the protocol receive input and any monitor nodes that need the same bytes. Keep the graph acyclic by using the existing protocol response-cycle pattern rather than drawing a direct cycle.

## Validation and resource identity

The frontend graph model, backend runtime, and MCP tools all validate the same constraints:

- `host` is required.
- `host` must not include a URL scheme such as `tcp://`.
- `port` must be an integer in `1..65535`.
- `protocol` must be `raw-tcp`.
- `role` must be `client`.
- `connectTimeoutMs`, `writeTimeoutMs`, and `reconnectIntervalMs` must be integers in `100..60000`.
- `readBufKB` must be an integer in `1..1024`.
- Two remote nodes cannot use the same computed endpoint identity in one graph.

Remote resource identity is computed as the normalized raw TCP endpoint, for example:

```text
raw-tcp://127.0.0.1:3001
```

Normalization is intentionally conservative: trim whitespace, lowercase host names, format with host/port joining, and do not resolve DNS. `localhost` and `127.0.0.1` remain distinct identities. Hosts must be entered without URL schemes.

This is deliberately different from physical/virtual serial nodes, which use `portName` as their resource key.

## Runtime behavior

- Initial dial succeeds: node status becomes `running`, `ResourceID` is `raw-tcp://host:port`, and the read loop emits data from `rx`.
- Initial dial fails with `allowStartDisconnected=false`: graph start fails and no runtime is left behind.
- Initial dial fails with `allowStartDisconnected=true`: graph starts, node status becomes `reconnecting`, and reconnect attempts continue when `reconnect=true`.
- A connected session drops with `reconnect=true`: node status becomes `reconnecting`, the graph runtime remains `running`, and a later successful dial restores `running`.
- A connected session drops with `reconnect=false`: node status becomes `error`; data is not queued indefinitely.
- Writes while disconnected return a clear error instead of buffering unbounded data.
- `StopSerialGraph` cancels reconnect/read loops and closes the current TCP connection.

## Troubleshooting

### Graph start fails with connection refused

- Confirm the remote serial server is listening on the configured host and port.
- Check firewalls, VPN routes, and SSH tunnel state.
- If the endpoint is expected to appear later, set `allowStartDisconnected = true` and keep `reconnect = true`.

### Node stays in `reconnecting`

- Verify the endpoint can accept a new TCP connection after the first disconnect.
- Check `reconnectIntervalMs`; very short intervals are rejected, and very long intervals delay recovery.
- Confirm the remote server does not allow only one connection that remains occupied by another client.

### Writes fail while disconnected

- This is expected. Phase 1 does not maintain an unlimited offline TX queue.
- Wait for status `running`, then send again.
- If the node is `error` with `reconnect=false`, stop/start the graph after restoring the endpoint or enable reconnect.

### No data appears in the endpoint buffer

- Confirm the graph connects `serial.remote.rx` to a monitor or protocol receive input.
- Confirm the remote device is actually transmitting bytes after TCP connection.
- Use a monitor branch to inspect raw bytes before protocol parsers.

### Baud rate or parity changes do not affect the device

- Raw TCP carries bytes only. It does not negotiate serial line parameters.
- Update the ser2net/adapter configuration on the remote host and restart that service if needed.

## Non-goals for Phase 1

- RFC2217 negotiation.
- TCP server/listener mode.
- TLS, authentication, SSH tunneling, or proxy management inside PortWeave.
- Serial line control negotiation over raw TCP.
- Cross-graph global endpoint locking beyond the active graph runtime lifecycle.
