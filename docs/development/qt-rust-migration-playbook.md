# Qt/QML + Rust Migration Playbook

This playbook describes how to migrate PortWeave from the current Go + Wails +
Vue/TypeScript stack to a Qt/QML desktop shell with a Rust core without losing
features, behavior, or test evidence. It is intentionally operational: a team
should be able to follow the phases, ownership model, gates, and templates here
to drive the migration to completion.

This document does not replace an architecture decision. Before starting a real
migration, add a new accepted ADR that supersedes ADR-0001 in
[Architecture Decisions](architecture-decisions.md).

## Target architecture

The target stack separates desktop UI, Qt integration, and product logic:

```text
QML UI
  -> Qt/C++ shell, ViewModels, native window lifecycle, Qt models
  -> Rust core library or service
  -> platform backends, protocols, workspace, graph runtime, MCP
```

Layer responsibilities:

- QML owns layout, controls, visual state, lightweight interaction, and
  presentation.
- Qt/C++ owns `QApplication`, windows, DPI, screens, native events, menus,
  file dialogs, `QObject`/`QAbstractListModel` ViewModels, and the bridge to
  Rust.
- Rust owns serial resources, virtual serial logic, graph runtime, protocol
  parsers/encoders, workspace/config persistence, automation APIs, state
  machines, task execution, and core tests.
- The bridge owns type conversion, error conversion, async invocation, thread
  handoff, and event delivery only.

Keep data flow one-way:

```text
User action -> QML -> Qt/C++ ViewModel -> Rust core
Rust event/result -> Qt/C++ ViewModel/model -> QML binding
```

Do not let QML call Rust directly. Do not let Rust depend on QML or Qt widgets.
Do not put protocol, serial, workspace, or graph business rules in QML.

## Definition of done

The migration is complete only when every old capability has one of these final
states:

- `migrated`: new Rust/Qt/QML implementation exists and is verified.
- `removed`: intentionally dropped, with reason, owner, and user-facing impact.
- `deferred`: not in this release, with reason, risk, and a follow-up issue.

No capability may remain in `unknown`, `unmapped`, or `assumed done`.

Every migrated feature must close this chain:

```text
old source -> feature ID -> Rust core API -> Qt/C++ ViewModel -> QML entry
-> automated test -> acceptance evidence
```

A feature is not done if any link is missing.

## Start-up checklist

Use this checklist to start the migration. Complete it before assigning feature
implementation work.

1. Create and accept a new ADR that supersedes ADR-0001.
2. Create `docs/development/migration-feature-matrix.csv` with the columns in
   this playbook.
3. Assign the migration lead, inventory owner, verification owner, Rust owner,
   Qt/C++ owner, QML owner, and release owner.
4. Generate the first inventory from Wails bindings, Go services, MCP tools,
   Vue commands/stores, docs, and tests.
5. Review the first inventory and assign stable feature IDs.
6. Pick the first release scope. The recommended first scope is:
   workspace open/save, settings/theme, serial enumerate/open/close, virtual
   pair loopback, one no-hardware graph demo, and Windows window lifecycle.
7. Move only those first-scope rows from `inventory` to `contracted`.
8. Add CI checks that prevent accepted rows without tests or acceptance
   evidence.
9. Create the Qt/QML + Rust skeleton PR.
10. Start implementation only after the skeleton can launch, emit a Rust event
    into QML, and survive cross-screen/minimize/restore smoke.

The first week should produce an accepted ADR, an initial matrix, a skeleton
build, and a first unmapped-feature report. It should not try to migrate every
domain at once.

## Source-of-truth inventory

Build the migration inventory from source, not memory. The first migration PR
should add a machine-readable feature matrix and keep it updated for every
later PR.

Required inventory sources:

- Current shell and commands: `frontend/src/App.vue`.
- Frontend modules: `frontend/src/serial`, `frontend/src/settings`,
  `frontend/src/workspace`, `frontend/src/core`.
- Generated Wails bindings: `frontend/src/bindings` or `bindings` depending on
  the current generated output path.
- Backend services: `internal/core/workspace`, `internal/core/runtime`,
  `internal/modules/serial`.
- Module lifecycle and app wiring: `internal/core/app`,
  `internal/core/module`, `main.go`.
