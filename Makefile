.PHONY: build build-all test test-race vet lint clean

APP_NAME := portweave
BUILD_DIR := bin

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

# Run backend/internal tests with coverage and enforce the current ratchet threshold.
coverage:
	./scripts/check-go-coverage.sh
	go tool cover -html=coverage.out -o coverage.html

# Clean build artifacts
clean:
	rm -rf $(BUILD_DIR) coverage.out coverage.html

# Generate Wails bindings
bindings:
	wails3 task darwin:common:generate:bindings

# Build frontend
frontend:
	cd frontend && pnpm run build

# Full build (frontend + Go)
full: frontend build
