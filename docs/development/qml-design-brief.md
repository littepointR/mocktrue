# QML Design Brief

This brief defines the work package for `opus-qml-design-lead`. It turns the
current Vue shell into a first-wave QML design contract without allowing design
work to bypass feature parity, implementation, or verification gates.

## Role

`opus-qml-design-lead` is responsible for front-end design judgment:

- Information architecture.
- Screen inventory.
- Interaction flow.
- QML state model.
- Feature-to-screen mapping.
- Design review of QML PRs.

It is not responsible for final implementation acceptance. A QML feature is
accepted only after implementation, feature ID mapping, Qt/QML smoke, and
verification evidence pass.

## First-wave screen inventory

The first QML design pass must cover only first-wave feature rows:

- Main window and lifecycle states.
- Activity bar and module navigation.
- Sidebar.
- Toolbar: new, open, save, save as.
- Editor area for one active graph.
- Status bar.
- Global settings with theme switch.
- Serial port panel: refresh, open, close, open handles.
- Virtual serial panel: backend status, create pair, delete pair, list pairs.
- Graph demo picker.
- Graph validation/runtime toolbar.
- Runtime status and node buffer panel.

Do not design Modbus, FECbus, broad MCP parity, packaging UI, or advanced graph
editing in the first pass.

## Required feature mapping

Every actionable QML element in the first pass must map to one or more feature
IDs from `migration-feature-matrix.csv`.

Examples:

```text
Toolbar.NewGraph -> workspace.new
Toolbar.Open -> workspace.open
Toolbar.Save -> workspace.save
Toolbar.SaveAs -> workspace.save_as
ThemeSelector -> settings.theme
Serial.Refresh -> serial.enumerate
Serial.Open -> serial.open_close
Serial.Close -> serial.open_close
VirtualPair.Create -> serial.virtual_pair
Graph.Validate -> graph.validate
Graph.Start -> graph.runtime_start_stop
Graph.Stop -> graph.runtime_start_stop
Graph.Status -> graph.runtime_status
NodeBuffer.Query -> graph.runtime_query_buffers
```

If a desired UI control has no feature ID, add a gap report entry instead of
silently adding it to the design.

## Required state model

Every first-wave screen must define these states where applicable:

```text
idle
loading
empty
dirty
saved
running
stopped
error
unsupported
permission_denied
```

QML may display these states, but the state truth must come from Qt ViewModels.
QML must not infer protocol, serial, workspace, or graph runtime state from
local string parsing.

## Design constraints

- Preserve feature parity before broad redesign.
- Prefer dense, tool-oriented layouts over marketing-style screens.
- Keep first-wave UI focused on repeated desktop workflows.
- Do not add UI paths that bypass matrix feature IDs.
- Do not duplicate Rust core logic in QML.
- Do not rely on WebView-specific behavior or browser APIs.
- Ensure labels fit at narrow and wide window sizes.
- Include keyboard focus expectations for actionable controls.
- Include theme behavior for light, dark, and system settings.

## Design handoff

The design lead must hand off:

```text
QML screen inventory:

Feature ID to control mapping:

State model:

Navigation model:

Required ViewModel properties/methods:

Screenshots or sketches if available:

Risks and open decisions:
```

The migration lead reviews the handoff before QML implementation begins. The
verification owner turns the handoff into smoke scenarios and screenshot
checks.
