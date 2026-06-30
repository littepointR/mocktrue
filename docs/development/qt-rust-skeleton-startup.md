# Qt/Rust Skeleton Startup Package

This is the handoff from migration preparation into skeleton development. Use it
as the first development card. It is intentionally limited to proving the new
desktop stack before any product feature is ported.

## Current Decision

Skeleton development may start.

Feature implementation beyond the skeleton must not start until all skeleton
exit gates in this document pass.

## Load Environment

Run all Qt/Rust development commands from a shell where the Qt/Rust
prerequisites are available.

Each developer may either configure the environment directly or point the
preflight script at a private machine-local bootstrap script:

```powershell
$env:PORTWEAVE_QT_RUST_ENV_SCRIPT = "<path-to-local-bootstrap.ps1>"
```

The bootstrap script is not part of the repository and must not be referenced by
absolute path in shared docs. It should set or load whatever is needed for:

- `QT_ROOT`
- `CMAKE_PREFIX_PATH`
- Rust/Cargo on `PATH`
- Qt tools on `PATH`
- CMake on `PATH`
- MSVC C++ build tools on `PATH`
- Ninja on `PATH`

Before editing the skeleton, run:

```powershell
powershell -ExecutionPolicy Bypass -File scripts\check-qt-rust-env.ps1
python scripts\check-migration-matrix.py
```

Or run both through the Makefile:

```powershell
make migration-preflight
```

Expected environment:

- `QT_ROOT` points to a Qt 6 compiler kit root.
- `CMAKE_PREFIX_PATH` includes `QT_ROOT`.
- `rustc`, `cargo`, `cmake`, `qmake`, `cl.exe`, and `ninja` are available.

## Skeleton Directory

Create the new stack under:

```text
desktop/qt-rust/
```

Initial layout:

```text
desktop/qt-rust/
  CMakeLists.txt
  app/
    main.cpp
    AppBridge.h
    AppBridge.cpp
    WindowLifecycleProbe.h
    WindowLifecycleProbe.cpp
  qml/
    Main.qml
    shell/
      MainWindow.qml
      MainToolbar.qml
      ActivityBar.qml
      Sidebar.qml
      StatusBar.qml
  rust/
    Cargo.toml
    src/
      lib.rs
      event.rs
      error.rs
```

Keep the existing Go/Wails app intact. Do not move `frontend/`, `internal/`,
`main.go`, or existing Wails configuration in the skeleton PR.

## First Development Card

Feature IDs:

- `shell.window_lifecycle`

Allowed files/modules:

- `desktop/qt-rust/**`
- `scripts/check-qt-rust-env.ps1` only if the environment check needs a narrow
  correction
- `docs/development/qt-rust-skeleton-startup.md` only for command updates
- `docs/development/migration-gap-report.md` only for evidence updates

Read first:

- `docs/development/architecture-decisions.md` ADR-0006
- `docs/development/qt-rust-migration-playbook.md`
- `docs/development/migration-contracts.md`
- `docs/development/migration-feature-matrix.csv`
- `docs/development/qml-first-wave-design.md`

Required output:

- CMake-configured Qt 6 app that starts from `desktop/qt-rust/`.
- QML main window rendered from Qt resources or a documented local QML path.
- Rust `staticlib` built by Cargo.
- Narrow C ABI smoke function callable from Qt/C++.
- One Rust-originated event/result delivered into a Qt ViewModel and shown in
  QML.
- Window lifecycle probe or smoke harness covering resize, minimize, restore,
  and screen/DPI-change event logging.
- Build/run commands documented in this file or a follow-up skeleton README.

Forbidden changes:

- Do not port workspace, serial, graph, protocol, MCP, or installer behavior.
- Do not let QML call Rust directly.
- Do not make Rust depend on Qt/QML.
- Do not remove or rewrite Wails code.
- Do not commit `.hermes/`, scratch logs, or generated local state.

## Agent Assignments

Use these roles for the skeleton PR:

