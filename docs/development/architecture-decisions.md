# Architecture Decisions

This document records current PortWeave architecture guardrails. Future work can replace these decisions with a new ADR, but should not silently drift away from them.

## ADR-0001: Keep Go + Wails + Vue/TypeScript

### Status

Accepted.

### Context

PortWeave currently uses:

- Go `1.26.0`.
- Wails v3 alpha.
- Vue 3, TypeScript, Vite, Pinia, Naive UI, Monaco, Vitest, and Playwright.
- Go MCP SDK for automation.
- Goja for script runtime.
- `go.bug.st/serial` for real serial boundaries.

PortWeave's center of gravity is serial topology orchestration, virtual/remote serial resources, protocol simulation, script processing, and automation validation.

### Decision

Keep PortWeave on Go + Wails + Vue/TypeScript. Do not rewrite PortWeave in Qt/C++ as an unplanned architecture migration.

### Consequences

- Put runtime/resource behavior in Go services.
- Put graph authoring, inspectors, and workspace state in Vue/TypeScript.
- Keep Wails as a narrow native shell and service bridge.
- Consider a stack migration only if the primary product direction changes to a high-refresh native telemetry dashboard.

## ADR-0002: MCP-first automation before gRPC

### Status

Accepted.

### Context

PortWeave already exposes a local MCP server and 62 current MCP tools covering serial resources, serial graph runtime, monitors, Modbus, and FECbus. Current automation recipes need catalog inspection, validate-before-start, start/status/send/query/stop, and cleanup.

### Decision

Keep MCP as the primary automation API. Do not add gRPC until there is evidence that MCP/HTTP cannot satisfy required streaming, typing, or integration behavior.

### Consequences

- Document and test MCP recipes first.
- Prefer schema/tool clarity and safety tiers over adding transports.
- If gRPC becomes necessary later, write an evidence-based design note first.

## ADR-0003: Topology-native visualization over dashboard breadth

### Status

Accepted.

### Context

Dashboard-heavy tools are optimized for high-refresh telemetry widgets. PortWeave's product value is the graph: endpoints, routing, scripts, protocol nodes, buffers, frames, counters, operation logs, and automation evidence.

### Decision

Prioritize topology-native debugging surfaces over broad dashboard widget parity.

Preferred UI investments:

- Node inspectors.
- Edge traffic overlays.
- Endpoint buffers and monitor frames.
- Protocol frame tables.
- Scenario timelines.
- Operation logs.
- Lightweight metrics.
- Validation and assertion reports.

### Consequences

- Do not build widget breadth only to match another product category.
- Keep docs and UI language focused on topology, protocols, scripts, and repeatable validation.

## ADR-0004: Benchmark Go hotpaths before optimizing or changing stack

### Status

Accepted.

### Context

Go is a strong fit for concurrent I/O, service lifecycles, JSON/MCP tooling, tests, and static distribution. Performance concerns should be measured in the current architecture before proposing large rewrites.

### Decision

Benchmark Go hotpaths before optimizing aggressively or changing stack.

Candidate benchmark categories:

- Graph routing and fan-out.
- Buffer append/query behavior.
- Script transform/generator/analyzer execution.
- Modbus and FECbus parsers/encoders.
- Remote raw TCP reconnect/write/read paths.

### Consequences

- Treat benchmark results as the trigger for optimization work.
- Prefer local hotpath improvements over broad architecture rewrites.
- Keep allocation and sustained-runtime behavior visible in quality docs.

## ADR-0005: Wails v3 alpha release and binding guardrails

### Status

Accepted.

### Context

`go.mod` uses Wails v3 alpha. Alpha APIs, binding generation, packaging, and runtime details can change.

### Decision

Keep Wails boundaries narrow, and treat binding/release changes as explicit artifacts.

### Consequences

- Regenerate/check bindings only when exported Go service/model signatures change.
- Review generated diffs carefully and do not stage unrelated binding churn.
- Run frontend typecheck/build after binding-sensitive changes.
- Add artifact smoke before widening release targets.
- Keep release/package docs explicit about Wails version assumptions.

## ADR-0006: Prepare a controlled Qt/QML + Rust migration

### Status

Accepted for migration preparation and skeleton development.

### Context

PortWeave has hit Windows desktop-shell reliability problems in the Wails
WebView2 host boundary, especially around cross-screen movement, DPI changes,
and WebView2 controller state. The product core remains serial topology,
protocol simulation, virtual/remote serial resources, workspace compatibility,
and MCP automation.

The migration risk is not primarily UI repainting. The larger risk is losing
behavior that currently exists across Vue stores, Wails bindings, Go services,
MCP tools, documentation, and tests.

### Decision

Prepare and start a controlled migration path to a Qt/QML desktop shell with a
Rust core. Broad feature migration must not start until the migration playbook,
feature matrix, contracts, readiness checklist, gap report, QML design handoff,
and matrix completeness gate are in place.

The intended target layering is:

```text
QML UI
  -> Qt/C++ shell, ViewModels, native window lifecycle, Qt models
  -> Rust core
```

The first skeleton will use a C ABI bridge from Qt/C++ to a Rust `staticlib`.
This is the first-stage bridge only: it keeps Qt ownership natural and avoids
committing the product to generated binding frameworks before the window
lifecycle and no-hardware smoke are proven.

First development is limited to:

- Qt application entry and QML main window.
- Rust core crate exporting one C ABI smoke function and one event/result path.
- Qt/C++ bridge wrapper and ViewModel proof.
- Windows cross-screen, resize, minimize, restore, and DPI smoke.
- No product feature implementation beyond the smoke path.

Full feature migration requires every old capability to have an explicit matrix
row and acceptance evidence.

### Consequences

- ADR-0001 remains the description of the current shipped architecture until
  the new skeleton is buildable and release scope is redefined.
- A migration PR must reference feature IDs from
  `migration-feature-matrix.csv`.
- QML must not call Rust directly.
- Rust must not depend on Qt/QML UI concepts.
- Qt/C++ owns native windows, DPI, screens, file dialogs, ViewModels, and
  UI-thread handoff.
- Rust owns workspace, serial, virtual serial, graph runtime, protocols, MCP
  behavior, and core tests.
- The first skeleton must prove Windows cross-screen/DPI/minimize/restore
  behavior before broader migration work begins.
- The first-stage C ABI must stay narrow. If it becomes hard to maintain before
  first-wave feature implementation, write a follow-up ADR comparing C ABI,
  `cxx-qt`, and IPC with concrete evidence.

## Non-decisions

These are intentionally not decided in this phase:

- A fully accepted release cutover from Wails to Qt/QML + Rust beyond the
  skeleton and first-wave migration path in ADR-0006.
- A gRPC service layer.
- A dashboard widget parity roadmap.
- A public raw TCP or public MCP hosting model.
- A release matrix broader than verified artifact smoke can support.
