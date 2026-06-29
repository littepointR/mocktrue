# Testing

This page maps the current PortWeave test, coverage, build, and CI commands, including local Makefile targets that mirror the GitHub Actions gate shape.

## Prerequisites

- Go `1.26.0` from `go.mod`.
- Node.js `22`, matching GitHub Actions.
- pnpm `10.32.1`, matching GitHub Actions.
- Wails v3 alpha CLI for development/package tasks.
- Linux CI backend dependencies for Wails/WebKit/socat: `libgtk-4-dev`, `libwebkitgtk-6.0-dev`, `pkg-config`, `socat`.
- Frontend dependencies installed with `cd frontend && pnpm install` or CI's `pnpm install --frozen-lockfile`.

## Test taxonomy

### Go package tests

Run all Go packages:

```bash
go test ./... -count=1
```

Run current CI backend subset:

```bash
go test ./internal/core/... ./internal/modules/serial/... -count=1
```

### Go integration tests

Opt-in integration packages use build tags:

```bash
go test -tags integration -timeout 120s ./tests/go/integration ./tests/automation/integration -count=1
```

POSIX virtual serial tests use `socat` and are excluded from Windows by build tags. Windows COM integration is optional and documented in [Windows Serial Testing](windows-serial-testing.md).

### Frontend Vitest

Run all frontend unit/component/store tests:

```bash
cd frontend
pnpm test -- --run
```

Run a targeted test file:

```bash
cd frontend
pnpm exec vitest run src/workspace/demoWorkspaces.test.ts
```

### Frontend coverage

Run the Vitest coverage gate:

```bash
cd frontend
pnpm run test:coverage
```

Current `vite.config.ts` enforces 90% defaults for statements, branches, functions, and lines. Local experiments may override with `VITEST_COVERAGE_STATEMENTS`, `VITEST_COVERAGE_BRANCHES`, `VITEST_COVERAGE_FUNCTIONS`, and `VITEST_COVERAGE_LINES`; CI uses the 90% defaults.

### Typecheck and build

```bash
cd frontend
pnpm exec vue-tsc --noEmit
pnpm run build:dev
```

`build:dev` runs `vue-tsc` and a Vite development-mode build without minification.

### Playwright smoke

```bash
cd frontend
pnpm run e2e:smoke
```

CI installs Playwright Chromium before this step.

### Backend coverage gate

Run the gate regression checks:

```bash
./scripts/test-check-go-coverage.sh
```

Run backend/internal coverage with the 90% floor:

```bash
./scripts/check-go-coverage.sh
```

The script runs:

```bash
go test -coverpkg=./internal/... -coverprofile=coverage.out ./internal/...
go tool cover -func=coverage.out
```

It enforces the threshold using exact statement counts from the cover profile rather than rounded text output. Override with `GO_COVERAGE_THRESHOLD` only for local experiments.

## Makefile commands

Current top-level targets:

```bash
make ci
make ci-strict
make test
make coverage
make lint
make build
make frontend-test
make frontend-coverage
make frontend-typecheck
make frontend-build
make frontend-e2e-smoke
make build-all
make frontend
make full
make clean
```

Important details:

- `make test` runs `go test ./...` and integration packages with `-tags integration`.
- `make coverage` runs coverage gate regression tests, backend/internal coverage, threshold enforcement, and HTML coverage generation.
- `make ci` runs vet, Go tests/integration, Go coverage, frontend coverage, frontend typecheck, and frontend development build without requiring `golangci-lint`.
- `make ci-strict` extends `make ci` with `make lint` and the Playwright smoke target.
- `make lint` requires `golangci-lint` to be installed.
- `make build` runs `go build -o bin/portweave .`.
- `make frontend` runs `cd frontend && pnpm run build`.
- `make frontend-test` runs `cd frontend && pnpm test -- --run`.
- `make frontend-coverage` runs `cd frontend && pnpm run test:coverage` and preserves the current 90% coverage gate.
- `make frontend-typecheck` runs `cd frontend && pnpm exec vue-tsc --noEmit`.
- `make frontend-build` runs `cd frontend && pnpm run build:dev`.
- `make frontend-e2e-smoke` runs `cd frontend && pnpm run e2e:smoke`; install Playwright browsers first when needed.
- `make full` runs frontend build then Go build.

## GitHub Actions CI map

Current `.github/workflows/ci.yml` jobs:

### `lint`

- Checkout.
- Set up Go from `go.mod`.
- Set up pnpm `10.32.1`.
- Set up Node `22` with pnpm cache.
- Install Linux backend dependencies (`libgtk-4-dev`, `libwebkitgtk-6.0-dev`, `pkg-config`, `socat`).
- `cd frontend && pnpm install --frozen-lockfile`.
- `cd frontend && pnpm run build:dev` to generate `frontend/dist` for the Go embed prerequisite before `go vet ./...` visits packages that include `//go:embed all:frontend/dist`.
- `go mod download`.
- Check `gofmt` for tracked Go files.
- `go vet ./...`.
- Check trailing whitespace in tracked Go, frontend, JSON/YAML, and Markdown source files, excluding generated lockfile noise.

### `frontend`

- Checkout.
- Set up pnpm `10.32.1`.
- Set up Node `22` with pnpm cache.
- `pnpm install --frozen-lockfile`.
- `pnpm run test:coverage`.
- Upload `frontend/coverage` when present.
- `pnpm exec vue-tsc --noEmit`.
- `pnpm run build:dev`.
- Install Playwright Chromium.
- `pnpm run e2e:smoke`.
- Upload Playwright reports/results when present.

### `coverage`

- Checkout.
- Set up Go from `go.mod`.
- Install Linux backend dependencies.
- `go mod download`.
- `./scripts/test-check-go-coverage.sh`.
- `./scripts/check-go-coverage.sh`.
- Upload `coverage.out` when present.

### `backend-matrix`

Runs on Ubuntu, macOS, and Windows:

- Checkout.
- Set up Go from `go.mod`.
- Linux dependency install on Linux only.
- `go mod download`.
- `go test ./internal/core/... ./internal/modules/serial/... -count=1`.
- `go test -tags integration -timeout 120s ./tests/go/integration ./tests/automation/integration -count=1`.

### `backend`

Aggregates `backend-matrix` and fails if any matrix entry failed.

## Wails binding-sensitive changes

Regenerate/check bindings only when exported Go service or model signatures change. When bindings change:

1. Regenerate through the project Wails task.
2. Inspect generated diffs for unrelated churn.
3. Run targeted Go tests for changed services.
4. Run frontend typecheck/build.
5. Do not stage unrelated binding output.

## Documentation-only changes

For docs-only changes, the minimum relevant check is usually:

```bash
git diff --check -- README.md docs
```

If docs mention source facts such as node types, demo IDs, MCP tools, or CI commands, re-check the source files before completion.

## Troubleshooting CI failures

- Frontend coverage failure: compare actual coverage output against the 90% thresholds.
- Backend coverage failure: inspect exact statement-count output from `scripts/check-go-coverage.sh`.
- Backend matrix failure on Windows: check whether integration tests skipped or failed due to COM setup; optional COM tests are documented separately.
- Build failure involving Wails embed: ensure frontend `dist/` exists when building packages that embed frontend assets.
- Lint failure locally: confirm `golangci-lint` is installed before treating `make lint` as a product failure.

## Related docs

- [Windows Serial Testing](windows-serial-testing.md)
- [Architecture Decisions](architecture-decisions.md)
- [Remote Serial Graph Node](../remote-serial-node.md)
