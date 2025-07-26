# PriceGuard API - Consolidated Makefile
# Combines development, testing, performance, and deployment commands

.PHONY: help clean build run test dev install-tools

# Variables
BINARY_NAME := priceguard-api
GO_VERSION := 1.21
DOCKER_COMPOSE_FILE := docker-compose.yml
DATABASE_URL := postgres://postgres:password@localhost:5432/priceguard?sslmode=disable
MIGRATE_PATH := ./db/migrations

# Directories
COVERAGE_DIR := coverage
PROFILE_DIR := profiles
REPORT_DIR := reports/performance
DOCS_DIR := docs

# Test Configuration
BENCHMARK_TIME := 30s
BENCHMARK_COUNT := 3
MIN_COVERAGE := 80

# Colors for output
CYAN := \033[36m
GREEN := \033[32m
YELLOW := \033[33m
RED := \033[31m
BLUE := \033[34m
RESET := \033[0m

# Default target
.DEFAULT_GOAL := help

## =============================================================================
## HELP & INFORMATION
## =============================================================================

help: ## Show this help message
	@echo "$(CYAN)🛠️  PriceGuard API - Development Commands$(RESET)"
	@echo ""
	@echo "$(BLUE)📚 Main Commands:$(RESET)"
	@awk '/^## ===.*MAIN/,/^## ===.*[^MAIN]/ { if(/^[a-zA-Z_-]+:.*##/) printf "  $(GREEN)%-20s$(RESET) %s\n", $$1, substr($$0, index($$0, "##") + 3) }' $(MAKEFILE_LIST)
	@echo ""
	@echo "$(BLUE)🧪 Testing Commands:$(RESET)"
	@awk '/^## ===.*TEST/,/^## ===.*[^TEST]/ { if(/^[a-zA-Z_-]+:.*##/) printf "  $(GREEN)%-20s$(RESET) %s\n", $$1, substr($$0, index($$0, "##") + 3) }' $(MAKEFILE_LIST)
	@echo ""
	@echo "$(BLUE)⚡ Performance Commands:$(RESET)"
	@awk '/^## ===.*PERFORMANCE/,/^## ===.*[^PERFORMANCE]/ { if(/^[a-zA-Z_-]+:.*##/) printf "  $(GREEN)%-20s$(RESET) %s\n", $$1, substr($$0, index($$0, "##") + 3) }' $(MAKEFILE_LIST)
	@echo ""
	@echo "$(BLUE)🐳 Docker Commands:$(RESET)"
	@awk '/^## ===.*DOCKER/,/^## ===.*[^DOCKER]/ { if(/^[a-zA-Z_-]+:.*##/) printf "  $(GREEN)%-20s$(RESET) %s\n", $$1, substr($$0, index($$0, "##") + 3) }' $(MAKEFILE_LIST)
	@echo ""
	@echo "$(BLUE)🗄️  Database Commands:$(RESET)"
	@awk '/^## ===.*DATABASE/,/^## ===.*[^DATABASE]/ { if(/^[a-zA-Z_-]+:.*##/) printf "  $(GREEN)%-20s$(RESET) %s\n", $$1, substr($$0, index($$0, "##") + 3) }' $(MAKEFILE_LIST)

version: ## Show version information
	@echo "$(CYAN)📋 Version Information:$(RESET)"
	@echo "Go Version: $(GO_VERSION)"
	@echo "Binary Name: $(BINARY_NAME)"
	@go version 2>/dev/null || echo "Go not installed"

## =============================================================================
## MAIN COMMANDS
## =============================================================================

build: create-dirs ## Build the application
	@echo "$(YELLOW)🏗️  Building $(BINARY_NAME)...$(RESET)"
	@go build -ldflags="-w -s -X main.version=$$(git describe --tags --always)" -o bin/$(BINARY_NAME) cmd/server/main.go
	@echo "$(GREEN)✅ Build completed: bin/$(BINARY_NAME)$(RESET)"

run: ## Run the application
	@echo "$(YELLOW)🚀 Running $(BINARY_NAME)...$(RESET)"
	@go run cmd/server/main.go

dev: docker-down clean ## Start full development environment
	@echo "$(CYAN)🚀 Starting PriceGuard API development environment...$(RESET)"
	@docker-compose -f $(DOCKER_COMPOSE_FILE) up --build -d
	@echo "$(GREEN)✅ Development environment started!$(RESET)"
	@echo ""
	@echo "$(CYAN)📊 Services:$(RESET)"
	@echo "  • API:           http://localhost:8080"
	@echo "  • Health Check:  http://localhost:8080/health"
	@echo "  • Database:      localhost:5432"
	@echo "  • Redis:         localhost:6379"
	@echo "  • Adminer:       http://localhost:8081"
	@echo ""
	@echo "$(CYAN)📋 Useful commands:$(RESET)"
	@echo "  • make logs      - View logs"
	@echo "  • make down      - Stop services"
	@echo "  • make test      - Run tests"

dev-local: ## Run with hot reload locally (requires air)
	@echo "$(YELLOW)🔥 Starting development server with hot reload...$(RESET)"
	@air || (echo "$(RED)❌ Air not installed. Run: make install-tools$(RESET)" && exit 1)

clean: ## Clean build artifacts and temporary files
	@echo "$(YELLOW)🧹 Cleaning build artifacts...$(RESET)"
	@rm -rf bin/ $(COVERAGE_DIR)/* $(PROFILE_DIR)/* $(REPORT_DIR)/*
	@go clean
	@echo "$(GREEN)✅ Clean completed$(RESET)"

deps: ## Install and verify dependencies
	@echo "$(YELLOW)📦 Installing dependencies...$(RESET)"
	@go mod tidy
	@go mod download
	@go mod verify
	@echo "$(GREEN)✅ Dependencies installed$(RESET)"

install-tools: ## Install development tools
	@echo "$(YELLOW)🔧 Installing development tools...$(RESET)"
	@go install github.com/air-verse/air@latest
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install golang.org/x/tools/cmd/goimports@latest
	@go install github.com/swaggo/swag/cmd/swag@latest
	@go install go.uber.org/mock/mockgen@latest
	@go install github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	@go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest
	@echo "$(GREEN)✅ Development tools installed$(RESET)"

## =============================================================================
## TESTING COMMANDS
## =============================================================================

test: create-dirs ## Run all tests
	@echo "$(YELLOW)🧪 Running all tests...$(RESET)"
	@go test -v -race -coverprofile=$(COVERAGE_DIR)/coverage.out ./...
	@echo "$(GREEN)✅ All tests completed$(RESET)"

test-unit: create-dirs ## Run unit tests only
	@echo "$(YELLOW)🧪 Running unit tests...$(RESET)"
	@go test -v -race -short -coverprofile=$(COVERAGE_DIR)/coverage-unit.out ./...
	@echo "$(GREEN)✅ Unit tests completed$(RESET)"

test-integration: create-dirs ## Run integration tests only
	@echo "$(YELLOW)🔗 Running integration tests...$(RESET)"
	@go test -v -race -tags=integration -coverprofile=$(COVERAGE_DIR)/coverage-integration.out ./...
	@echo "$(GREEN)✅ Integration tests completed$(RESET)"

test-unit-coverage: create-dirs ## Run unit tests with HTML coverage report
	@echo "$(YELLOW)🧪 Running unit tests with coverage...$(RESET)"
	@go test -v -race -short -coverprofile=$(COVERAGE_DIR)/coverage.out -covermode=atomic ./...
	@go tool cover -html=$(COVERAGE_DIR)/coverage.out -o $(COVERAGE_DIR)/coverage.html
	@go tool cover -func=$(COVERAGE_DIR)/coverage.out | tail -1
	@cp $(COVERAGE_DIR)/coverage.out ./coverage.out
	@cp $(COVERAGE_DIR)/coverage.html ./coverage.html
	@echo "$(GREEN)✅ Unit tests with coverage completed$(RESET)"
	@echo "$(CYAN)📊 Coverage report: $(COVERAGE_DIR)/coverage.html$(RESET)"

test-race: ## Run tests with race detection
	@echo "$(YELLOW)🏁 Running tests with race detection...$(RESET)"
	@go test -race -v ./...
	@echo "$(GREEN)✅ Race detection tests completed$(RESET)"

coverage-check: create-dirs ## Validate coverage meets minimum threshold
	@echo "$(YELLOW)📊 Checking coverage threshold...$(RESET)"
	@go test -coverprofile=$(COVERAGE_DIR)/coverage.out ./... > /dev/null
	@COVERAGE=$$(go tool cover -func=$(COVERAGE_DIR)/coverage.out | grep total | awk '{print $$3}' | sed 's/%//'); \
	if [ $$(echo "$$COVERAGE < $(MIN_COVERAGE)" | bc -l) -eq 1 ]; then \
		echo "$(RED)❌ Coverage $$COVERAGE% is below required $(MIN_COVERAGE)%$(RESET)"; \
		exit 1; \
	else \
		echo "$(GREEN)✅ Coverage $$COVERAGE% meets requirement ($(MIN_COVERAGE)%)$(RESET)"; \
	fi

test-watch: ## Run tests in watch mode (requires air)
	@echo "$(YELLOW)👁️  Running tests in watch mode...$(RESET)"
	@air -c .air-test.toml || echo "$(RED)❌ Create .air-test.toml configuration file$(RESET)"

## =============================================================================
## PERFORMANCE COMMANDS
## =============================================================================

benchmark: create-dirs ## Run basic benchmarks
	@echo "$(YELLOW)⚡ Running benchmarks...$(RESET)"
	@go test -bench=. -benchmem -count=$(BENCHMARK_COUNT) ./... > $(REPORT_DIR)/benchmark_basic.txt
	@echo "$(GREEN)✅ Benchmarks completed: $(REPORT_DIR)/benchmark_basic.txt$(RESET)"

benchmark-all: create-dirs ## Run all performance benchmarks
	@echo "$(YELLOW)⚡ Running comprehensive benchmarks...$(RESET)"
	@go test -bench=BenchmarkDatabase -benchmem -count=$(BENCHMARK_COUNT) ./... > $(REPORT_DIR)/benchmark_db.txt 2>/dev/null || echo "No database benchmarks found"
	@go test -bench=BenchmarkCache -benchmem -count=$(BENCHMARK_COUNT) ./... > $(REPORT_DIR)/benchmark_cache.txt 2>/dev/null || echo "No cache benchmarks found"
	@go test -bench=BenchmarkRedis -benchmem -count=$(BENCHMARK_COUNT) ./... > $(REPORT_DIR)/benchmark_redis.txt 2>/dev/null || echo "No Redis benchmarks found"
	@go test -bench=BenchmarkConcurrent -benchmem -count=$(BENCHMARK_COUNT) ./... > $(REPORT_DIR)/benchmark_concurrent.txt 2>/dev/null || echo "No concurrent benchmarks found"
	@echo "$(GREEN)✅ All benchmarks completed$(RESET)"

profile-cpu: create-dirs ## Generate CPU profile
	@echo "$(YELLOW)📈 Generating CPU profile...$(RESET)"
	@go test -bench=. -cpuprofile=$(PROFILE_DIR)/cpu.prof ./... || echo "No benchmarks available for profiling"
	@echo "$(CYAN)🔍 View profile: go tool pprof $(PROFILE_DIR)/cpu.prof$(RESET)"

profile-mem: create-dirs ## Generate memory profile
	@echo "$(YELLOW)🧠 Generating memory profile...$(RESET)"
	@go test -bench=. -memprofile=$(PROFILE_DIR)/mem.prof ./... || echo "No benchmarks available for profiling"
	@echo "$(CYAN)🔍 View profile: go tool pprof $(PROFILE_DIR)/mem.prof$(RESET)"

load-test: create-dirs ## Run load tests
	@echo "$(YELLOW)⚡ Running load tests...$(RESET)"
	@go test -bench=BenchmarkConcurrent -benchtime=$(BENCHMARK_TIME) -count=1 ./... > $(REPORT_DIR)/load_test.txt 2>/dev/null || echo "No concurrent benchmarks for load testing"
	@echo "$(GREEN)✅ Load test completed: $(REPORT_DIR)/load_test.txt$(RESET)"

## =============================================================================
## DOCKER COMMANDS  
## =============================================================================

docker-up: ## Start Docker services
	@echo "$(YELLOW)🐳 Starting Docker services...$(RESET)"
	@docker-compose -f $(DOCKER_COMPOSE_FILE) up -d
	@echo "$(GREEN)✅ Docker services started$(RESET)"

docker-down: ## Stop Docker services
	@echo "$(YELLOW)🛑 Stopping Docker services...$(RESET)"
	@docker-compose -f $(DOCKER_COMPOSE_FILE) down
	@echo "$(GREEN)✅ Docker services stopped$(RESET)"

docker-clean: ## Clean Docker environment completely
	@echo "$(YELLOW)🧹 Cleaning Docker environment...$(RESET)"
	@docker-compose -f $(DOCKER_COMPOSE_FILE) down -v --remove-orphans || true
	@docker system prune -af --volumes || true
	@echo "$(GREEN)✅ Docker environment cleaned$(RESET)"

docker-logs: ## Show Docker logs
	@echo "$(CYAN)📋 Docker logs:$(RESET)"
	@docker-compose -f $(DOCKER_COMPOSE_FILE) logs -f

docker-logs-api: ## Show API logs only
	@echo "$(CYAN)📋 API logs:$(RESET)"
	@docker-compose -f $(DOCKER_COMPOSE_FILE) logs -f api

status: ## Show Docker services status
	@echo "$(CYAN)📊 Services Status:$(RESET)"
	@docker-compose -f $(DOCKER_COMPOSE_FILE) ps

shell: ## Access API container shell
	@docker-compose -f $(DOCKER_COMPOSE_FILE) exec api sh

health: ## Check health of all services
	@echo "$(CYAN)🏥 Health Check:$(RESET)"
	@echo "API Health:" && curl -s http://localhost:8080/health | jq . || echo "$(RED)❌ API not responding$(RESET)"
	@echo "Database:" && docker-compose -f $(DOCKER_COMPOSE_FILE) exec postgres pg_isready -U postgres || echo "$(RED)❌ Database not ready$(RESET)"
	@echo "Redis:" && docker-compose -f $(DOCKER_COMPOSE_FILE) exec redis redis-cli ping || echo "$(RED)❌ Redis not ready$(RESET)"

## =============================================================================
## DATABASE COMMANDS
## =============================================================================

migrate-up: ## Run database migrations up
	@echo "$(YELLOW)📈 Running migrations up...$(RESET)"
	@migrate -path $(MIGRATE_PATH) -database "$(DATABASE_URL)" up
	@echo "$(GREEN)✅ Migrations completed$(RESET)"

migrate-down: ## Run database migrations down
	@echo "$(YELLOW)📉 Rolling back migrations...$(RESET)"
	@migrate -path $(MIGRATE_PATH) -database "$(DATABASE_URL)" down
	@echo "$(GREEN)✅ Rollback completed$(RESET)"

migrate-status: ## Check migration status
	@echo "$(CYAN)📊 Migration status:$(RESET)"
	@migrate -path $(MIGRATE_PATH) -database "$(DATABASE_URL)" version

migrate-create: ## Create new migration (usage: make migrate-create NAME=migration_name)
	@if [ -z "$(NAME)" ]; then echo "$(RED)❌ Usage: make migrate-create NAME=migration_name$(RESET)"; exit 1; fi
	@echo "$(YELLOW)📝 Creating migration: $(NAME)$(RESET)"
	@migrate create -ext sql -dir $(MIGRATE_PATH) $(NAME)
	@echo "$(GREEN)✅ Migration created$(RESET)"

db-shell: ## Access database shell
	@docker-compose -f $(DOCKER_COMPOSE_FILE) exec postgres psql -U postgres -d priceguard

redis-shell: ## Access Redis shell
	@docker-compose -f $(DOCKER_COMPOSE_FILE) exec redis redis-cli

backup-db: ## Backup database
	@echo "$(YELLOW)💾 Backing up database...$(RESET)"
	@docker-compose -f $(DOCKER_COMPOSE_FILE) exec postgres pg_dump -U postgres priceguard > backup_$(shell date +%Y%m%d_%H%M%S).sql
	@echo "$(GREEN)✅ Database backed up$(RESET)"

## =============================================================================
## CODE QUALITY & DOCS
## =============================================================================

lint: ## Run linter
	@echo "$(YELLOW)🔍 Running linter...$(RESET)"
	@golangci-lint run || (echo "$(RED)❌ Install golangci-lint: make install-tools$(RESET)" && exit 1)
	@echo "$(GREEN)✅ Linting completed$(RESET)"

fmt: ## Format code
	@echo "$(YELLOW)🎨 Formatting code...$(RESET)"
	@go fmt ./...
	@goimports -w . 2>/dev/null || echo "goimports not available"
	@echo "$(GREEN)✅ Code formatted$(RESET)"

security: ## Run security scan
	@echo "$(YELLOW)🔒 Running security scan...$(RESET)"
	@gosec ./... || (echo "$(RED)❌ Install gosec: make install-tools$(RESET)" && exit 1)
	@echo "$(GREEN)✅ Security scan completed$(RESET)"

docs: create-dirs ## Generate API documentation
	@echo "$(YELLOW)📚 Generating API documentation...$(RESET)"
	@swag init -g cmd/server/main.go -o $(DOCS_DIR)/swagger || (echo "$(RED)❌ Install swag: make install-tools$(RESET)" && exit 1)
	@echo "$(GREEN)✅ Documentation generated: $(DOCS_DIR)/swagger$(RESET)"

## =============================================================================
## UTILITIES
## =============================================================================

create-dirs: ## Create necessary directories
	@mkdir -p bin $(COVERAGE_DIR) $(PROFILE_DIR) $(REPORT_DIR) $(DOCS_DIR)/swagger

restart: ## Restart Docker services
	@echo "$(YELLOW)🔄 Restarting services...$(RESET)"
	@docker-compose -f $(DOCKER_COMPOSE_FILE) restart
	@echo "$(GREEN)✅ Services restarted$(RESET)"

restart-api: ## Restart API service only
	@echo "$(YELLOW)🔄 Restarting API service...$(RESET)"
	@docker-compose -f $(DOCKER_COMPOSE_FILE) restart api
	@echo "$(GREEN)✅ API service restarted$(RESET)"

## =============================================================================
## COMPOUND COMMANDS
## =============================================================================

dev-setup: deps install-tools create-dirs ## Complete development setup
	@echo "$(CYAN)🎯 Setting up development environment...$(RESET)"
	@cp .env.example .env 2>/dev/null || echo "$(YELLOW)⚠️  No .env.example found$(RESET)"
	@echo "$(GREEN)✅ Development environment ready!$(RESET)"
	@echo "$(CYAN)💡 Don't forget to update .env with your configuration!$(RESET)"

ci-test: test-unit-coverage lint security ## Run CI/CD tests (coverage + lint + security)
	@echo "$(GREEN)✅ CI/CD tests completed$(RESET)"

ci-build: clean build ## Clean build for CI/CD
	@echo "$(GREEN)✅ CI/CD build completed$(RESET)"

all: clean deps ci-test ci-build ## Run complete build pipeline
	@echo "$(GREEN)🎉 Complete build pipeline finished!$(RESET)"

quick-start: docker-up health ## Quick start without rebuild
	@echo "$(GREEN)⚡ Quick start completed!$(RESET)"

full-reset: docker-clean clean deps dev ## Complete reset and restart
	@echo "$(GREEN)🔄 Full environment reset completed!$(RESET)"