- MCP tools and automation surface: `internal/modules/mcpserver/server.go`.
- Product docs: `docs/use-cases.md`, `docs/getting-started.md`,
  `docs/mcp-api.md`, `docs/workspace-graph-schema.md`,
  `docs/serial-graph-node-catalog.md`, `docs/serial-graph-runtime-flow.md`.
- Existing tests: `internal/**/*_test.go`, `tests/**/*.go`,
  `frontend/src/**/*.test.ts`.

The inventory owner must classify each discovered item into a stable feature
ID. Prefer product-level IDs over implementation names.

Examples:

```text
workspace.new
workspace.open
workspace.save
workspace.save_as
workspace.restore_last
settings.theme
serial.enumerate
serial.open_close
serial.send_bytes
serial.monitor_frames
serial.virtual_pair
graph.node_catalog
graph.edit_nodes_edges
graph.runtime_start_stop
graph.runtime_query_buffers
graph.script_node
modbus.read_registers
modbus.scan_registers
fecbus.parse_frame
mcp.serial_tools
mcp.graph_tools
runtime.metrics
```

## Feature matrix

Store the matrix in the repo, for example:

```text
docs/development/migration-feature-matrix.csv
```

CSV is enough for the first version because it is easy to diff and easy to
check in CI. Use these columns:

```csv
feature_id,domain,old_ui,old_frontend_state,old_binding,old_backend,old_mcp_tool,old_docs,old_tests,new_rust_api,new_qt_viewmodel,new_qml_entry,new_tests,status,owner,acceptance,notes
```

Column rules:

- `feature_id`: stable ID, lower snake/dot case, never reused for a different
  behavior.
- `domain`: one of `shell`, `workspace`, `settings`, `serial`, `virtual`,
  `graph`, `protocol`, `mcp`, `runtime`, `release`.
- `old_*`: exact source path, symbol, test, tool name, or doc section.
- `new_rust_api`: Rust module/function/type that owns the behavior.
- `new_qt_viewmodel`: C++ ViewModel/model/property/method exposed to QML.
- `new_qml_entry`: QML file/control/action where the user reaches the feature.
- `new_tests`: exact automated test names or smoke script names.
- `status`: one of `inventory`, `contracted`, `implemented`, `tested`,
  `accepted`, `removed`, `deferred`.
- `owner`: person responsible for end-to-end closure of this row.
- `acceptance`: exact observable proof required before `accepted`.
- `notes`: reason for differences, removal, deferral, or known risk.

Status transitions:

```text
inventory -> contracted -> implemented -> tested -> accepted
inventory -> removed
inventory -> deferred
```

Do not skip `contracted`. It means the Rust API, Qt ViewModel surface, QML
entry, error behavior, events, and tests have been reviewed before implementation.

Minimal example rows:

```csv
feature_id,domain,old_ui,old_frontend_state,old_binding,old_backend,old_mcp_tool,old_docs,old_tests,new_rust_api,new_qt_viewmodel,new_qml_entry,new_tests,status,owner,acceptance,notes
workspace.open,workspace,frontend/src/App.vue openGraph,frontend/src/workspace/stores/workspaceFileStore.ts selectOpenPath,bindings/.../workspace/service.ts SelectWorkspaceOpenPath,internal/core/workspace/service.go,,docs/getting-started.md,frontend/src/workspace/stores/workspaceFileStore.test.ts,portweave_workspace::open_workspace,ProjectViewModel::openWorkspace,qml/shell/MainToolbar.qml actionOpen,workspace_open_roundtrip_test; qml_smoke_workspace_open,inventory,workspace-owner,Old workspace JSON opens and active graph/settings restore,
serial.enumerate,serial,,frontend/src/serial/stores/serialStore.ts refreshPorts,bindings/.../serial/service.ts ListPorts,internal/modules/serial/service.go,serial_enumerate_ports,docs/mcp-api.md,internal/modules/serial/*_test.go,portweave_serial::list_ports,SerialPortViewModel::refresh, qml/serial/SerialPortPanel.qml refresh button,serial_list_ports_test; qml_smoke_serial_refresh,inventory,serial-owner,Physical and no-device cases match old behavior,
```

## Team and agent model

Use one human owner per responsibility area and one matching agent profile. The
human owns decisions and final review; the agent performs bounded analysis,
implementation, or verification inside assigned files and feature IDs.

Recommended full team:

