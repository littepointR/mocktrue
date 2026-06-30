# Migration Readiness Checklist

This checklist is the gate between planning and development. Do not start
feature implementation until all required items are complete or explicitly
waived by the migration lead and verification owner.

## Required before development

- [x] Migration playbook exists.
- [x] Initial feature matrix exists.
- [x] First-wave contracts exist.
- [x] First gap report exists.
- [x] First-wave feature rows are identified.
- [x] Existing source anchors for UI, bindings, Go services, MCP, docs, and
  tests are documented.
- [x] New ADR is accepted and supersedes the old no-migration non-decision.
- [x] Rust toolchain can be verified through `make migration-preflight`.
- [x] Qt 6 compiler kit can be verified through `make migration-preflight`.
- [x] `opus-qml-design-lead` first-wave screen inventory and state model are
  reviewed.
- [x] New stack skeleton PR scope is approved.
- [x] Matrix completeness script exists.
- [x] Current baseline tests are run and recorded.
- [x] Qt/Rust skeleton startup package exists.
- [x] Local migration preflight target exists: `make migration-preflight`.

## Development may start when

Development may start for the skeleton only when:

- The ADR is accepted.
- Rust and Qt build prerequisites are available in the current shell or loaded
  through a machine-local bootstrap script referenced by
  `PORTWEAVE_QT_RUST_ENV_SCRIPT`.
- `make migration-preflight` passes locally.
- The skeleton scope is limited to launch, QML rendering, Rust event delivery,
  and Windows window lifecycle smoke.

Feature implementation may start when:

- The skeleton launches.
- A Rust event reaches QML through the selected bridge.
- Windows cross-screen/minimize/restore smoke passes.
- First-wave feature rows remain `contracted`.

## First implementation order

1. Skeleton app and build system.
2. Window lifecycle smoke.
3. `opus-qml-design-lead` first-wave QML screen inventory and state model.
4. Workspace load/save/new graph.
5. Theme and module navigation.
6. Serial enumerate/open/close.
7. Virtual serial backend status and pair loopback.
8. No-hardware graph demo load/validate/start/status/buffer/stop.

Do not start Modbus, FECbus, broad MCP parity, installer packaging, or complex
graph editing before this order is complete.

## Current blockers

No planning/tooling blockers remain for skeleton development.

The next allowed development task is:

```text
Create the Qt/QML + Rust skeleton under desktop/qt-rust and satisfy the
skeleton exit gates in docs/development/qt-rust-skeleton-startup.md.
```

Feature implementation beyond the skeleton remains blocked until:

- The skeleton launches.
- A Rust event reaches QML through the C ABI bridge.
- Windows cross-screen/minimize/restore smoke passes.
- First-wave feature rows remain `contracted`.
