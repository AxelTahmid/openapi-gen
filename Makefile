.PHONY: all tidy install test build run clean fmt lint vet

all: tidy build

# Run go mod tidy to clean up dependencies
tidy:
	go mod tidy

# Install dependencies (go mod download)
install:
	go mod download

# Run tests
test:
	go test -v ./...

# Build the project (no output binary, just check build)
build:
	go build ./...

# Run gofmt on all Go files
fmt:
	gofmt -s -w .

# Run golint (requires golint to be installed)
lint:
	@golint ./...

# Run go vet for static analysis
vet:
	go vet ./...

# Clean build artifacts
clean:
	rm -rf openapi.json

# Run the project (customize as needed)
run:
	go run .