| Human role | Agent profile | Owns | Must not own |
| --- | --- | --- | --- |
| Migration lead | `architect-agent` | Architecture boundary, matrix policy, interface reviews, final PR gate | Large unreviewed implementation |
| Inventory owner | `inventory-agent` | Old feature discovery, matrix completeness, unmapped reports | New architecture decisions |
| Rust serial owner | `rust-serial-agent` | Serial ports, handles, read/write, status events, virtual serial backend abstraction | QML layout |
| Rust protocol owner | `rust-protocol-agent` | Modbus, FECbus, protocol templates, parsers, encoders, graph protocol nodes | Window lifecycle |
| Rust workspace owner | `rust-workspace-agent` | Workspace JSON, recent file, config, settings, compatibility | Visual design |
| Qt bridge owner | `qt-bridge-agent` | App shell, windows, DPI/screens, ViewModels, Qt models, Rust bridge | Product protocol rules |
| QML design lead | `opus-qml-design-lead` | Information architecture, QML screen inventory, interaction model, state model, visual parity review | Direct implementation sign-off or bypassing smoke evidence |
| QML shell owner | `qml-shell-agent` | Activity bar, sidebar, toolbar, editor groups, status bar, app styling | Core business state |
| QML feature owner | `qml-feature-agent` | Serial, graph, Modbus/FECbus, virtual serial, settings panels | Direct Rust calls |
| Verification owner | `verification-agent` | Matrix checks, Rust tests, Qt tests, GUI smoke, hardware evidence | Product implementation shortcuts |
| Release owner | `release-agent` | CMake/Cargo/Qt deploy, installer, signing, artifact smoke, checksums | Feature semantics |

For a smaller team, merge roles but keep gates separate:

```text
1. Migration lead + inventory
2. Rust core
3. Qt/C++ bridge
4. QML UI
5. Verification + release
```

Do not merge implementation review and final acceptance into the same person
when the feature touches real serial devices, workspace compatibility, or MCP.

The `opus-qml-design-lead` role is intentionally a design and review role, not
a replacement for QML implementation and local rendering validation. It should
be used where high-level judgment matters:

- Extract the old Vue information architecture into QML screen groups.
- Define toolbar, activity bar, sidebar, editor area, settings, and status bar
  layout rules.
- Map every visible QML command to a feature ID.
- Define loading, empty, dirty, running, stopped, error, unsupported, and
  permission-denied states.
- Review whether QML PRs preserve feature parity and tool-oriented usability.
- Challenge UI redesigns that make feature parity harder to verify.

It must not mark a feature accepted, write broad QML implementation without a
feature row, or override `verification-agent` smoke evidence. QML is complete
only when design review, implementation, feature ID mapping, and GUI smoke all
pass.

## Agent task contract

Every agent task must include:

```text
Agent:
Feature IDs:
Allowed files/modules:
Read first:
Required output:
Tests to run:
Forbidden changes:
Handoff notes:
```

Example:

```text
Agent: rust-serial-agent
Feature IDs:
- serial.enumerate
- serial.open_close

Allowed files/modules:
- rust/core/src/serial/**
- rust/core/tests/serial_*.rs

Read first:
- migration-feature-matrix.csv rows for serial.enumerate and serial.open_close
- internal/modules/serial/service.go
- frontend/src/serial/stores/serialStore.ts
- docs/mcp-api.md serial resource tools

Required output:
- Rust API and tests for listing ports and opening/closing handles
- Error mapping documented in the matrix notes
- Handoff listing public Rust types and event names

Tests to run:
- cargo test -p portweave-core serial

Forbidden changes:
- Do not edit QML.
- Do not edit Qt ViewModels.
- Do not change Modbus, FECbus, or graph runtime behavior.
```

Agents may report gaps, but must not silently expand scope. New gaps become new
matrix rows or explicit notes on existing rows.

## Migration phases

### Phase 0: Decision and baseline

Goal: make the migration intentional and measurable.

Required work:

- Add a new ADR that changes the accepted architecture direction.
- Capture baseline build/test status for the current app.
- Capture baseline no-hardware workflows: open app, load demo workspace, start
  graph, query status, stop graph.
- Capture baseline hardware workflows if devices are available: serial
  enumerate/open/send/monitor, virtual pair, Modbus, FECbus.
