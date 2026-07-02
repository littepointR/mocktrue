
# PortWeave Qt/Rust Skeleton

This directory contains the first Qt/QML + Rust skeleton required by the
migration plan. It intentionally proves only the shell, the Rust C ABI smoke
call, and the window lifecycle probe.

## Prerequisites

On macOS with Homebrew Qt:

```bash
export QT_ROOT="$(brew --prefix qt)"
export CMAKE_PREFIX_PATH="$QT_ROOT"
export PATH="$QT_ROOT/bin:$PATH"
```

The repository preflight target expects a `powershell` command on PATH. If
your shell only has `pwsh`, run the check script directly:

```bash
pwsh -ExecutionPolicy Bypass -File scripts/check-qt-rust-env.ps1
python scripts/check-migration-matrix.py
```

## Configure

```bash
cmake -S desktop/qt-rust -B build/qt-rust -G Ninja -DCMAKE_BUILD_TYPE=Debug
```

## Build

```bash
cmake --build build/qt-rust
```

## Run

```bash
./build/qt-rust/PortWeaveQtRust
```

The executable loads `build/qt-rust/qml/Main.qml` from the build tree after
copying the `qml/` directory beside the binary.
