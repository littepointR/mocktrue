# Architecture

PortWeave is an open-source serial topology, protocol simulation, and MCP automation workbench. The architecture keeps the desktop shell thin, puts serial/runtime behavior in Go services, and keeps graph authoring plus inspectors in Vue/TypeScript.

## Product architecture summary

Current source anchors:

- `go.mod` uses Go `1.26.0`, Wails v3 alpha, the Go MCP SDK, Goja, and `go.bug.st/serial`.
- `main.go` embeds `frontend/dist`, registers the serial module and MCP server module, adds runtime/workspace services, and creates a Wails webview window.
- `internal/core/app/app.go` builds the Wails app, wraps Go services, bridges events to the frontend, and starts modules on `ApplicationStarted`.
- `internal/core/module/registry.go` provides topological module lifecycle ordering.
- `frontend/package.json` uses Vue 3, TypeScript, Vite, Pinia, Naive UI, Monaco, Vitest, and Playwright.
- `frontend/src/serial/graph/serialGraph.ts` owns frontend graph schema, provider metadata, validation helpers, and default config.
- `internal/modules/serial/graph_runtime.go` owns backend runtime validation, resource startup, byte routing, node buffers, frames, counters, and cleanup.
- `internal/modules/mcpserver/server.go` exposes the automation surface through MCP tools.

## Runtime layering

```text
PortWeave desktop process
  -> Wails v3 application shell
  -> Go module registry
  -> serial module + MCP server module + workspace/runtime services
  -> Wails service bridge and application events
  -> Vue/TypeScript stores and views
  -> serial graph editor, node inspectors, buffers, frames, and operation logs
```

This layering is intentionally service-oriented rather than dashboard-first. The graph runtime is the product core: nodes own resources, route bytes, record evidence, and expose state to UI and MCP automation.

## Backend services

The backend is organized around Go modules and services:

- Core app builder: Wails application setup and lifecycle events.
- Module registry: dependency ordering, service aggregation, `Init`/`Start`/`Stop`/`Dispose` lifecycle.
- Serial module: serial ports, virtual ports, bridges, monitors, Modbus, FECbus, graph runtime, script execution, buffers, and frames.
- MCP server module: local HTTP MCP endpoint, tool registration, origin guard, and graph automation templates.
- Workspace/runtime services: app metrics and workspace import/export support.

Modules should keep exported Wails service/model boundaries deliberate because generated bindings are affected by exported Go signatures.

## Wails application boundary

Wails provides the native desktop shell and service bridge. PortWeave currently uses Wails v3 alpha, so the boundary should stay narrow and explicit:

- Keep product logic in Go services and TypeScript stores rather than Wails glue.
- Regenerate/check bindings only when exported Go service/model signatures change.
- Review generated binding diffs carefully.
- Run frontend typecheck/build and backend tests around binding-sensitive changes.
- Add release artifact smoke checks before expanding package matrices.

## Frontend architecture

The frontend uses Vue/TypeScript stores and views:

- Graph provider metadata and validation: `frontend/src/serial/graph/serialGraph.ts`.
- Graph runtime store and Wails bindings: `frontend/src/serial/stores/graphStore.ts`.
- Demo workspaces: `frontend/src/workspace/demoWorkspaces.ts`.
- Workspace/graph snapshot formats: `frontend/src/workspace/workspaceSnapshot.ts`.
- Settings/demo loader surface: `frontend/src/settings/views/global/GlobalSettingsPanel.vue`.

The UI should invest in topology-native debugging surfaces: node inspectors, edge traffic views, protocol frame tables, scenario timelines, operation logs, lightweight metrics, and validation reports.

## Serial graph layer

A serial graph has two related forms:

- `SerialGraphDocument`: authoring state saved in workspace/graph snapshots. It includes graph ID/name, nodes, edges, selected IDs, node tabs, and active node tab.
- Runtime graph: backend resources and goroutines created by `StartSerialGraph`. It validates nodes/edges/resources, opens resource-owning nodes, routes bytes along edges, records buffers/frames, and exposes status/counters.

Current node provider types are documented in [Serial Graph Node Catalog](serial-graph-node-catalog.md). Runtime behavior is documented in [Serial Graph Runtime Flow](serial-graph-runtime-flow.md).

## Automation layer

PortWeave is MCP-first today:

1. Inspect `serial_graph_provider_catalog` and `serial_graph_demo_catalog`.
2. Generate a safe template through `serial_graph_demo_template`.
3. Validate nodes and edges before starting resources.
4. Start graph runtime.
5. Query status, buffers, and frames for evidence.
6. Stop graph and clean up resources.

The current default MCP server is local-only by configuration (`127.0.0.1:39391/mcp`) and should not be exposed to untrusted networks. See [MCP API and Recipes](mcp-api.md) and [AI Automation Guide](ai-automation.md).

## Architecture direction

PortWeave should keep Go + Wails + Vue/TypeScript. It should not be rewritten in Qt/C++ without an explicit new architecture decision and product-direction change. It should also defer gRPC until MCP/HTTP shows concrete evidence of insufficient streaming or typing behavior.

More formal decisions are captured in [Architecture Decisions](development/architecture-decisions.md).

## Extension guide

When adding a graph capability:

1. Add backend runtime behavior in the serial module.
2. Add/update frontend provider metadata and validation in `serialGraph.ts`.
3. Add MCP catalog/tool coverage only if automation needs it.
4. Add demo snapshots only after behavior is stable and no-hardware paths are possible.
5. Update docs and tests together.
6. Run targeted Go tests, targeted Vitest tests, typecheck/build, and binding checks when exported service/model signatures change.

Avoid introducing a second desktop stack, undocumented transports, or UI features that bypass graph validation and runtime evidence.
