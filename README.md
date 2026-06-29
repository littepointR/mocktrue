# PortWeave

[![CI](https://github.com/littepointR/portweave/actions/workflows/ci.yml/badge.svg)](https://github.com/littepointR/portweave/actions/workflows/ci.yml)

PortWeave 是面向嵌入式调试和自动化验证的开源串口拓扑工作台，可用图形化节点连接物理串口、虚拟串口、远端 raw TCP 串口、脚本处理器和协议模拟器。

PortWeave is an open-source serial topology workbench for embedded debugging and automated validation, connecting physical serial ports, virtual ports, remote raw TCP endpoints, scripts, and protocol simulators as graph nodes.

## 5-minute no-hardware quick start

Start with a demo that does not require external serial hardware:

1. Install frontend dependencies: `cd frontend && pnpm install`.
2. Start the Wails dev app: `wails3 dev -config ./build/config.yml -port 9245`.
3. Open Settings -> Global / 全局设置 -> 示例配置.
4. Load `serial-observability-demo` or `remote-serial-demo`.
5. Open the loaded graph tab, start the graph runtime, and inspect node buffers, monitor frames, and status.

Detailed walkthrough: [Getting Started](docs/getting-started.md). Full demo list: [Examples](docs/examples.md). Documentation map: [docs/index.md](docs/index.md).

## Why PortWeave?

- Build serial topologies visually: model endpoints, filters, scripts, protocol nodes, and monitor branches as a graph.
- Simulate devices without hardware: use generated virtual ports, script generators, Modbus/FECbus nodes, and localhost raw TCP loopback demos.
- Automate validation through MCP: inspect catalogs, validate/start graphs, send/query data, collect evidence, and stop resources repeatably.

PortWeave is not a generic dashboard widget clone. It focuses on topology-native debugging, protocol simulation, runtime evidence, and agent/CI-friendly automation.

## Features

- Serial graph topology: current nodes include `serial.physical`, `serial.virtual`, `serial.remote`, `serial.bridge`, `serial.monitor`, `serial.filter`, script nodes, Modbus nodes, and FECbus nodes. See [Serial Graph Node Catalog](docs/serial-graph-node-catalog.md).
- No-hardware demo workspaces: generated examples use graph-owned virtual endpoints, script generators, filters, protocol nodes, and raw TCP loopback paths. See [Examples](docs/examples.md).
- Runtime observability: node status, RX/TX counters, endpoint buffers, monitor frames, protocol frames, and script analyzer fields. See [Serial Graph Runtime Flow](docs/serial-graph-runtime-flow.md).
- Remote raw TCP serial: `serial.remote` supports raw TCP client/server roles for trusted LAN/VPN/SSH-tunnel scenarios. Raw TCP has no auth, encryption, or serial negotiation. See [Remote Serial Graph Node](docs/remote-serial-node.md).
- Modbus support: Modbus RTU/ASCII master/slave graph nodes plus existing session tools for register and Unit ID workflows.
- FECbus support: FECbus master/slave graph nodes, function catalog, frame querying, and device-state simulation workflows.
- Script nodes: safe script generation, transformation, and analysis with timeout/output/state limits.
- MCP automation: 64 current tools for serial resources, serial graph runtime, monitors, Modbus, and FECbus. See [MCP API and Recipes](docs/mcp-api.md) and [AI Automation Guide](docs/ai-automation.md).
- Workspace snapshots: whole-workspace and graph-tab snapshot formats for repeatable demos and saved graphs. See [Workspace and Graph Schema](docs/workspace-graph-schema.md).

## Documentation

Start here:

- [Documentation index](docs/index.md)
- [Getting Started](docs/getting-started.md)
- [Workflows](docs/workflows.md)
- [Use Cases](docs/use-cases.md)
- [Examples](docs/examples.md)
- [Troubleshooting](docs/troubleshooting.md)

Developer/architecture docs:

- [Architecture](docs/architecture.md)
- [Architecture Decisions](docs/development/architecture-decisions.md)
- [Testing](docs/development/testing.md)
- [Windows Serial Testing](docs/development/windows-serial-testing.md)

## Tech stack

- Desktop shell: Wails v3 alpha.
- Backend: Go services/modules.
- Frontend: Vue 3, TypeScript, Vite, Naive UI, Pinia, Monaco Editor.
- Automation: Go MCP SDK, local MCP server.
- Tests: Go test, Vitest, Playwright.
- CI: GitHub Actions with frontend, coverage, backend matrix, and backend aggregate jobs.

The current architecture intentionally keeps Go + Wails + Vue/TypeScript, defers gRPC until evidence requires it, and does not plan a Qt/C++ rewrite. See [Architecture Decisions](docs/development/architecture-decisions.md).

## Requirements

- Go `1.26.0`, from `go.mod`.
- Node.js `22`, matching CI.
- pnpm `10.32.1`, matching CI.
- Wails CLI v3 alpha.

Install Wails CLI:

```bash
go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-alpha2.103
```

Install frontend dependencies:

```bash
cd frontend
pnpm install
```

## Development

Start the desktop development app from the repository root:

```bash
wails3 dev -config ./build/config.yml -port 9245
```

Taskfile alternatives:

```bash
wails3 task dev
wails3 task build
wails3 task package
wails3 task run:server
```

## Tests and verification

Frontend:

```bash
cd frontend
pnpm test -- --run
pnpm run test:coverage
pnpm exec vue-tsc --noEmit
pnpm run build:dev
pnpm run e2e:smoke
```

Backend:

```bash
go test ./... -count=1
go test ./internal/core/... ./internal/modules/serial/... -count=1
go test -tags integration -timeout 120s ./tests/go/integration ./tests/automation/integration -count=1
```

Coverage and Makefile:

```bash
./scripts/test-check-go-coverage.sh
./scripts/check-go-coverage.sh
make test
make coverage
make lint
make build
```

Frontend and Go coverage gates default to 90% in CI. `make lint` requires `golangci-lint` to be installed. More detail: [Testing](docs/development/testing.md).

## Project structure

```text
.
├── build/                 # Wails build configuration and platform tasks
├── docs/                  # User, architecture, automation, and development docs
├── frontend/              # Vue/TypeScript frontend
├── internal/modules/      # Backend modules: serial, Modbus, FECbus, MCP
├── main.go                # Application entry point
├── Makefile               # Backend/build/coverage targets
├── Taskfile.yml           # Wails development, build, package, and server tasks
└── README.md
```

## Contributing

Please read [CONTRIBUTING.md](CONTRIBUTING.md) before submitting changes. Run the tests relevant to your change; for frontend/backend public behavior, also run typecheck/build checks.

Security issues should be reported through [SECURITY.md](SECURITY.md). Do not disclose sensitive details in public issues.

## License

PortWeave is licensed under the [MIT License](LICENSE).
