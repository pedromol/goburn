# Makefile for goburn - Dynamic Kubernetes Resource Utilization Tool

.PHONY: help build test test-unit test-integration test-coverage test-race test-bench clean deps lint fmt vet security deploy

# Default target
.DEFAULT_GOAL := help

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=gofmt
GOVET=$(GOCMD) vet

# Binary name
BINARY_NAME=goburn
BINARY_UNIX=$(BINARY_NAME)_unix

# Docker parameters
DOCKER_IMAGE=pedromol/goburn
DOCKER_TAG=latest

# Test parameters
TEST_TIMEOUT=300s
COVERAGE_THRESHOLD=80

help: ## Show this help message
	@echo "🔥 goburn - Dynamic Kubernetes Resource Utilization Tool"
	@echo ""
	@echo "Available targets:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)
	@echo ""
	@echo "Environment Variables:"
	@echo "  TEST_TIMEOUT        Test timeout (default: 300s)"
	@echo "  COVERAGE_THRESHOLD  Coverage threshold % (default: 80)"
	@echo "  VERBOSE            Verbose test output (default: false)"

deps: ## Download and tidy dependencies
	@echo "📦 Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy
	@echo "✅ Dependencies updated"

fmt: ## Format Go code
	@echo "🎨 Formatting Go code..."
	$(GOFMT) -w .
	@echo "✅ Code formatted"

vet: ## Run go vet
	@echo "🔍 Running go vet..."
	$(GOVET) ./...
	@echo "✅ go vet passed"

lint: fmt vet ## Run formatting and vetting

security: ## Run security checks (requires gosec)
	@echo "🔒 Running security checks..."
	@if command -v gosec >/dev/null 2>&1; then \
		gosec ./...; \
	else \
		echo "⚠️  gosec not installed. Install with: go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest"; \
	fi

build: deps lint ## Build the binary
	@echo "🔨 Building $(BINARY_NAME)..."
	$(GOBUILD) -o $(BINARY_NAME) -v ./...
	@echo "✅ Build complete: $(BINARY_NAME)"

build-linux: deps lint ## Build for Linux
	@echo "🐧 Building for Linux..."
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BINARY_UNIX) -v ./...
	@echo "✅ Linux build complete: $(BINARY_UNIX)"

test: ## Run all tests
	@echo "🧪 Running all tests..."
	./run_tests.sh all

test-unit: ## Run unit tests only
	@echo "🧪 Running unit tests..."
	./run_tests.sh unit

test-integration: ## Run integration tests only
	@echo "🧪 Running integration tests..."
	./run_tests.sh integration

test-coverage: ## Run tests with coverage analysis
	@echo "📊 Running tests with coverage..."
	./run_tests.sh coverage

test-race: ## Run race condition tests
	@echo "🏃 Running race condition tests..."
	./run_tests.sh race

test-bench: ## Run benchmark tests
	@echo "⚡ Running benchmark tests..."
	./run_tests.sh benchmark

test-stress: ## Run stress tests
	@echo "🔥 Running stress tests..."
	./run_tests.sh stress

test-arch: ## Test architecture-specific scenarios
	@echo "🏗️  Testing architecture-specific scenarios..."
	./run_tests.sh arch

test-quick: deps ## Run quick tests (unit + integration)
	@echo "⚡ Running quick tests..."
	$(GOTEST) -timeout $(TEST_TIMEOUT) -short ./...

docker-build: ## Build Docker image
	@echo "🐳 Building Docker image..."
	docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) .
	@echo "✅ Docker image built: $(DOCKER_IMAGE):$(DOCKER_TAG)"

docker-build-multiarch: ## Build multi-architecture Docker images
	@echo "🐳 Building multi-architecture Docker images..."
	docker buildx build --platform linux/amd64,linux/arm64 -t $(DOCKER_IMAGE):$(DOCKER_TAG) --push .
	@echo "✅ Multi-arch Docker images built and pushed"

docker-run: docker-build ## Run Docker container locally
	@echo "🐳 Running Docker container..."
	docker run --rm -it \
		-e TARGET_CPU_UTILIZATION=80 \
		-e TARGET_MEMORY_UTILIZATION=80 \
		-e MIN_CPU_UTILIZATION=20 \
		-e MIN_MEMORY_UTILIZATION=20 \
		-e MIN_NETWORK_UTILIZATION_MBPS=20 \
		-e NODE_NAME=docker-test \
		$(DOCKER_IMAGE):$(DOCKER_TAG)

deploy-k8s: ## Deploy to Kubernetes
	@echo "☸️  Deploying to Kubernetes..."
	./deploy.sh both
	@echo "✅ Deployed to Kubernetes"

deploy-amd64: ## Deploy to AMD64 nodes only
	@echo "🖥️  Deploying to AMD64 nodes..."
	./deploy.sh amd64

