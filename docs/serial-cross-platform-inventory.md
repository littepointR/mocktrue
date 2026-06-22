# Serial Cross-Platform Inventory

This inventory captures the current real serial boundaries before introducing the cross-platform backend abstraction.

## Production real serial dependencies

| File | Usage | Classification | Notes |
| --- | --- | --- | --- |
| `internal/modules/serial/port/serial.go` | Imports `go.bug.st/serial`; maps `SerialConfig` to `serial.Mode`; calls `serial.Open` in `Open`. Also owns POSIX `StartVirtualPair` via `socat`. | Primary low-level production boundary | Best first abstraction point. Existing `port.Port` already wraps read/write/timeout/close and should be reused. |
| `internal/modules/serial/port/enumerator.go` | Imports `go.bug.st/serial/enumerator`; calls `enumerator.GetDetailedPortsList`. | Primary enumeration boundary | Should become the default real backend `ListPorts` implementation. |
| `internal/modules/serial/manager/manager.go` | Calls `port.Open(req.Config)` directly. | Production service boundary | Should receive/infer a `port.Backend` and call `backend.Open`. |
| `internal/modules/serial/service.go` | Calls `manager.NewManager(activeBus)` and `port.Enumerate(ctx)` directly. | Production facade boundary | Needs default real backend injection into manager; enumeration can use backend or stay as low-level wrapper in first pass. |
| `internal/modules/serial/virtualserial/bridge.go` | Imports `go.bug.st/serial`; opens bridge endpoints directly with `serial.Open`. | Production virtual bridge boundary | Can be abstracted later; it is tightly coupled to real OS virtual serial behavior. Do not block initial manager backend on this. |
| `internal/modules/serial/virtualserial/manager.go` | Creates POSIX PTY pairs through `socat`, `/tmp`, `rm`, and `test`. | POSIX-only virtual serial boundary | Should become platform-gated or documented as POSIX-only; Windows path should be opt-in com0com, not default. |

## Test and integration real serial dependencies

| File | Usage | Classification | Notes |
| --- | --- | --- | --- |
| `internal/modules/serial/service_test.go` | Imports `go.bug.st/serial`; opens other ends of virtual pairs in several tests. | Integration-like tests inside default package | Candidate for memory backend conversion if these run in default suite. |
| `internal/modules/serial/virtualserial/virtualserial_test.go` | Imports `go.bug.st/serial`; opens socat-created pair. | Real virtual serial integration | Should be `integration` or skip when no `socat`; currently default suite passes on this machine but is not Windows-native. |
| `tests/go/integration/virtual_serial_test.go` | `//go:build integration`; uses `socat` helper and `go.bug.st/serial`. | Opt-in POSIX integration | Keep opt-in; add clearer POSIX policy/env docs. |
| `tests/automation/integration/serial_integration_test.go` | `//go:build integration`; uses `socat` helper and `go.bug.st/serial`. | Opt-in POSIX integration | Keep opt-in; likely not Windows default. |

## Existing useful abstractions

- `internal/modules/serial/port.SerialConfig` already centralizes port settings.
- `internal/modules/serial/port.Port` already abstracts opened port behavior and includes `SetReadTimeout`.
- `internal/modules/serial/manager.PortManager` owns lifecycle/read-loop behavior and is a good consumer of an injected backend.
- `internal/modules/serial/service.Service` is the frontend facade and can retain default production behavior by constructing a manager with a real backend.

## Initial cut line decision

Start inside `internal/modules/serial/port`:

1. Add `Backend` interface next to existing `Port`/`Open` concepts.
2. Add `RealBackend` that delegates to existing `Open` and `Enumerate` functions.
3. Inject `Backend` into `manager.PortManager` while preserving `manager.NewManager(bus)` default behavior.
4. Defer `virtualserial.Bridge` abstraction until after manager/service are injectable; it is POSIX/socat-centered and should not block the first safe refactor.

## Baseline verification

Ran before code changes for this inventory:

```bash
go test ./internal/modules/serial/... ./internal/core/... -count=1
```

Result: PASS across all listed packages on the current Windows/MSYS environment.