- Freeze the initial migration vocabulary: layer names, feature ID format,
  matrix status values, and PR gate rules.

Exit gate:

- ADR accepted.
- Baseline command log recorded in the migration issue or docs.
- Initial feature matrix file exists with at least top-level domains.

### Phase 1: Inventory and contracts

Goal: discover every old feature and define the new interface before building
the new app.

Required work:

- Inventory Wails bindings, Go services, MCP tools, Vue store actions, visible
  commands, docs flows, and tests.
- Assign every discovered item to a feature ID.
- Mark duplicates as aliases, not separate features.
- For each high-risk feature, write the intended Rust API, Qt ViewModel method
  or property, QML entry point, event names, and error behavior.
- Decide removed/deferred items explicitly.

Exit gate:

- No generated binding, MCP tool, visible command, or documented workflow is
  unmapped.
- Each first-wave feature is in `contracted` status.
- Verification owner can produce an `unmapped` report with zero blockers for
  first-wave implementation.

### Phase 2: New skeleton

Goal: create the new application shell without product behavior drift.

Required work:

- Create the Qt/C++ app entry, main window, QML engine, resource loading, and
  logging startup.
- Create the Rust core crate or workspace with initial error and event types.
- Create the bridge shape chosen by the ADR: C ABI, `cxx-qt`, or IPC.
- Create empty ViewModels for workspace, serial, graph, protocol, settings,
  runtime status, and MCP status.
- Create QML shell: toolbar, activity bar, sidebar, editor area, panel/status
  bar, settings shell.
- Add a diagnostics page or log surface for early bridge and runtime errors.

Exit gate:

- New app launches on the target developer OS.
- Cross-screen drag, minimize/restore, resize, and DPI changes keep QML content
  visible.
- Empty ViewModels can emit a test event from Rust to QML.
- CI can build the skeleton.

### Phase 3: Core migration by domain

Goal: migrate core behavior behind stable Rust APIs before relying on UI.

Suggested order:

1. Workspace/config/settings because all later UI state depends on persistence.
2. Serial enumerate/open/close/send/status because it is the hardware boundary.
3. Virtual serial and bridges because they enable no-hardware tests.
4. Monitor/buffer/frame storage because they provide runtime evidence.
5. Graph document, graph validation, and runtime start/stop/query.
6. Protocol domains: Modbus, FECbus, protocol templates.
7. MCP tools and automation recipes.
8. Runtime metrics and release/package behavior.

For each domain:

- Port old behavior tests into Rust first where practical.
- Implement Rust core without Qt dependencies.
- Add event and error models.
- Add compatibility tests for old saved workspace JSON.
- Update matrix rows to `implemented` only after Rust tests pass.

Exit gate:

- Domain Rust tests pass.
- Old behavior differences are documented in matrix notes.
- No QML-specific logic is required to make core tests pass.

### Phase 4: Qt bridge and QML feature migration

Goal: expose migrated core behavior through native ViewModels and QML screens.

Required work:

- Implement ViewModels with stable properties, invokable methods, signals, and
  list models.
- Move long-running work off the UI thread and return results through Qt signal
  handoff.
- Implement QML screens using ViewModels only.
- Bind every QML command/control to a feature ID.
- Keep visible UX close to the old behavior until the matching feature is
  accepted. Design changes require their own matrix rows and acceptance notes.

Exit gate:

- QML smoke tests or manual scripts cover every first-wave visible command.
- UI state updates come from ViewModels, not duplicated QML business state.
- Feature rows move to `tested` only when Rust and UI tests both exist or an
  explicit hardware/manual verification reason is recorded.

### Phase 5: Automation, MCP, and compatibility

Goal: restore external automation and saved data compatibility.

Required work:

- Recreate MCP tool names, argument shapes, structured result shapes, and
  safety boundaries unless intentionally changed.
- Restore graph demo/template/catalog behavior.
- Validate old workspace and graph snapshots.
- Provide migration or clear rejection for incompatible saved files.
- Re-run documented MCP recipes against the new implementation.

Exit gate:

- `docs/mcp-api.md` tool groups have migrated or explicitly removed/deferred
  rows in the matrix.
- Safe no-hardware MCP recipes pass on the new app or headless mode.
- Old demo workspaces load or have explicit migration notes.

### Phase 6: Release hardening

Goal: prove the new app is shippable and the old app can be retired.