| Agent | Responsibility | Write Scope | Output |
| --- | --- | --- | --- |
| `architect-agent` | Guard ADR boundary and first card scope | docs only when scope changes | Scope review and gate decision |
| `qt-bridge-agent` | Qt entry, CMake, QML engine, C ABI wrapper, lifecycle probe | `desktop/qt-rust/app/**`, `desktop/qt-rust/CMakeLists.txt` | Launching Qt shell and bridge call |
| `rust-core-agent` | Rust crate, exported C ABI, event/error smoke types | `desktop/qt-rust/rust/**` | Cargo staticlib and testable smoke API |
| `qml-shell-agent` | Minimal QML shell surface and feature annotations | `desktop/qt-rust/qml/**` | Visible shell with event display |
| `verification-agent` | Preflight, build, run, lifecycle smoke, matrix check | read-only except evidence docs | Exact commands and pass/fail evidence |
| `opus-qml-design-lead` | Design review of shell layout and state mapping | read-only | Design findings only |

Agents are not alone in the codebase. They must inspect current dirty state,
avoid unrelated files, and never revert changes they did not make.

## Build Command Shape

The skeleton should use CMake as the outer build because Qt application setup is
native there. CMake may invoke Cargo for the Rust `staticlib`.

Recommended first command shape after files exist:

```powershell
cmake -S desktop\qt-rust -B build\qt-rust -G Ninja -DCMAKE_BUILD_TYPE=Debug
cmake --build build\qt-rust
```

The first run command should be documented by the skeleton PR after the target
name is created, for example:

```powershell
.\build\qt-rust\PortWeaveQtRust.exe
```

## Skeleton Exit Gates

The skeleton is complete only when:

- `powershell -ExecutionPolicy Bypass -File scripts\check-qt-rust-env.ps1`
  passes.
- `python scripts\check-migration-matrix.py` passes.
- CMake configure and build pass for `desktop/qt-rust`.
- The Qt/QML app launches.
- QML content is visible on first launch.
- A Rust-originated smoke value/event is visible in QML.
- Moving the window between monitors keeps QML content visible.
- Resize, minimize, and restore keep QML content visible and interactive.
- Lifecycle/DPI/screen events are logged or otherwise observable.
- `git diff --check` passes.

These gates allow first-wave product feature work to begin. They do not accept
any product feature row by themselves.

## Rust Core Decision Gate

After the skeleton exit gates pass, the migration lead must make an explicit
decision before first-wave product feature implementation starts.

Continue with Rust core only when all of these are true:

- Qt/C++ can call a narrow Rust C ABI without generated bridge sprawl.
- A Rust-originated event/result reaches a Qt ViewModel and QML without Rust
  depending on Qt/QML.
- CMake/Cargo integration rebuilds repeatably from a clean `build/qt-rust`
  directory.
- The bridge code remains limited to type conversion, error conversion,
  async/thread handoff, and event delivery.
- Rust unit tests can exercise the smoke core without launching Qt.
- Debugging a failed smoke event has a clear path in both Rust and Qt/C++.

Reconsider Rust before feature work if any of these are true:

- The C ABI requires broad generated binding code or large manual wrappers.
- QML needs to call Rust directly to keep the UI usable.
- Rust needs Qt/QML types to report state.
- The build cannot be made reliable on the developer machines and CI target.
- The bridge adds more failure modes than it removes for the first-wave scope.

If Rust is rejected at this gate, write a follow-up ADR before continuing. The
ADR must compare continuing Rust core, switching to pure Qt/QML + Qt/C++ core,
and changing the bridge to `cxx-qt` or IPC.

After the skeleton build command exists, add CI coverage for
`make migration-preflight` and the new Qt/Rust configure/build command. Do not
wire Qt/Rust CI before the build target exists.

After QML action controls exist, extend `scripts/check-migration-matrix.py` so
actionable controls must carry a `featureId` property or adjacent
`// feature: <id>` annotation.

## Stop Conditions

Stop and update the migration lead if any of these occur:

- Qt/QML content disappears during cross-screen movement, resize, minimize, or
  restore.
- The C ABI bridge needs broad generated binding code before the smoke works.
- Rust needs to depend on Qt or QML to deliver the smoke event.
- CMake/Cargo integration cannot build repeatably from a clean `build/qt-rust`
  directory.
- The matrix check fails because the skeleton changed feature scope.

If the C ABI bridge becomes hard to maintain before first-wave product
implementation starts, write a follow-up ADR comparing C ABI, `cxx-qt`, and IPC
using evidence from the skeleton.

## Ready State

When this startup package exists, environment preflight passes, and the matrix
gate passes, preparation is complete. The next task can be:

```text
Create the Qt/QML + Rust skeleton under desktop/qt-rust and satisfy the
skeleton exit gates in docs/development/qt-rust-skeleton-startup.md.
```
