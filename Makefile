.PHONY: build build-all build-darwin build-linux build-windows test test-race vet lint coverage coverage-gate-test clean bindings frontend frontend-test frontend-coverage frontend-typecheck frontend-build frontend-e2e-smoke ci ci-strict full

APP_NAME := portweave
BUILD_DIR := bin
GO_COVERAGE_PROFILE ?= coverage.out
GO_COVERAGE_HTML ?= coverage.html

# Default: build for current platform
build:
	go build -o $(BUILD_DIR)/$(APP_NAME) .

# Cross-platform builds
build-all: build-darwin build-linux build-windows

build-darwin:
	GOOS=darwin GOARCH=arm64 go build -o $(BUILD_DIR)/$(APP_NAME)-darwin-arm64 .
	GOOS=darwin GOARCH=amd64 go build -o $(BUILD_DIR)/$(APP_NAME)-darwin-amd64 .

build-linux:
	GOOS=linux GOARCH=amd64 go build -o $(BUILD_DIR)/$(APP_NAME)-linux-amd64 .
	GOOS=linux GOARCH=arm64 go build -o $(BUILD_DIR)/$(APP_NAME)-linux-arm64 .

build-windows:
	GOOS=windows GOARCH=amd64 go build -o $(BUILD_DIR)/$(APP_NAME)-windows-amd64.exe .

# Run all default tests, including real socat-backed integration tests.
test:
	go test ./...
	go test -tags integration -timeout 120s ./tests/go/integration ./tests/automation/integration

# Run tests with race detector
test-race:
	go test -race ./...

# Run go vet
vet:
	go vet ./...

# Run linter (requires golangci-lint)
lint:
	golangci-lint run ./...

# Run the coverage gate regression checks.
coverage-gate-test:
	./scripts/test-check-go-coverage.sh

# Run backend/internal tests with coverage and enforce the 90% coverage floor.
coverage: coverage-gate-test
	GO_COVERAGE_PROFILE="$(GO_COVERAGE_PROFILE)" ./scripts/check-go-coverage.sh
	go tool cover -html="$(GO_COVERAGE_PROFILE)" -o "$(GO_COVERAGE_HTML)"

# Run a local CI-like command set without requiring golangci-lint.
ci: vet test coverage frontend-coverage frontend-typecheck frontend-build

# Run the CI-like set plus optional strict checks that may require local tools/browsers.
ci-strict: ci lint frontend-e2e-smoke

# Clean build artifacts
clean:
	rm -rf $(BUILD_DIR) coverage.out coverage.html

# Generate Wails bindings
bindings:
	wails3 task darwin:common:generate:bindings

# Build frontend
frontend:
	cd frontend && pnpm run build

# Run frontend unit tests without coverage.
frontend-test:
	cd frontend && pnpm test -- --run

# Run frontend unit tests with the configured coverage gate.
frontend-coverage:
	cd frontend && pnpm run test:coverage

# Run frontend TypeScript/Vue type checks.
frontend-typecheck:
	cd frontend && pnpm exec vue-tsc --noEmit

# Run the CI frontend development build.
frontend-build:
	cd frontend && pnpm run build:dev

# Run the GUI smoke test used by CI after Playwright browser install.
frontend-e2e-smoke:
	cd frontend && pnpm run e2e:smoke

# Full build (frontend + Go)
full: frontend build
