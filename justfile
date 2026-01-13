# Planq justfile

# Default recipe
default: check

# Build the binary
build:
    go build -o planq ./cmd/planq

# Run the exploration/demo script
run:
    go run ./cmd/planq

# Run tests
test:
    go test ./...

# Run tests with verbose output
test-v:
    go test -v ./...

# Run linter
lint:
    golangci-lint run

# Format code
fmt:
    go fmt ./...
    goimports -w .

# Run fmt, lint, and tests
check: fmt lint test

# Install dependencies
deps:
    go mod tidy
    go mod download

# Clean build artifacts
clean:
    rm -f planq
    go clean

# Install the binary locally
install:
    go install ./cmd/planq
