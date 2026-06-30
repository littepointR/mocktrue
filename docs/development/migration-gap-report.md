# Migration Gap Report

Generated during migration preparation. This report lists what is ready, what is
blocked, and what must be resolved before implementation begins.

## Current environment

Confirmed:

- Repository root: `D:\Users\pc\Documents\WorkSpace\git\mocktrue`.
- Qt installations exist under `D:\Qt`, including `D:\Qt\6.11.0`.
- CMake is available as `C:\Python314\Scripts\cmake.exe`, version `4.3.1`.
- Rust is installed under `D:\Tools`:
  - `RUSTUP_HOME=D:\Tools\rustup`
  - `CARGO_HOME=D:\Tools\cargo`
  - active toolchain `stable-x86_64-pc-windows-msvc`, `rustc 1.96.0`.
- Visual Studio 2022 Build Tools are available through
  `C:\Program Files (x86)\Microsoft Visual Studio\2022\BuildTools\Common7\Tools\VsDevCmd.bat`.
- Qt 6.11.0 MSVC kit is available at `D:\Qt\6.11.0\msvc2022_64`.
- Reusable environment bootstrap script: `D:\Tools\portweave-env.ps1`.
- Reusable local preflight script: `scripts\check-qt-rust-env.ps1`.
- Existing generated Wails bindings are under `frontend/bindings`.
- Existing frontend stack remains Vue/TypeScript under `frontend/src`.
- Existing backend stack remains Go/Wails under `internal`.

Baseline checks passed during preparation:

- `powershell -ExecutionPolicy Bypass -File scripts\check-qt-rust-env.ps1`
- `make migration-preflight`
- `python scripts\check-migration-matrix.py`
- `make migration-matrix-check`
- `go test ./internal/core/... ./internal/modules/serial/... -count=1`
- `cd frontend && pnpm exec vitest run src/workspace/workspaceSession.test.ts src/workspace/stores/workspaceFileStore.test.ts src/serial/stores/virtualStore.test.ts src/core/registry.test.ts src/settings/views/serial/SerialSettingsPanel.test.ts src/serial/stores/monitorStore.test.ts`
- `cd frontend && pnpm exec vue-tsc --noEmit`

Toolchain smoke passed:

- `. D:\Tools\portweave-env.ps1; rustc --version`
- `. D:\Tools\portweave-env.ps1; cargo --version`
- `. D:\Tools\portweave-env.ps1; cl`
- `. D:\Tools\portweave-env.ps1; qmake -query QT_VERSION`
- `. D:\Tools\portweave-env.ps1; cargo new --bin $env:TEMP\portweave-rust-msvc-smoke --name portweave_rust_msvc_smoke; cargo run --manifest-path $env:TEMP\portweave-rust-msvc-smoke\Cargo.toml`

## Inventory coverage

Covered by the first feature matrix:

- Shell window lifecycle and module navigation.
- Workspace new/open/save/save-as/restore-last/dirty-state/demo catalog.
- Settings theme and serial defaults.
- Runtime metrics.
- Serial enumerate/open/close/send/buffer/monitor.
- Virtual serial backend status, ports, pairs, bridges, cleanup.
- Graph document, catalog, editor, validation, runtime, buffers, frames, script,
  remote node.
- Modbus, FECbus, protocol templates.
- MCP serial, graph, Modbus, FECbus, protocol template tool groups.
- Windows release/package smoke.

Source evidence used:

- `frontend/src/App.vue`.
- `frontend/src/main.ts`.
- `frontend/src/core/registry.ts`.
- `frontend/src/settings`.
- `frontend/src/serial`.
- `frontend/src/workspace`.
- `frontend/bindings/.../workspace/service.ts`.
- `frontend/bindings/.../runtime/service.ts`.
- `frontend/bindings/.../serial/service.ts`.
- `internal/core/workspace/service.go`.
- `internal/core/runtime/service.go`.
- `internal/modules/serial/service.go`.
- `internal/modules/serial/graph_runtime.go`.
- `internal/modules/mcpserver/server.go`.
- `internal/modules/mcpserver/server_test.go`.
- Existing docs under `docs/`.

## Known gaps

These are not blockers for the first-wave skeleton, but they must be resolved
before broader migration acceptance.

- Keep `scripts/check-migration-matrix.py` in CI once the new-stack build
  pipeline exists. The local gate already compares `frontend/bindings/**/*.ts`
  exported functions and MCP tool names against `migration-feature-matrix.csv`.
- Extend the matrix gate after QML skeleton files exist so visible QML action
  controls must carry a `featureId` property or adjacent `// feature: <id>`
  annotation.
- Extract exact old workspace JSON fixtures from current demo workspaces and
  saved files for Rust compatibility tests.
- The first-stage Rust/Qt bridge is selected: C ABI from Qt/C++ to a Rust
  `staticlib`. Revisit only if the smoke bridge becomes hard to maintain before
  first-wave feature implementation.
- The first skeleton should use a CMake-first Qt app that invokes Cargo for the
  Rust static library until a dedicated build ADR says otherwise.
- Decide whether MCP runs inside the Qt desktop process or as a separate Rust
  service process.
- Decide the final Windows virtual serial backend strategy and install/signing
  requirements.
- Define release artifact location, installer format, and dependency smoke for
  Qt runtime deployment.
- `opus-qml-design-lead` first-wave screen inventory, state model, navigation
  model, and feature-ID-to-control mapping are captured in
  `qml-first-wave-design.md`.

## First-wave readiness

Ready:

- Feature IDs and owners are assigned for the first wave.
- First-wave contracts are documented in `migration-contracts.md`.
- Matrix rows for the first wave are `contracted`.
- Skeleton startup package is documented in `qt-rust-skeleton-startup.md`.
- Matrix completeness gate passes and currently covers 53 feature rows, 74 Wails
  binding exports, and 64 MCP tools.
- No first-wave row depends on Modbus/FECbus full parity.
- No first-wave row requires physical hardware if virtual backend is available.
- Current baseline checks for core/serial, selected migration-related frontend
  tests, and frontend typecheck passed.

Not ready:

- New Qt/QML + Rust skeleton project files do not exist yet.
- No CI job exists yet for the new Qt/Rust stack.

## Required next actions

Before implementation:

1. Load `D:\Tools\portweave-env.ps1` before Rust/Qt development commands.
2. Run `make migration-preflight`.
3. Create the skeleton PR:
   - Qt app entry.
   - QML main window.
   - Rust core crate.
   - Rust-to-Qt event proof.
   - Windows cross-screen smoke harness.
4. Add CI coverage for the new stack once the skeleton build command exists.

Planning/tooling preparation is sufficient to begin the skeleton. Product
feature implementation remains blocked until the skeleton launches, a Rust event
reaches QML through the C ABI bridge, and the Windows lifecycle smoke passes.
