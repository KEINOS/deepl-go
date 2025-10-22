.PHONY: help test test-unit test-e2e test-all mock-up mock-down clean fmt vet

# Default target
help:
	@echo "Available targets:"
	@echo "  test           - Run unit tests only"
	@echo "  test-unit      - Same as test"
	@echo "  test-e2e       - Run E2E tests with mock server"
	@echo "  test-all       - Run both unit and E2E tests"
	@echo "  mock-up        - Start DeepL mock server"
	@echo "  mock-down      - Stop and clean up Docker containers"
	@echo "  clean          - Clean up test artifacts and Docker containers"
	@echo "  fmt            - Run go fmt on all files"
	@echo "  vet            - Run go vet on all files"

# Run unit tests only (no E2E tests)
test: test-unit

test-unit:
	go test -v -count=1 -race -shuffle=on -coverprofile=coverage.txt ./...

# Run E2E tests with mock server
test-e2e:
	@echo "Starting DeepL mock server..."
	docker compose up deepl-mock -d
	@echo "Running E2E tests..."
	docker compose run --rm deepl-test
	@echo "Cleaning up..."
	docker compose down --remove-orphans

# Run all tests (unit + E2E)
test-all: test-unit test-e2e

# Start mock server
mock-up:
	docker compose up deepl-mock -d

# Stop mock server
mock-down:
	docker compose down --remove-orphans

# Clean up
clean: mock-down
	rm -f coverage.txt

# Format code
fmt:
	gofmt -s -w .

# Vet code
vet:
	go vet ./...
