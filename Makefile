# Blockchain Project Makefile

# Variables
BINARY_NODE = blockchain-node
BINARY_CLI = blockchain-cli
CMD_NODE = ./cmd/node
CMD_CLI = ./cmd/cli
BUILD_DIR = build
DOCKER_COMPOSE = docker-compose.yml

# Go parameters
GOCMD = go
GOBUILD = $(GOCMD) build
GOCLEAN = $(GOCMD) clean
GOTEST = $(GOCMD) test
GOMOD = $(GOCMD) mod
GOGET = $(GOCMD) get

# Build flags
LDFLAGS = -ldflags="-w -s"
BUILD_FLAGS = -a -installsuffix cgo

.PHONY: all build clean test deps docker-build docker-up docker-down demo help

# Default target
all: clean deps build

# Build both binaries
build: build-node build-cli

# Build node binary
build-node:
	@echo "Building blockchain node..."
	CGO_ENABLED=1 $(GOBUILD) $(LDFLAGS) -o $(BINARY_NODE) $(CMD_NODE)

# Build CLI binary
build-cli:
	@echo "Building blockchain CLI..."
	CGO_ENABLED=1 $(GOBUILD) $(LDFLAGS) -o $(BINARY_CLI) $(CMD_CLI)

# Build for production (optimized)
build-prod:
	@echo "Building optimized binaries..."
	mkdir -p $(BUILD_DIR)
	CGO_ENABLED=1 $(GOBUILD) $(BUILD_FLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NODE) $(CMD_NODE)
	CGO_ENABLED=1 $(GOBUILD) $(BUILD_FLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_CLI) $(CMD_CLI)

# Clean build artifacts
clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	rm -f $(BINARY_NODE) $(BINARY_CLI)
	rm -rf $(BUILD_DIR)
	rm -rf data/
	rm -rf wallets/

# Run tests
test:
	@echo "Running tests..."
	$(GOTEST) -v ./pkg/...

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

# Verify dependencies
deps-verify:
	@echo "Verifying dependencies..."
	$(GOMOD) verify

# Update dependencies
deps-update:
	@echo "Updating dependencies..."
	$(GOMOD) get -u ./...
	$(GOMOD) tidy

# Build Docker images
docker-build:
	@echo "Building Docker images..."
	docker-compose build

# Start Docker containers
docker-up:
	@echo "Starting blockchain network..."
	docker-compose up -d

# Stop Docker containers
docker-down:
	@echo "Stopping blockchain network..."
	docker-compose down

# View Docker logs
docker-logs:
	@echo "Viewing Docker logs..."
	docker-compose logs -f

# Clean Docker resources
docker-clean:
	@echo "Cleaning Docker resources..."
	docker-compose down -v
	docker system prune -f

# Run the demo
demo: build-cli
	@echo "Running blockchain demo..."
	./demo.sh

# Run a single node locally
run-node: build-node
	@echo "Starting single node..."
	export NODE_ID=node1 && \
	export IS_LEADER=true && \
	export LISTEN_ADDR=:50051 && \
	export DATA_DIR=./data && \
	./$(BINARY_NODE)

# Create Alice and Bob wallets
create-wallets: build-cli
	@echo "Creating demo wallets..."
	./$(BINARY_CLI) create-user alice
	./$(BINARY_CLI) create-user bob

# Send demo transaction
demo-tx: build-cli
	@echo "Sending demo transaction..."
	./$(BINARY_CLI) send alice bob 10.0

# Check blockchain status
status: build-cli
	@echo "Checking blockchain status..."
	./$(BINARY_CLI) status

# List all blocks
list-blocks: build-cli
	@echo "Listing all blocks..."
	./$(BINARY_CLI) list-blocks

# Validate blockchain
validate: build-cli
	@echo "Validating blockchain..."
	./$(BINARY_CLI) validate

# Format Go code
fmt:
	@echo "Formatting code..."
	$(GOCMD) fmt ./...

# Lint code (requires golangci-lint)
lint:
	@echo "Linting code..."
	golangci-lint run

# Generate protocol buffers (requires protoc)
proto:
	@echo "Generating protocol buffers..."
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		proto/blockchain.proto

# Install development tools
install-tools:
	@echo "Installing development tools..."
	$(GOGET) -u github.com/golangci/golangci-lint/cmd/golangci-lint
	$(GOGET) -u google.golang.org/protobuf/cmd/protoc-gen-go
	$(GOGET) -u google.golang.org/grpc/cmd/protoc-gen-go-grpc

# Create project structure
init-project:
	@echo "Initializing project structure..."
	mkdir -p pkg/blockchain pkg/wallet pkg/storage pkg/consensus pkg/p2p
	mkdir -p cmd/node cmd/cli
	mkdir -p proto
	mkdir -p data wallets

# Help
help:
	@echo "Blockchain Project Makefile"
	@echo ""
	@echo "Available targets:"
	@echo "  all           - Clean, download deps, and build"
	@echo "  build         - Build both node and CLI binaries"
	@echo "  build-node    - Build node binary only"
	@echo "  build-cli     - Build CLI binary only"
	@echo "  build-prod    - Build optimized production binaries"
	@echo "  clean         - Clean build artifacts and data"
	@echo "  test          - Run tests"
	@echo "  deps          - Download and tidy dependencies"
	@echo "  deps-verify   - Verify dependencies"
	@echo "  deps-update   - Update dependencies"
	@echo "  docker-build  - Build Docker images"
	@echo "  docker-up     - Start Docker containers"
	@echo "  docker-down   - Stop Docker containers"
	@echo "  docker-logs   - View Docker logs"
	@echo "  docker-clean  - Clean Docker resources"
	@echo "  demo          - Run blockchain demo"
	@echo "  run-node      - Run single node locally"
	@echo "  create-wallets- Create Alice and Bob wallets"
	@echo "  demo-tx       - Send demo transaction"
	@echo "  status        - Check blockchain status"
	@echo "  list-blocks   - List all blocks"
	@echo "  validate      - Validate blockchain"
	@echo "  fmt           - Format Go code"
	@echo "  lint          - Lint code (requires golangci-lint)"
	@echo "  proto         - Generate protocol buffers"
	@echo "  install-tools - Install development tools"
	@echo "  init-project  - Initialize project structure"
	@echo "  help          - Show this help message" 