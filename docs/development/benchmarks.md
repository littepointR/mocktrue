# Benchmarks

This page defines the benchmark taxonomy and thresholds PortWeave should use before broad performance work or architecture changes.

## Benchmark principles

- Benchmark Go hotpaths before proposing stack-level changes.
- Keep benchmark fixtures deterministic and no-hardware by default.
- Report throughput, allocations, and sustained-runtime behavior together.
- Prefer small targeted benchmarks over dashboard-style aggregate numbers.
- Do not use benchmark results from debug builds or contaminated developer machines as release claims.

## Categories

### Serial graph routing and fan-out

Targets:

- Single edge byte routing.
- Multi-edge fan-out from one output port.
- Backpressure/counter behavior when downstream nodes differ in cost.
- Stop/cleanup cost for graph-owned resources.

Suggested package area:

```bash
go test ./internal/modules/serial -run '^$' -bench 'SerialGraph|GraphRouting|FanOut' -benchmem -count=5
```

### Buffer append/query

Targets:

- Ring append throughput.
- Query by offset/length.
- Snapshot allocation behavior.
- Large capture truncation behavior.

Suggested package area:

```bash
go test ./internal/modules/serial/... -run '^$' -bench 'Buffer|Query' -benchmem -count=5
```

### Script runtime

Targets:

- Generator output cost.
- Transform pass-through cost.
- Analyzer field/error extraction cost.
- State helper cost and state-size guardrails.
- Timeout behavior for hostile scripts.

Suggested package area:

```bash
go test ./internal/modules/serial -run '^$' -bench 'Script' -benchmem -count=5
```

### Protocol parsers

Targets:

- Visual parser header search and length/checksum paths.
- Modbus RTU parser/encoder paths.
- FECbus parser/encoder paths.
- Script parser templates once they have executable parser coverage.

Suggested package area:

```bash
go test ./internal/modules/serial/protocol/... ./internal/modules/serial/modbus/... ./internal/modules/serial/fecbus/... -run '^$' -bench . -benchmem -count=5
```

### Remote raw TCP

Targets:

- Localhost server/client connect time.
- Reconnect loop behavior.
- Read/write throughput with bounded buffers.
- Cleanup latency.

These should stay opt-in until the fixture can avoid port collisions and prove teardown.

## Initial thresholds

Use these as smoke thresholds, not product marketing claims:

| Category | Initial gate | Notes |
| --- | --- | --- |
| Allocations | No unbounded allocations per routed frame in steady state | Investigate regressions before release. |
| Graph routing | No order-of-magnitude regression vs previous baseline | Store baseline command/output in PR or release notes. |
| Script runtime | Timeout and output/state limits remain enforced | Security/correctness beats raw throughput. |
| Buffer query | Query latency remains bounded by requested length, not total capture size | Large captures should not require full-buffer scans for simple pages. |
| Remote TCP | No goroutine/resource leaks after stop | Verify with targeted tests before using in release smoke. |

A future CI benchmark gate should compare against checked-in or artifact baselines. Until then, benchmark output is advisory evidence for review, not an automatic pass/fail gate.

## Reporting format

When a PR changes hotpaths, include:

- Exact command.
- OS/arch and Go version.
- Benchmark output before and after when available.
- Any changed thresholds or skipped categories.
- Whether generated bindings or frontend code were unaffected.

## Architecture guardrail

Do not use unmeasured performance concerns to justify a Qt/C++ rewrite, gRPC service layer, or dashboard-first redesign. Collect benchmark evidence on current Go hotpaths first, then optimize the narrow bottleneck.

## Related docs

- [Architecture Decisions](architecture-decisions.md)
- [Testing](testing.md)
- [Release](release.md)