deploy-arm64: ## Deploy to ARM64 nodes only
	@echo "💪 Deploying to ARM64 nodes..."
	./deploy.sh arm64

deploy-status: ## Check deployment status
	@echo "📊 Checking deployment status..."
	./deploy.sh status

deploy-cleanup: ## Clean up Kubernetes deployment
	@echo "🧹 Cleaning up deployment..."
	./deploy.sh cleanup

verify: ## Verify requirements on current node
	@echo "🔍 Verifying requirements..."
	./examples/verify-requirements.sh

clean: ## Clean build artifacts and test outputs
	@echo "🧹 Cleaning up..."
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_UNIX)
	rm -rf coverage/
	@echo "✅ Cleanup complete"

ci: deps lint test-coverage ## Run CI pipeline (deps, lint, coverage tests)
	@echo "🚀 CI pipeline completed successfully"

ci-full: deps lint test-coverage test-race security ## Run full CI pipeline with security checks
	@echo "🚀 Full CI pipeline completed successfully"

github-actions-test: ## Test the same steps as GitHub Actions locally
	@echo "🧪 Running GitHub Actions equivalent tests locally..."
	@echo "📦 Installing dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy
	$(GOMOD) verify
	@echo "🎨 Checking code formatting..."
	@if [ "$$(gofmt -l . | wc -l)" -gt 0 ]; then \
		echo "❌ Code is not formatted. Run 'make fmt'"; \
		gofmt -l .; \
		exit 1; \
	fi
	@echo "🔍 Running go vet..."
	$(GOVET) ./...
	@echo "🧪 Running unit tests..."
	$(GOTEST) -timeout 300s -run "^Test[^I]" -v ./...
	@echo "🧪 Running integration tests..."
	$(GOTEST) -timeout 300s -run "^TestI" -v ./...
	@echo "📊 Running coverage tests..."
	mkdir -p coverage
	$(GOTEST) -timeout 300s -coverprofile=coverage/coverage.out -covermode=atomic ./...
	go tool cover -html=coverage/coverage.out -o coverage/coverage.html
	@echo "🏃 Running race condition tests..."
	$(GOTEST) -race -timeout 300s ./...
	@echo "⚡ Running benchmark tests..."
	$(GOTEST) -bench=. -benchmem -count=3 ./...
	@echo "✅ All GitHub Actions equivalent tests passed!"

release: clean deps lint test docker-build ## Prepare release (clean, test, build)
	@echo "🎉 Release preparation completed"

dev-setup: ## Set up development environment
	@echo "🛠️  Setting up development environment..."
	$(GOGET) -u golang.org/x/tools/cmd/goimports
	$(GOGET) -u github.com/securecodewarrior/gosec/v2/cmd/gosec
	@if ! command -v kubectl >/dev/null 2>&1; then \
		echo "⚠️  kubectl not found. Please install kubectl for Kubernetes testing."; \
	fi
	@if ! command -v docker >/dev/null 2>&1; then \
		echo "⚠️  docker not found. Please install Docker for container testing."; \
	fi
	chmod +x run_tests.sh
	chmod +x deploy.sh
	chmod +x build.sh
	chmod +x examples/verify-requirements.sh
	@echo "✅ Development environment setup complete"

# Example targets for different environments
test-prod: ## Run production-ready tests
	@echo "🏭 Running production tests..."
	TEST_TIMEOUT=600s COVERAGE_THRESHOLD=85 ./run_tests.sh all

test-dev: ## Run development tests (faster, less strict)
	@echo "🛠️  Running development tests..."
	TEST_TIMEOUT=120s COVERAGE_THRESHOLD=70 ./run_tests.sh unit

# Performance testing
perf-test: ## Run performance tests
	@echo "⚡ Running performance tests..."
	$(GOTEST) -bench=. -benchmem -count=5 ./...

memory-profile: ## Generate memory profile
	@echo "🧠 Generating memory profile..."
	$(GOTEST) -memprofile=coverage/mem.prof -bench=. ./...
	@echo "📊 Memory profile saved to coverage/mem.prof"
	@echo "💡 View with: go tool pprof coverage/mem.prof"

cpu-profile: ## Generate CPU profile
	@echo "🖥️  Generating CPU profile..."
	$(GOTEST) -cpuprofile=coverage/cpu.prof -bench=. ./...
	@echo "📊 CPU profile saved to coverage/cpu.prof"
	@echo "💡 View with: go tool pprof coverage/cpu.prof"

# Documentation
docs: ## Generate documentation
	@echo "📚 Generating documentation..."
	$(GOCMD) doc -all . > docs/api.md
	@echo "✅ Documentation generated in docs/"

# Git hooks
install-hooks: ## Install git pre-commit hooks
	@echo "🪝 Installing git hooks..."
	@echo '#!/bin/bash\nmake lint test-quick' > .git/hooks/pre-commit
	@chmod +x .git/hooks/pre-commit
	@echo "✅ Git pre-commit hook installed"
