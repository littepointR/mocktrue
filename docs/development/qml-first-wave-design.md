# QML First-Wave Design Handoff

This is the `opus-qml-design-lead` handoff for first-wave skeleton and feature
work. It is a design contract, not implementation acceptance.

## Screen Inventory

First-wave QML screens:

- `qml/shell/MainWindow.qml`: root window content, theme surface, global layout.
- `qml/shell/MainToolbar.qml`: new, open, save, save as, runtime actions.
- `qml/shell/ActivityBar.qml`: serial and settings module activation.
- `qml/shell/Sidebar.qml`: active module views and graph/demo navigation.
- `qml/shell/EditorGroups.qml`: one active editor group in the first wave.
- `qml/shell/StatusBar.qml`: runtime metrics, workspace path, dirty marker,
  MCP/server placeholder.
- `qml/settings/GlobalSettingsPanel.qml`: theme and demo catalog.
- `qml/settings/SerialSettingsPanel.qml`: serial defaults placeholder for
  first-wave persistence.
- `qml/serial/SerialPortPanel.qml`: refresh ports, open, close, open handle
  list, error state.
- `qml/serial/VirtualPairPanel.qml`: backend status, pair list, create pair,
  delete pair, unsupported state.
- `qml/graph/GraphEditor.qml`: first-wave graph document view with node/edge
  presence and active graph selection; advanced editing remains out of scope.
- `qml/graph/RuntimeToolbar.qml`: validate, start, stop, status refresh.
- `qml/graph/RuntimeStatusPanel.qml`: graph status, node status, counters,
  last error.
- `qml/graph/NodeBufferPanel.qml`: node buffer query, clear, reset counters.

Out of first-wave scope:

- Modbus-specific panels.
- FECbus-specific panels.
- MCP tool browser/editor.
- Advanced graph drag/drop editing.
- Packaging/update UI.

## Feature-to-Control Mapping

| Control | Feature ID |
| --- | --- |
| MainToolbar.New | `workspace.new` |
| MainToolbar.Open | `workspace.open` |
| MainToolbar.Save | `workspace.save` |
| MainToolbar.SaveAs | `workspace.save_as` |
| AppStartup.LoadLast | `workspace.restore_last` |
| MainWindow.DirtyMarker | `workspace.dirty_state` |
| GlobalSettings.ThemeSelector | `settings.theme` |
| ActivityBar.Serial | `shell.module_navigation` |
| ActivityBar.Settings | `shell.module_navigation` |
| SerialPort.Refresh | `serial.enumerate` |
| SerialPort.Open | `serial.open_close` |
| SerialPort.Close | `serial.open_close` |
| VirtualPair.BackendStatus | `serial.virtual_backend_status` |
| VirtualPair.Create | `serial.virtual_pair` |
| VirtualPair.Delete | `serial.virtual_pair` |
| DemoCatalog.Load | `workspace.demo_catalog`; `graph.demo_catalog` |
| GraphEditor.DocumentLoad | `graph.document` |
| RuntimeToolbar.Validate | `graph.validate` |
| RuntimeToolbar.Start | `graph.runtime_start_stop` |
| RuntimeToolbar.Stop | `graph.runtime_start_stop` |
| RuntimeStatus.Refresh | `graph.runtime_status` |
| NodeBuffer.Query | `graph.runtime_query_buffers` |
| NodeBuffer.Clear | `graph.runtime_query_buffers` |
| NodeBuffer.ResetCounters | `graph.runtime_query_buffers` |

Every QML action control should expose either `property string featureId` or an
adjacent `// feature: <id>` comment until a shared feature annotation helper
exists.

## State Model

Global states:

- `idle`: no operation running.
- `loading`: workspace, demo, port list, or runtime query in progress.
- `dirty`: current graph/workspace differs from saved content.
- `saved`: save completed and dirty cleared.
- `error`: last operation failed with a classified error.

Serial states:

- `empty`: no ports or no open handles.
- `unsupported`: virtual backend not available on this machine.
- `permission_denied`: OS denied opening or creating the resource.
- `busy`: refresh/open/close/create/delete in progress.

Runtime states:

- `stopped`: no runtime for active graph.
- `running`: runtime exists and resources are active.
- `degraded`: runtime exists but one or more nodes report an error.
- `stopping`: stop requested and cleanup is in progress.

QML displays states from Qt ViewModels. It must not parse backend error strings
to infer state.

## Navigation Model

Initial navigation:

```text
ActivityBar.Serial -> Sidebar.SerialGraph -> Editor.Graph
ActivityBar.Settings -> Sidebar.GlobalSettings / Sidebar.SerialSettings
```

First-wave graph editor uses one editor group. Multi-group split behavior is
documented in the matrix as `graph.layout_tabs` but remains inventory until the
skeleton and first-wave smoke pass.

## Required ViewModel Surface

QML design assumes these ViewModels exist:

- `ProjectViewModel`
- `ModuleRegistryViewModel`
- `SettingsViewModel`
- `SerialPortViewModel`
- `VirtualSerialViewModel`
- `GraphDocumentViewModel`
- `GraphRuntimeViewModel`
- `GraphRuntimeStatusModel`
- `GraphNodeBufferModel`

The ViewModels own truth for loading, dirty, error, running, stopped,
unsupported, and permission states.

## Design Review Checklist

Before QML implementation PRs are accepted:

- Each actionable control maps to a matrix feature ID.
- Empty, loading, error, unsupported, dirty, running, and stopped states render
  where applicable.
- Text fits in narrow and wide windows.
- Controls remain reachable by keyboard focus.
- Theme switch does not hide text or controls.
- Window lifecycle smoke keeps visible content interactive.
- No QML code duplicates serial, workspace, graph runtime, or protocol logic.
