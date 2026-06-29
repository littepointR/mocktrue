# Troubleshooting

Use this guide when a PortWeave graph, serial endpoint, protocol node, script node, or MCP automation recipe does not behave as expected.

## Quick diagnosis checklist

1. Is the graph valid before start?
2. Are node IDs, edge IDs, source handles, and target handles correct?
3. Is the relevant resource already owned by another node/session?
4. Did graph runtime actually start, or are you only looking at saved workspace state?
5. Are bytes visible in the upstream node buffer?
6. Are monitor/protocol/script frames visible downstream?
7. Is the node status `running`, `reconnecting`, or `error`?
8. Did you stop the graph and release resources after the previous run?
9. If using MCP, did you validate before start and query evidence after action?

## Port not visible

**Symptoms**

A physical port is not listed or cannot be selected.

**Likely causes**

- Device is unplugged or driver is missing.
- OS permissions block serial access.
- Wrong platform-specific port name.

**Checks**

- Use the UI port list or `serial_enumerate_ports`.
- Confirm the OS can see the port outside PortWeave.
- Check cable/adapter/driver state.

**Fixes**

- Reconnect the device.
- Fix driver or permission setup.
- Use a no-hardware demo first to verify PortWeave itself starts.

## Port open fails

**Symptoms**

Graph start or serial open reports a resource/open error.

**Likely causes**

- Port is already open by another app or node.
- Config line parameters are wrong for the device.
- The same `portName` is reused by two resource owners in one graph.

**Checks**

- Look for duplicate `serial.physical` or `serial.virtual` nodes with the same `portName`.
- Stop prior graph runtimes.
- Query status/list tools for open resources.

**Fixes**

- Close the other app/session.
- Stop the graph before restarting.
- Give each graph-owned virtual port a unique `portName`.

## Virtual port creation fails

**Symptoms**

A `serial.virtual` node fails during graph start.

**Likely causes**

- Duplicate `portName`.
- Platform virtual serial backend is unavailable.
- Leftover OS resources from a previous interrupted run.

**Checks**

- Check graph validation for duplicate resource errors.
- List virtual resources.
- On POSIX, verify required virtual serial tooling if using integration paths.

**Fixes**

- Use generated unique `portweave-demo-*` names for demos.
- Stop graph runtimes cleanly.
- Use cleanup only when sure resources are stale and not owned by another active graph.

## Graph validation fails

**Symptoms**

The UI or MCP reports validation errors and refuses to start.

**Likely causes**

- Unknown node type.
- Missing source/target node.
- Wrong handle ID.
- Incompatible port kinds.
- Input already connected.
- Directed cycle.
- Duplicate resource key.
- Invalid remote config.

**Checks**

- Compare node types with [Serial Graph Node Catalog](serial-graph-node-catalog.md).
- Compare handle IDs with provider inputs/outputs.
- Check remote config against [Remote Serial Graph Node](remote-serial-node.md).

**Fixes**

- Fix IDs and handles.
- Use output fan-out instead of multi-source input.
- Use the protocol response pattern for Modbus/FECbus master responses.
- Remove duplicate resource owners.

## Duplicate resource owner / duplicate endpoint

**Symptoms**

Validation says resource port or remote endpoint is duplicated.

**Likely causes**

- Two physical/virtual nodes use the same `portName`.
- Two remote nodes use the same endpoint and role.

**Checks**

- Inspect every resource-owning node.
- For remote nodes, normalize to host + port + role.

**Fixes**

- Rename one virtual port.
- Use a different remote port or role.
- Remove accidental duplicate nodes.

## Graph start fails

**Symptoms**

Validation passes, but runtime start fails.

**Likely causes**

- Real resource unavailable at runtime.
- Remote endpoint unavailable and `allowStartDisconnected = false`.
- Virtual serial backend error.
- Script/protocol config error encountered at startup.

**Checks**

- Read the graph start error.
- Check resource owner nodes first.
- Try a no-hardware demo to isolate environment problems.

**Fixes**

- Restore the missing resource.
- For remote client nodes expected to appear later, set `allowStartDisconnected = true` and keep `reconnect = true`.
- Reduce graph to a minimal generator -> virtual -> monitor path and expand from there.

## Node buffer has no data

**Symptoms**

Graph is running, but a node buffer is empty.

**Likely causes**

- Upstream generator/endpoint is not producing bytes.
- Edge source/target handles route bytes somewhere else.
- Endpoint is disconnected.
- Data is being dropped by a filter or script.

**Checks**

- Inspect upstream node counters.
- Add or inspect a monitor branch before filters/protocol parsers.
- Query `serial_graph_status` and `serial_graph_query_node_buffer`.

**Fixes**

- Start/enable generator or auto-send config.
- Fix edge handles.
- Query upstream before downstream.

## Monitor frames are empty

**Symptoms**

A monitor node has no frames.

**Likely causes**

- Monitor input is not connected.
- Upstream node has no bytes.
- Filter/script upstream drops all bytes.

**Checks**

