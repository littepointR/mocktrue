# Migration Contracts

This document defines the first wave of contracts that must be stable before
Qt/QML + Rust implementation begins. It complements
`migration-feature-matrix.csv`; the matrix tracks every feature row, while this
file defines the first rows that may enter implementation.

## First-wave scope

The first wave is deliberately small. It proves the new stack can replace the
problematic WebView shell and run one no-hardware product path before broader
protocol work begins.

Included feature rows:

- `shell.window_lifecycle`
- `shell.module_navigation`
- `workspace.new`
- `workspace.open`
- `workspace.save`
- `workspace.save_as`
- `workspace.restore_last`
- `workspace.dirty_state`
- `workspace.demo_catalog`
- `settings.theme`
- `serial.enumerate`
- `serial.open_close`
- `serial.virtual_backend_status`
- `serial.virtual_pair`
- `graph.document`
- `graph.demo_catalog`
- `graph.validate`
- `graph.runtime_start_stop`
- `graph.runtime_status`
- `graph.runtime_query_buffers`

Everything else stays in `inventory` until this wave launches, verifies, and
produces a working development skeleton.

## Architecture contract

Target layering:

```text
QML UI -> Qt/C++ ViewModels and native shell -> Rust core
```

Hard boundaries:

- QML must not call Rust directly.
- Rust must not depend on Qt, QML, widgets, or window types.
- Qt/C++ must own application startup, windows, screens, DPI, native menus,
  dialogs, ViewModels, Qt models, and UI-thread handoff.
- Rust must own workspace parsing/persistence, serial/virtual serial behavior,
  graph validation/runtime behavior, and error classification.
- Bridge code must only convert types, map errors, hand off async calls, and
  forward events.

## Workspace contract

Rust API names are provisional but must preserve these capabilities:

```text
portweave_workspace::new_graph() -> GraphDocument
portweave_workspace::open_workspace(path) -> WorkspaceLoadResult
portweave_workspace::save_workspace(path, WorkspaceDocument) -> SaveResult
portweave_workspace::save_workspace_as(path, WorkspaceDocument) -> SaveResult
portweave_workspace::load_last_workspace() -> Option<WorkspaceLoadResult>
portweave_workspace::remember_last_workspace(path)
portweave_workspace::demo_catalog() -> Vec<DemoWorkspaceInfo>
portweave_workspace::demo_workspace(id) -> WorkspaceDocument
```

Required compatibility:

- Load current `portweave.workspace.v1` and `portweave.graph.v1` snapshots.
- Preserve graph IDs, node IDs, edge IDs, selected IDs, node tabs, active graph,
  serial settings, editor layout, monitor state, Modbus state, FECbus state,
  graph runtime snapshots, virtual resource snapshots, and received bytes where
  old snapshots contain them.
- Save stable JSON so unchanged load-save-load cycles do not produce
  meaningless churn.
- Keep dirty-state semantics outside QML business logic.

Qt surface:

```text
ProjectViewModel
  properties: currentPath, title, dirty, loading, lastError, activeGraphId
  methods: newGraph(), openWorkspace(), saveWorkspace(), saveWorkspaceAs(),
           loadLastWorkspace(), loadDemo(id)
  signals: workspaceLoaded, workspaceSaved, dirtyChanged, errorRaised
```

Acceptance:

- Old demo workspace opens.
- Save and reopen restores graph and serial settings.
- Save clears dirty state.
- Startup loads remembered workspace or creates a default graph.

## Shell and module contract

Qt owns:

- Main window creation.
- Cross-screen movement.
- DPI change handling.
- Minimize/restore.
- Resize.
- Native file dialogs.
- Close confirmation when workspace is dirty.

QML owns:

- Activity bar.
- Sidebar.
- Toolbar actions.
- Editor area.
- Status bar.
- Settings panels.

The `opus-qml-design-lead` owns the first-wave QML design contract before
implementation starts. Required design outputs:

- QML screen inventory for first-wave rows.
- Mapping from each toolbar/sidebar/settings/graph action to a feature ID.
- State model for loading, empty, dirty, saved, running, stopped, error, and
  unsupported states.
- Navigation model for activity bar, sidebar, editor area, and settings.
- Visual parity checklist against the old Vue shell, focused on feature
  discoverability rather than a broad redesign.

