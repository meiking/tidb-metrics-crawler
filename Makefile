# Define project variables
PROJECT_NAME := tidb-metrics-crawler
MODULE := github.com/meiking/tidb-metrics-crawler
BINARY := $(PROJECT_NAME)
CMD_DIR := ./cmd
ETC_DIR := ./etc
OUTPUT_DIR := ./bin
CONFIG_FILE := $(ETC_DIR)/config.yaml

# Go related variables
GO := go
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)
GOFLAGS := -mod=mod

# Build variables
LDFLAGS := -s -w
BUILD_FLAGS := -ldflags "$(LDFLAGS)"

# Default target
.DEFAULT_GOAL := build

# Build the application
build:
	@mkdir -p $(OUTPUT_DIR)
	$(GO) build $(GOFLAGS) $(BUILD_FLAGS) -o $(OUTPUT_DIR)/$(BINARY)-$(GOOS)-$(GOARCH) $(CMD_DIR)
	@ln -sf $(BINARY)-$(GOOS)-$(GOARCH) $(OUTPUT_DIR)/$(BINARY)
	@echo "Built: $(OUTPUT_DIR)/$(BINARY)"

# Build for all major platforms
build-all:
	@mkdir -p $(OUTPUT_DIR)
	GOOS=linux GOARCH=amd64 $(GO) build $(GOFLAGS) $(BUILD_FLAGS) -o $(OUTPUT_DIR)/$(BINARY)-linux-amd64 $(CMD_DIR)
	GOOS=darwin GOARCH=amd64 $(GO) build $(GOFLAGS) $(BUILD_FLAGS) -o $(OUTPUT_DIR)/$(BINARY)-darwin-amd64 $(CMD_DIR)
	GOOS=windows GOARCH=amd64 $(GO) build $(GOFLAGS) $(BUILD_FLAGS) -o $(OUTPUT_DIR)/$(BINARY)-windows-amd64.exe $(CMD_DIR)
	@echo "Built all platform binaries in: $(OUTPUT_DIR)"

# Run the application
run: build
	$(OUTPUT_DIR)/$(BINARY) -config $(CONFIG_FILE)

# Run with race detection
run-race:
	$(GO) run -race $(GOFLAGS) $(CMD_DIR) -config $(CONFIG_FILE)

# Run tests
test:
	$(GO) test $(GOFLAGS) -v ./pkg/...
	@echo "Tests completed"

# Run tests with coverage
test-coverage:
	$(GO) test $(GOFLAGS) -v -coverprofile=coverage.out ./pkg/...
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Clean build artifacts
clean:
	$(GO) clean $(GOFLAGS)
	rm -rf $(OUTPUT_DIR)
	rm -f coverage.out coverage.html
	@echo "Cleaned build artifacts and coverage reports"

# Format code
fmt:
	$(GO) fmt ./...
	@echo "Code formatted"

# Check for linting issues
lint:
	@if ! command -v golint &> /dev/null; then \
		echo "Installing golint..."; \
		$(GO) install golang.org/x/lint/golint@latest; \
	fi
	golint ./...
	@echo "Linting completed"

# Run go mod tidy
tidy:
	$(GO) mod tidy
	@echo "Dependencies tidied"

# Create example config if missing
config-example:
	@if [ ! -f $(CONFIG_FILE) ]; then \
		cp $(ETC_DIR)/config.yaml.example $(CONFIG_FILE); \
		echo "Created example config: $(CONFIG_FILE)"; \
	else \
		echo "Config file already exists: $(CONFIG_FILE)"; \
	fi

# Show help
help:
	@echo "Available targets:"
	@echo "  build           - Build the application for current platform"
	@echo "  build-all       - Build for Linux, Darwin, and Windows"
	@echo "  run             - Build and run the application"
	@echo "  run-race        - Run with race detection"
	@echo "  test            - Run all tests"
	@echo "  test-coverage   - Run tests with coverage report"
	@echo "  clean           - Clean build artifacts"
	@echo "  fmt             - Format code with gofmt"
	@echo "  lint            - Check code with golint"
	@echo "  tidy            - Run go mod tidy"
	@echo "  config-example  - Create example config file if missing"
	@echo "  help            - Show this help message"
    