- Confirm edge target is `monitor.in`.
- Inspect upstream buffer.
- Temporarily connect monitor directly to endpoint `rx`.

**Fixes**

- Correct the input edge.
- Add direct monitor branch before filters.

## Filter drops unexpected data

**Symptoms**

A filter branch produces no downstream monitor frames.

**Likely causes**

- Wrong `mode`.
- Regex expression mismatch.
- Case sensitivity or whole-word behavior.
- Expression does not match fields such as length/text/hex.

**Checks**

- Inspect raw upstream bytes and text/hex rendering.
- Compare with the `serial-observability-demo` examples.

**Fixes**

- Start with `mode = plain` and a known substring.
- Disable case sensitivity until the expression is proven.
- Add a parallel monitor branch before the filter.

## Script node produces no output

**Symptoms**

Script transform/generator does not produce expected bytes.

**Likely causes**

- Empty or invalid script.
- Script timed out.
- Output exceeded `maxOutputBytes`.
- Transform has no input bytes.
- Analyzer is being used where a transform/generator is needed.

**Checks**

- Check node error/status.
- Confirm script node type and available ports.
- Inspect upstream buffer for transform/analyzer inputs.

**Fixes**

- Use a minimal script such as `output.bytes(input.bytes())` for transform or `output.text("tick", "utf-8")` for generator.
- Increase limits only after confirming the script is bounded.
- Use `serial.script.transform` when downstream bytes are required.

## Script analyzer fields are missing

**Symptoms**

Analyzer receives bytes but no fields appear.

**Likely causes**

- Script does not call `field()`.
- Parser condition skips field creation.
- Input data shape differs from expected.

**Checks**

- Confirm analyzer input edge.
- Inspect raw monitor branch from the same upstream output.
- Start with `field("length", input.bytes().length)`.

**Fixes**

- Add simple field extraction first.
- Add error fields for parse failures.

## Modbus master receives no response

**Symptoms**

Master sends but response frames are empty.

**Likely causes**

- Request path does not reach slave `rx`.
- Slave `tx` is not connected to master `rx`.
- Unit ID/mode mismatch.
- Invalid function/address/quantity config.

**Checks**

- Compare with `modbus-demo` topology.
- Inspect monitor branch on slave `tx`.
- Query master/slave frames.

**Fixes**

- Use: master `tx` -> virtual `tx`, virtual `rx` -> slave `rx`, slave `tx` -> master `rx`.
- Align `mode` and `unitIds`.

## FECbus slave does not answer

**Symptoms**

Master request appears but no slave response is visible.

**Likely causes**

- Request does not reach slave `rx`.
- Address mismatch.
- Invalid data hex/function config.
- No monitor on slave `tx`.

**Checks**

- Compare with `fecbus-demo` topology.
- Inspect master request frame and slave status.

**Fixes**

- Align master `targetAddress` with slave `address`.
- Use default demo config first.
- Add monitor branch to slave `tx`.

## Remote raw TCP connection refused or reconnecting

**Symptoms**

`serial.remote` start fails or status stays `reconnecting`.

**Likely causes**

- No TCP server at host/port.
- Firewall/VPN/tunnel issue.
- Host includes a URL scheme.
- Server allows only one connection.
- Raw TCP endpoint is exposed outside trusted network boundaries.

**Checks**

- Use `remote-serial-demo` to prove in-graph loopback.
- Check `host`, `port`, `protocol`, and `role`.
- Confirm endpoint availability outside PortWeave.

**Fixes**

- Remove schemes such as `tcp://` from `host`.
- Set `allowStartDisconnected = true` only when reconnect is expected.
- Use SSH/VPN/TLS outside PortWeave for untrusted networks.

## MCP tool call fails

**Symptoms**

MCP client cannot connect or a tool returns an error.

**Likely causes**

- MCP server disabled or not running.
- Wrong local endpoint/path.
- Origin guard blocks request.
- Tool arguments do not match current schema.
- Graph was not validated before start.

**Checks**

- Default local config is `127.0.0.1:39391/mcp`.
- Inspect `serial_graph_provider_catalog` before constructing graph args.
- Use `serial_graph_demo_template` for known-good no-hardware examples.

**Fixes**

- Keep MCP local unless intentionally configured otherwise.
- Use exact current tool names from [MCP API and Recipes](mcp-api.md).
- Validate before start and stop runtime after collecting evidence.

## Platform notes

### Windows

Default checks do not require real COM ports. Optional com0com testing requires `PORTWEAVE_TEST_COM_A` and `PORTWEAVE_TEST_COM_B`. See [Windows Serial Testing](development/windows-serial-testing.md).

### macOS/Linux

POSIX virtual serial integration tests use `socat` and are opt-in with integration build tags. Development builds may require Wails/WebKit platform dependencies depending on OS.

### Remote TCP

Raw TCP carries bytes only. It does not configure baud rate, parity, stop bits, flow control, authentication, or encryption. Configure serial line settings on the remote adapter/server and secure the network path outside PortWeave.
