# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [v1.0.0-dev] - 2025-07-13

### 🚀 Added

#### Development Environment
- **Complete Docker Compose setup** with PostgreSQL, Redis, API, Adminer, and Redis Commander
- **Hot reload development** with Air for automatic recompilation
- **Environment configuration** with proper Docker networking and volumes
- **Health checks** for all services to ensure reliability
- **Persistent data storage** with Docker volumes

#### Scripts and Automation
- **Development startup script** (`scripts/start.sh`) with error recovery
- **Environment verification script** (`scripts/check-environment.sh`)
- **Comprehensive Makefile** (`Makefile.dev`) with colored output and development commands
- **Air configuration** (`.air.toml`) for optimized hot reload

#### Documentation
- **Complete OpenAPI 3.0 specification** (852 lines) in `docs/api-spec.yaml`
- **Postman collection** and environment files for API testing
- **Comprehensive testing guides** in Portuguese and English
- **Swagger integration examples** for documentation setup
- **Development setup instructions** and quick start guides

#### Database
- **PostgreSQL initialization script** with extensions and timezone setup
- **Database migration** support and schema management
- **Redis configuration** for caching and session management

#### Configuration
- **Development environment variables** (`.env`) with Docker-optimized settings
- **Docker development image** (`Dockerfile.dev`) with optimized layers
- **Service networking** and inter-container communication

### 🏗️ Infrastructure

#### Services Available
- **API Server**: `http://localhost:8080` - Main application with hot reload
- **PostgreSQL**: `localhost:5432` - Primary database
- **Redis**: `localhost:6379` - Cache and sessions
- **Adminer**: `http://localhost:8081` - Database web interface
- **Redis Commander**: `http://localhost:8082` - Redis web interface

#### Key Endpoints
- **Health Check**: `GET /health` - Service health monitoring
- **Metrics**: `GET /metrics` - Application metrics
- **WebSocket**: `ws://localhost:8080/ws` - Real-time connections
- **API Routes**: `/api/*` - RESTful API endpoints

### 🔧 Development Tools

#### Quick Commands
```bash
# Start complete environment
make -f Makefile.dev dev

# View logs
make -f Makefile.dev logs

# Check environment status
./scripts/check-environment.sh

# Access containers
make -f Makefile.dev shell     # API container
make -f Makefile.dev db-shell  # PostgreSQL
```

#### Features
- **Automatic dependency management** with Go modules
- **Code formatting** and linting integration
- **Test execution** with coverage reports
- **Database backup** and restore capabilities
- **Container debugging** and monitoring tools

### 🌟 BREAKING CHANGES
- **Development environment now requires Docker and Docker Compose**
- **New port mappings** and service configurations
- **Updated environment variable structure** for Docker compatibility

### 🐛 Bug Fixes
- **Fixed buildvcs issues** in Docker compilation
- **Resolved container networking** problems
- **Corrected environment variable** propagation
- **Fixed hot reload** file watching patterns

---

## How to Update

1. **Pull latest changes**: `git pull origin main`
2. **Clean Docker environment**: `make -f Makefile.dev clean`
3. **Start development**: `make -f Makefile.dev dev`
4. **Verify setup**: `./scripts/check-environment.sh`

## Requirements

- Docker 20.10+
- Docker Compose 2.0+
- Git 2.30+
- Make 4.0+

## Quick Start

```bash
# Clone and setup
git clone <repository>
cd go-priceguard-api

# Start development environment
make -f Makefile.dev dev

# Check if everything is working
./scripts/check-environment.sh

# View API documentation
# Import docs/PriceGuard_API.postman_collection.json into Postman
```

---

**Full development environment ready! 🎉**