Required work:

- Build release artifacts for the target OS matrix.
- Verify Qt runtime deployment, installer behavior, version metadata, logs,
  crash dumps if available, and checksums.
- Run cross-screen/DPI/minimize/restore GUI smoke on Windows.
- Run no-hardware smoke and hardware smoke.
- Compare feature matrix accepted count against old source inventory.
- Archive old Wails-specific recovery patches or mark them obsolete.

Exit gate:

- Matrix has zero `inventory`, `contracted`, `implemented`, or `tested` rows
  for release scope.
- Every release-scope row is `accepted`, `removed`, or `deferred`.
- Release owner signs off on artifact smoke.
- Verification owner signs off on behavior acceptance.
- Migration lead signs off on architecture boundary and known risks.

## Work breakdown by domain

### Shell and workspace

Feature rows:

- `shell.launch`
- `shell.window_lifecycle`
- `workspace.new`
- `workspace.open`
- `workspace.save`
- `workspace.save_as`
- `workspace.restore_last`
- `workspace.dirty_state`
- `settings.theme`
- `settings.serial_defaults`

Rust responsibilities:

- Workspace schema load/save.
- Recent workspace path persistence.
- Settings persistence and defaults.
- Compatibility with old `portweave.workspace.v1` and `portweave.graph.v1`
  snapshots.

Qt/C++ responsibilities:

- File dialogs.
- Recent file actions.
- Window title and dirty marker.
- Theme preference handoff to QML.
- Native lifecycle and close-confirm behavior.

QML responsibilities:

- Toolbar actions.
- Empty, loading, dirty, error, and saved states.
- Activity/sidebar/editor/status shell.

Acceptance:

- New/open/save/save-as match old behavior.
- Old demo and saved files load.
- Dirty state changes on graph/settings changes and clears after save.
- App remains visible after cross-screen drag, resize, minimize/restore, and
  DPI changes.

### Serial and virtual serial

Feature rows:

- `serial.enumerate`
- `serial.open_close`
- `serial.send_bytes`
- `serial.status_events`
- `serial.monitor_frames`
- `serial.buffer_query_clear`
- `serial.virtual_port`
- `serial.virtual_pair`
- `serial.bridge`
- `serial.cleanup_virtual`

Rust responsibilities:

- Platform serial backend abstraction.
- Handle lifecycle and duplicate-open behavior.
- Read/write worker tasks.
- Status/event stream.
- Ring buffers and monitor frame storage.
- Virtual serial creation/deletion/listing.

Qt/C++ responsibilities:

- Serial list model.
- Open handle/status model.
- Async command wrapper and error mapping.

QML responsibilities:

- Port list refresh.
- Connect/disconnect/send controls.
- Monitor and buffer displays.
- Virtual pair/bridge management panels.

Acceptance:

- No-device enumerate case is handled.
- Physical port open/close/send path works when hardware is available.
- Virtual pair loopback works without hardware.
- Duplicate, missing, busy, and permission errors are visible and classified.

### Graph runtime

Feature rows:

- `graph.document`
- `graph.node_catalog`
- `graph.edit_nodes_edges`
- `graph.layout_tabs`
- `graph.validate`
- `graph.runtime_start_stop`
- `graph.runtime_status`
- `graph.runtime_send`
- `graph.runtime_query_buffers`
- `graph.runtime_query_frames`
- `graph.script_node`
- `graph.remote_node`

Rust responsibilities:

- Graph document types and validation.
- Runtime resource ownership.
- Byte routing and fan-out.
- Node buffers, frames, counters, cleanup.
- Script node execution.
- Remote raw TCP behavior.

Qt/C++ responsibilities:

- Graph document ViewModel.
- Node/edge/list models.
- Runtime state and operation log models.
- Thread handoff for runtime events.

QML responsibilities:

- Graph editor.
- Node inspector.
- Edge and runtime state displays.
- Editor tabs/split layout.

Acceptance:

- Existing demo graphs load.
- Validate/start/status/send/query/stop flow matches current MCP recipes.
- Runtime cleanup closes resources.
- Script errors and validation errors are visible and testable.

### Protocols

Feature rows:

- `modbus.master_session`
- `modbus.slave_session`
- `modbus.read_registers`
- `modbus.scan_registers`
- `modbus.register_tables`
- `fecbus.parse_frame`
- `fecbus.build_frame`
- `fecbus.service`
- `protocol.template_catalog`
- `protocol.template_describe`