This role does not implement the QML pages and does not accept feature rows.
The QML implementation agents must still run the local QML/GUI smoke path and
the verification owner must still record acceptance evidence.

Required window lifecycle smoke:

```text
launch -> move to secondary monitor -> resize -> minimize -> restore
-> move back to primary monitor -> verify QML content is visible and clickable
```

The skeleton is not ready for feature implementation until this smoke passes.

## Serial first-wave contract

Rust API names are provisional but must preserve these capabilities:

```text
portweave_serial::list_ports() -> Vec<PortInfo>
portweave_serial::open_port(OpenPortRequest) -> PortHandleStatus
portweave_serial::close_port(handle_id)
portweave_serial::list_open_ports() -> Vec<PortHandleStatus>
portweave_virtual::backend_status() -> VirtualBackendStatus
portweave_virtual::create_pair(CreatePairRequest) -> VirtualPairInfo
portweave_virtual::delete_pair(id)
portweave_virtual::list_pairs() -> Vec<VirtualPairInfo>
```

Required behavior:

- No-device enumeration returns an empty list without error.
- Missing port, busy port, permission denied, invalid config, and unsupported
  virtual backend errors map to stable classified errors.
- Open handles expose IDs, port config, RX/TX counters, and open/closed state.
- Virtual pair create/delete/list works without physical hardware when the
  platform backend supports it.
- Virtual unsupported state disables create actions but does not break the UI.

Qt surface:

```text
SerialPortListModel
SerialPortViewModel
  properties: selectedPort, openHandles, lastError, busy
  methods: refreshPorts(), openPort(request), closePort(handleId)

VirtualSerialViewModel
  properties: backendStatus, pairs, lastError, busy
  methods: refreshBackendStatus(), createPair(request), deletePair(id),
           refreshPairs()
```

Acceptance:

- Refresh works with zero serial devices.
- Open/close can be tested with a physical port or mocked backend.
- Virtual pair loopback works in no-hardware smoke when backend is available.

## Graph first-wave contract

Rust API names are provisional but must preserve these capabilities:

```text
portweave_graph::load_document(snapshot) -> GraphDocument
portweave_graph::validate(GraphDocument) -> ValidationResult
portweave_graph::start_runtime(StartRuntimeRequest) -> RuntimeInfo
portweave_graph::stop_runtime(graph_id)
portweave_graph::runtime_status(graph_id) -> RuntimeInfo
portweave_graph::query_node_buffer(BufferQuery) -> BufferSnapshot
portweave_graph::clear_node_buffer(graph_id, node_id)
portweave_graph::reset_node_counters(graph_id, node_id)
portweave_graph::demo_catalog() -> Vec<GraphDemoInfo>
portweave_graph::demo_template(id) -> GraphDocument
```

Required behavior:

- Current no-hardware demos validate.
- Start creates runtime resources and status entries.
- Stop releases runtime resources.
- Buffer query and clear semantics match current `graph_runtime_test.go`.
- Validation errors must remain specific enough for UI and MCP recipes.

Qt surface:

```text
GraphDocumentViewModel
GraphRuntimeViewModel
GraphRuntimeStatusModel
GraphNodeBufferModel
```

Acceptance:

- Load one no-hardware demo.
- Validate it.
- Start it.
- Query runtime status.
- Query at least one node buffer.
- Stop it and verify cleanup.

## Error contract

Rust must expose classified errors instead of raw strings:

```text
invalid
not_found
conflict
permission
unsupported
io
timeout
internal
```

Qt maps classified errors to UI state and user-visible messages. QML displays
errors but does not classify or parse them.

## Testing contract

Before first-wave implementation starts:

- Matrix rows listed in this document must stay `contracted`.
- Rust tests may be written before full implementation.
- Qt/QML smoke tests must include the window lifecycle case.
- GUI smoke must verify visible content, not only process survival.
- `opus-qml-design-lead` must review the first-wave QML screen inventory and
  state model before QML implementation begins.
- `opus-qml-design-lead` design approval does not replace Qt/QML smoke,
  screenshot evidence, or final feature acceptance.

First-wave implementation may start only after:

- Rust toolchain is installed.
- Qt 6 CMake build path is confirmed.
- Skeleton build can run a Rust-to-QML event.
- The first unmapped report has no blockers for first-wave rows.
