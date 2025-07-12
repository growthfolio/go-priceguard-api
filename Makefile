# Go PriceGuard API Makefile

.PHONY: help build run test test-cover lint clean docker-up docker-down migrate-up migrate-down deps dev install-tools docs

# Variables
BINARY_NAME=priceguard-api
DOCKER_COMPOSE_FILE=docker-compose.yml
MIGRATE_PATH=./db/migrations
DATABASE_URL=postgres://postgres:password@localhost:5432/priceguard?sslmode=disable

# Default target
help: ## Show this help message
	@echo 'Management commands for PriceGuard API:'
	@echo
	@echo 'Usage:'
	@echo '    make build           Compile the project'
	@echo '    make run             Run the application'
	@echo '    make dev             Run with hot reload (air)'
	@echo '    make test            Run tests'
	@echo '    make test-cover      Run tests with coverage'
	@echo '    make test-race       Run tests with race detection'
	@echo '    make lint            Run linter'
	@echo '    make fmt             Format code'
	@echo '    make clean           Clean build files'
	@echo '    make deps            Install dependencies'
	@echo '    make install-tools   Install development tools'
	@echo '    make docker-up       Start all services with Docker'
	@echo '    make docker-down     Stop all Docker services'
	@echo '    make migrate-up      Run database migrations'
	@echo '    make migrate-down    Rollback database migrations'
	@echo '    make docs            Generate API documentation'
	@echo

# Build the application
build: ## Compile the application
	@echo "Building $(BINARY_NAME)..."
	@go build -o bin/$(BINARY_NAME) cmd/server/main.go

# Run the application
run: ## Run the application
	@echo "Running $(BINARY_NAME)..."
	@go run cmd/server/main.go

# Install dependencies
deps: ## Install project dependencies
	@echo "Installing dependencies..."
	@go mod tidy
	@go mod download

# Run tests
test: ## Run all tests
	@echo "Running tests..."
	@go test -v ./...

# Run tests with coverage
test-cover: ## Run tests with coverage report
	@echo "Running tests with coverage..."
	@go test -v -race -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run tests with race detection
test-race: ## Run tests with race detection
	@echo "Running tests with race detection..."
	@go test -race -v ./...

# Run linter
lint: ## Run golangci-lint
	@echo "Running linter..."
	@golangci-lint run

# Clean build files
clean: ## Clean build artifacts
	@echo "Cleaning..."
	@rm -rf bin/
	@rm -f coverage.out coverage.html
	@go clean

# Development mode with hot reload (requires air)
dev: ## Run in development mode with hot reload
	@echo "Starting development server with hot reload..."
	@air

# Docker commands
docker-up: ## Start all services with Docker Compose
	@echo "Starting Docker services..."
	@docker-compose -f $(DOCKER_COMPOSE_FILE) up -d

docker-down: ## Stop Docker services
	@echo "Stopping Docker services..."
	@docker-compose -f $(DOCKER_COMPOSE_FILE) down

docker-logs: ## Show Docker logs
	@docker-compose -f $(DOCKER_COMPOSE_FILE) logs -f

# Database migrations (requires golang-migrate)
migrate-up: ## Run database migrations
	@echo "Running database migrations..."
	@migrate -path $(MIGRATE_PATH) -database "$(DATABASE_URL)" up

migrate-down: ## Rollback database migrations
	@echo "Rolling back database migrations..."
	@migrate -path $(MIGRATE_PATH) -database "$(DATABASE_URL)" down

migrate-create: ## Create a new migration file (usage: make migrate-create NAME=migration_name)
	@if [ -z "$(NAME)" ]; then echo "Usage: make migrate-create NAME=migration_name"; exit 1; fi
	@echo "Creating migration: $(NAME)"
	@migrate create -ext sql -dir $(MIGRATE_PATH) $(NAME)

# Documentation
docs: ## Generate API documentation
	@echo "Generating API documentation..."
	@swag init -g cmd/server/main.go -o docs/swagger

# Install development tools
install-tools: ## Install development tools
	@echo "Installing development tools..."
	@go install github.com/air-verse/air@latest
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install golang.org/x/tools/cmd/goimports@latest
	@go install github.com/swaggo/swag/cmd/swag@latest
	@go install go.uber.org/mock/mockgen@latest
	@go install github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Development setup
dev-setup: deps install-tools ## Setup complete development environment
	@echo "Setting up development environment..."
	@cp .env.example .env || true
	@echo "Development environment ready!"
	@echo "Don't forget to update .env with your configuration!"

# All-in-one development start
dev-start: docker-up ## Start complete development environment
	@echo "Starting development environment..."
	@echo "Waiting for services to be ready..."
	@sleep 5
	@echo "Development environment ready!"
	@echo "API: http://localhost:8080"
	@echo "Adminer: http://localhost:8081"
	@echo "Redis Commander: http://localhost:8082"

# Production build
build-prod: ## Build for production
	@echo "Building for production..."
	@CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-extldflags "-static"' -o bin/$(BINARY_NAME) cmd/server/main.go