Rust responsibilities:

- Protocol encoders/decoders.
- Register/coils data models.
- Scan/read operations.
- Template catalog and descriptions.

Qt/C++ responsibilities:

- Protocol ViewModels and table models.
- Result/error delivery.

QML responsibilities:

- Register/coils tables.
- Read/scan forms.
- Frame displays and template pickers.

Acceptance:

- Existing protocol unit tests pass in Rust.
- Register tables preserve old workspace state.
- MCP protocol tool results match old shapes unless intentionally changed.

### MCP and automation

Feature rows:

- `mcp.server_start`
- `mcp.origin_guard`
- `mcp.serial_tools`
- `mcp.graph_tools`
- `mcp.monitor_tools`
- `mcp.modbus_tools`
- `mcp.fecbus_tools`
- `mcp.protocol_template_tools`
- `mcp.docs_recipes`

Rust responsibilities:

- MCP server/tool implementation or service endpoint backing it.
- Tool schemas, safety checks, structured results, and error mapping.
- Automation recipe compatibility.

Qt/C++ responsibilities:

- App lifecycle hook to start/stop automation service if in-process.
- Status ViewModel for server health and endpoint.

QML responsibilities:

- MCP status panel and endpoint display if the current product keeps it.

Acceptance:

- Tool list matches the matrix.
- Safe no-hardware recipes pass.
- Destructive/send/write tools retain safety labeling and local trust boundary.

## PR workflow

Every PR must be tied to feature rows. Avoid broad PRs that mix unrelated
domains.

Recommended PR sequence:

1. ADR and matrix scaffolding.
2. Skeleton app and CI build.
3. Workspace/config compatibility.
4. Serial enumerate/open/close.
5. Serial send/monitor/buffer.
6. Virtual serial pair/bridge.
7. Graph document/catalog/editor basics.
8. Graph runtime validate/start/status/stop.
9. Script/remote/protocol graph nodes.
10. Modbus and FECbus.
11. MCP tool parity.
12. Release packaging and final acceptance.

PR template:

```text
Feature IDs:

Matrix rows changed:

Old behavior covered:

New Rust API:

New Qt/C++ surface:

New QML entry:

Tests run:

Acceptance evidence:

Known differences:

Out-of-scope changes:
```

Review rules:

- Reject code without a feature ID unless it is pure infrastructure listed in
  the PR.
- Reject QML business logic that duplicates Rust core behavior.
- Reject Rust code that depends on Qt/QML UI concepts.
- Reject ViewModels that hide protocol or persistence rules.
- Reject `done` claims without tests or explicit manual evidence.
- Reject drive-by refactors outside the PR feature IDs.
- Require `opus-qml-design-lead` review for QML PRs that change navigation,
  editor layout, graph editing, serial panels, protocol panels, settings, or
  global state presentation.
- Treat `opus-qml-design-lead` approval as design approval only. It does not
  replace QML smoke, screenshot review, feature matrix closure, or final
  acceptance.

## Verification gates

Use layered verification. A GUI that looks correct is not enough.

### Matrix completeness gate

Add a script that fails CI when:

- A current Wails binding is not represented in the matrix.
- A current MCP tool is not represented in the matrix.
- A visible shell command from the old app is not represented in the matrix.
- A row has `accepted` status without `new_tests` and `acceptance`.
- A row has `removed` or `deferred` status without `notes`.
- A QML file adds an actionable control without a `featureId` property or
  adjacent feature ID annotation.

The first version can be simple and conservative. It is acceptable for the
script to require explicit allowlists for generated files and purely visual
controls.

### Rust core gate

Run:

```bash
cargo test --workspace
cargo clippy --workspace --all-targets -- -D warnings
cargo fmt --check
```

Add domain-specific tests for:

- Workspace roundtrip and old JSON compatibility.
- Serial backend no-device behavior and virtual pair loopback.
- Modbus/FECbus parser and encoder golden cases.
- Graph validation and runtime cleanup.
- MCP tool schema and result shapes.

### Qt/QML gate

Run the selected Qt test/build commands from the new build system. The final
commands must be documented in this file after the skeleton PR chooses CMake,
Cargo-driven Qt, or hybrid CMake/Cargo.

