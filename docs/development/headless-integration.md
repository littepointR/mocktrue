# Headless Integration

This page records the confirmed PortWeave server/headless behavior and the gates required before adding broad integration harnesses.

## Confirmed startup mode

PortWeave can be built with the Wails server build tag:

```bash
go build -tags server -o /tmp/portweave-server-smoke .
```

Local smoke result on macOS during this phase:

- The binary stayed running until sent a shutdown signal.
- Logs reported `Platform Info: mode=server` and `Server mode enabled (built with -tags server)`.
- Logs reported `Server mode starting address=localhost:8080`.
- `GET http://localhost:8080/` returned `200` and served the embedded frontend HTML.
- `GET http://localhost:8080/mcp` returned `404`, so readiness for MCP must not be inferred from a browser-style GET request.

Important limitation: the current `-tags server` binary still links native macOS frameworks through Wails/runtime dependencies. Treat it as Wails server mode, not as a proven minimal daemon with no native dependencies.

## Build and run commands

Direct build:

```bash
go build -tags server -o /tmp/portweave-server-smoke .
/tmp/portweave-server-smoke
```

Wails task path from `build/Taskfile.yml`:

```bash
wails3 task build:server
wails3 task run:server
```

The Wails task builds `bin/portweave-server` with `-tags server` after building the frontend.

## Readiness checks

Use bounded readiness loops. Do not use unbounded sleeps.

Minimum HTTP readiness:

1. Start the server process.
2. Poll `http://localhost:8080/` until it returns `200` or the timeout expires.
3. Capture stderr/stdout logs on failure.
4. Terminate the process and wait for exit in cleanup.

MCP readiness must use the actual MCP HTTP transport rather than `GET /mcp`. The MCP module endpoint is configured separately by `internal/core/config/config.go` and defaults to `127.0.0.1:39391/mcp` for the desktop module path. A future harness should prove whether Wails server mode starts the MCP module and which address/path are reachable before it sends graph calls.

## Teardown rules

Every headless smoke must:

- Start exactly one process owned by the test.
- Use a context/timeout around startup and MCP calls.
- Stop graphs it starts with `serial_graph_stop`.
- Terminate the process on every exit path.
- Fail with captured logs when readiness or teardown fails.
- Avoid physical serial devices unless the test is explicitly opt-in and environment-gated.

## Minimal MCP smoke plan

Only add an integration harness after the MCP endpoint is confirmed reachable in server mode. The first harness should be narrow:

1. Build or locate the server-mode binary.
2. Start it on a non-conflicting localhost port if Wails exposes that configuration, otherwise skip with a clear reason.
3. Connect through a real MCP HTTP client/transport.
4. Call `tools/list` and assert read-only catalog tools exist:
   - `serial_graph_provider_catalog`
   - `serial_graph_demo_catalog`
   - `serial_graph_demo_template`
   - `protocol_template_catalog`
   - `protocol_template_describe`
5. Call `serial_graph_demo_template` for `serial-remote-raw-tcp` using localhost.
6. Call `serial_graph_validate` on the returned nodes/edges.
7. Start the graph only if validation succeeds.
8. Query status/buffer/frame evidence.
9. Stop the graph and terminate the process.

## Deferred decisions

No headless integration harness is added in this phase because the confirmed facts are sufficient for documentation but not yet sufficient for a stable MCP smoke:

- `GET /mcp` is not a valid readiness signal.
- The exact server-mode MCP endpoint and transport behavior still need a real MCP HTTP client check.
- Port configurability for parallel CI execution needs confirmation before adding a non-flaky test.

Do not fake a passing headless MCP smoke by checking only the frontend HTML server.

## Related docs

- [MCP API and Recipes](../mcp-api.md)
- [AI Automation Guide](../ai-automation.md)
- [Testing](testing.md)
- [Release](release.md)
