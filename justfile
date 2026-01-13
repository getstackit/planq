# Default recipe
default: check

# Setup the development environment (install tools, dependencies, and build)
setup: install-tools deps build
	@echo "Setup complete! You can now run 'just check' to verify everything is working."

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	go mod download
	go mod tidy

# Install development tools (gotestsum, golangci-lint, goimports)
install-tools:
	@echo "Installing development tools..."
	go install gotest.tools/gotestsum@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install golang.org/x/tools/cmd/goimports@latest

# Run all tests (with caching for faster repeated runs)
test:
	@echo "Running tests..."
	@if command -v gotestsum >/dev/null 2>&1; then \
		gotestsum --format pkgname-and-test-fails -- ./...; \
	else \
		go test ./...; \
	fi

# Run all tests without caching (for CI or debugging flaky tests)
test-fresh:
	@echo "Running tests (no cache)..."
	@if command -v gotestsum >/dev/null 2>&1; then \
		gotestsum --format pkgname-and-test-fails -- ./... -count=1; \
	else \
		go test ./... -count=1; \
	fi

# Run tests with verbose output
test-verbose:
	@if command -v gotestsum >/dev/null 2>&1; then \
		gotestsum --format standard-verbose -- ./...; \
	else \
		go test -v ./...; \
	fi

# Run tests with coverage
test-coverage:
	@if command -v gotestsum >/dev/null 2>&1; then \
		gotestsum --format pkgname-and-test-fails -- -coverprofile=coverage.out ./...; \
	else \
		go test -coverprofile=coverage.out ./...; \
	fi
	go tool cover -html=coverage.out -o coverage.html

# Run tests with race detection
test-race:
	@if command -v gotestsum >/dev/null 2>&1; then \
		gotestsum --format pkgname-and-test-fails -- -race ./...; \
	else \
		go test -race ./...; \
	fi

# Run tests for a specific package
# Usage: just test-pkg ./internal/tmux
test-pkg pkg:
	@if [ -z "{{pkg}}" ]; then \
		echo "Usage: just test-pkg ./internal/tmux"; \
		exit 1; \
	fi
	@if command -v gotestsum >/dev/null 2>&1; then \
		gotestsum --format standard-verbose -- {{pkg}}; \
	else \
		go test -v {{pkg}}; \
	fi

# Run tests in watch mode (requires gotestsum)
test-watch:
	@if command -v gotestsum >/dev/null 2>&1; then \
		gotestsum --watch --format pkgname-and-test-fails -- ./...; \
	else \
		echo "gotestsum not installed. Install with: go install gotest.tools/gotestsum@latest"; \
		exit 1; \
	fi

# Clean test artifacts
clean:
	rm -f coverage.out coverage.html
	rm -f planq
	go clean

# Format code
fmt:
	@if command -v goimports >/dev/null 2>&1; then \
		goimports -w .; \
	elif [ -f "$(go env GOPATH)/bin/goimports" ]; then \
		"$(go env GOPATH)/bin/goimports" -w .; \
	else \
		go fmt ./...; \
	fi

# Run linter (requires golangci-lint)
lint:
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run --allow-parallel-runners; \
	elif [ -f "$(go env GOPATH)/bin/golangci-lint" ]; then \
		"$(go env GOPATH)/bin/golangci-lint" run --allow-parallel-runners; \
	else \
		echo "golangci-lint not installed. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

# Run linter and fix issues (if supported by the linter)
lint-fix:
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run --allow-parallel-runners --fix; \
	elif [ -f "$(go env GOPATH)/bin/golangci-lint" ]; then \
		"$(go env GOPATH)/bin/golangci-lint" run --allow-parallel-runners --fix; \
	else \
		echo "golangci-lint not installed. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

# Run all checks (format, lint, tests)
check:
	@echo "Formatting..."
	@just fmt
	@echo "Linting..."
	@just lint
	@echo "Testing..."
	@just test

# Build the planq binary with local version info
build:
	#!/bin/bash
	mkdir -p bin
	COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
	DATE=$(date -u +%Y-%m-%dT%H:%M:%SZ)
	go build -ldflags "-X main.version=dev-local -X main.commit=$COMMIT -X main.date=$DATE" -o bin/planq ./cmd/planq

# Install binary to ~/.local/bin (with distinct version)
install:
	#!/bin/bash
	mkdir -p ~/.local/bin
	COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
	DATE=$(date -u +%Y-%m-%dT%H:%M:%SZ)
	go build -ldflags "-X main.version=dev-installed -X main.commit=$COMMIT -X main.date=$DATE" -o ~/.local/bin/planq ./cmd/planq
	echo "Installed to ~/.local/bin"

# Remove installed binary from ~/.local/bin
uninstall:
	rm -f ~/.local/bin/planq
	@echo "Removed from ~/.local/bin"

# Run planq command (builds first, then runs)
# Usage: just run <args>
run *args:
	@echo "Building planq..."
	@just build
	./bin/planq {{args}}