Minimum expected checks:

```text
configure/build debug
QTest ViewModel tests
QML smoke/load tests
package resource load check
```

### GUI smoke gate

On Windows, every release candidate must verify:

- Launch main window.
- Cross-screen drag between monitors.
- Resize on both monitors.
- Minimize and restore.
- DPI boundary movement if monitors differ.
- Open/save workspace.
- Start/stop no-hardware demo graph.
- Virtual serial pair loopback.

Use screenshots or window-capture evidence for visual regressions. The pass
criteria is not just "process did not crash"; visible content must remain
rendered and interactive.

For QML-heavy changes, verification must also check the design contract:

- Each actionable control has a feature ID or adjacent feature ID annotation.
- Loading, empty, error, dirty, running, stopped, and unsupported states render.
- Text fits in narrow and wide window sizes.
- Keyboard focus and tab order work for the changed screen.
- The screen remains usable after theme changes and window lifecycle smoke.
- Design-review comments from `opus-qml-design-lead` are resolved or explicitly
  rejected by the migration lead with a matrix note.

### Hardware gate

When hardware is available, verify:

- Serial enumerate sees the expected device.
- Open/close works repeatedly.
- Send/receive or monitor evidence is captured.
- Modbus/FECbus operations match a known device or loopback fixture.

If hardware is unavailable for a release, mark affected rows `deferred` or
record a manual-risk exception approved by the migration lead and verification
owner.

## Daily operating rhythm

Daily migration review should use three lists only:

```text
1. Rows moved to accepted since yesterday.
2. Rows blocked and the exact missing link.
3. Rows still unmapped, deferred, or removed.
```

Avoid status reports like "UI is mostly done" or "serial is almost finished".
Progress is counted by feature rows moving through the approved status flow.

Weekly migration review should check:

- Matrix status distribution by domain.
- Old source inventory changes and new unmapped items.
- Test coverage and smoke evidence.
- Cross-layer boundary violations.
- Release risks and hardware availability.

## Anti-drift rules

These are hard rules during migration:

- No feature ID, no feature PR.
- No matrix row, no acceptance.
- No Rust test or explicit manual evidence, no `tested` status.
- No acceptance evidence, no `accepted` status.
- No direct QML-to-Rust calls.
- No QML-owned protocol, serial, workspace, or graph runtime rules.
- No silent behavior changes.
- No removal without reason and owner.
- No broad rewrite PRs that cannot be reviewed against feature IDs.

## Final cutover checklist

Before retiring the old Wails app:

- All release-scope matrix rows are `accepted`, `removed`, or `deferred`.
- Removed/deferred rows are approved and documented in release notes.
- Old workspace/demo files load or have migration notes.
- MCP safe recipes pass or are documented as removed/deferred.
- Windows cross-screen/DPI/minimize/restore smoke passes.
- Release artifacts have existence checks, dependency checks, and checksums.
- Crash/log paths are documented.
- Users can identify the new version and migration notes from the app or
  release package.
- The old app remains buildable or is archived with a tagged final release.

## Templates

### Feature row review

```text
Feature ID:
Owner:
Current status:

Old entry points:
- UI:
- frontend state:
- binding:
- backend:
- MCP:
- docs:
- tests:

New implementation:
- Rust API:
- Qt/C++ ViewModel:
- QML entry:
- tests:

Acceptance evidence:

Differences from old behavior:

Decision:
- accept / keep open / remove / defer
```

### Gap report

```text
Date:
Generated by:

Unmapped old bindings:

Unmapped MCP tools:

Unmapped visible commands:

Rows missing tests:

Rows missing acceptance evidence:

Rows marked removed/deferred without reason:

Recommended next owners:
```

### Domain handoff

```text
Domain:
Owner:
Feature IDs:
Completed rows:
Open rows:
Rust APIs:
Qt ViewModels:
QML files:
Tests:
Manual evidence:
Known risks:
Next PR:
```

## Expected outcome

Following this playbook should produce:

- A complete feature inventory.
- A controlled Qt/QML + Rust implementation path.
- Clear ownership for every human and agent.
- A testable contract for every migrated capability.
- No silent loss of Wails bindings, MCP tools, workspace fields, protocol
  behavior, graph runtime behavior, or visible commands.
- A final release decision based on evidence rather than confidence.
