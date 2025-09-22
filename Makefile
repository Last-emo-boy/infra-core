# InfraCore Makefile
# Author: last-emo-boy

.PHONY: help build build-ui test clean dev prod logs stop restart health

# Default environment
ENV ?= development

# Docker configuration
DOCKER_IMAGE = infra-core
DOCKER_TAG = latest

# Colors for output
RED = \033[0;31m
GREEN = \033[0;32m
YELLOW = \033[0;33m
BLUE = \033[0;34m
CYAN = \033[0;36m
NC = \033[0m # No Color

# Detect OS for executable suffix
ifeq ($(OS),Windows_NT)
    EXE_SUFFIX := .exe
else
    EXE_SUFFIX :=
endif

help: ## Show this help message
	@echo "${CYAN}InfraCore Build System${NC}"
	@echo "======================"
	@echo ""
	@echo "Available commands:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  ${GREEN}%-15s${NC} %s\n", $$1, $$2}'
	@echo ""
	@echo "Environment variables:"
	@echo "  ${BLUE}ENV${NC}        Environment to use (development|production) [default: development]"
	@echo ""
	@echo "Examples:"
	@echo "  make build              # Build for development"
	@echo "  make build ENV=production"
	@echo "  make dev                # Start development environment"
	@echo "  make prod               # Start production environment"

build: ## Build the application
	@echo "${YELLOW}🔨 Building InfraCore...${NC}"
	@echo "${BLUE}Building Go applications...${NC}"
	@mkdir -p bin
	@go build -o bin/gate$(EXE_SUFFIX) cmd/gate/main.go
	@go build -o bin/console$(EXE_SUFFIX) cmd/console/main.go
	@echo "${GREEN}✅ Go applications built successfully${NC}"

build-ui: ## Build the frontend UI
	@echo "${BLUE}Building frontend UI...${NC}"
	@cd ui && npm install && npm run build
	@echo "${GREEN}✅ Frontend UI built successfully${NC}"

build-all: build build-ui ## Build both backend and frontend

test: ## Run tests
	@echo "${YELLOW}🧪 Running tests...${NC}"
	@go test -v ./...
	@echo "${GREEN}✅ All tests passed${NC}"

test-api: ## Test API endpoints
	@echo "${YELLOW}🧪 Testing API endpoints...${NC}"
	@go run cmd/api-test/main.go

clean: ## Clean build artifacts
	@echo "${YELLOW}🧹 Cleaning build artifacts...${NC}"
	@rm -rf bin/
	@rm -rf ui/dist/
	@rm -rf ui/node_modules/
	@docker system prune -f
	@echo "${GREEN}✅ Cleanup completed${NC}"

dev: ## Start development environment
	@echo "${YELLOW}🛠️  Starting development environment...${NC}"
	@docker-compose -f docker-compose.dev.yml down
	@docker-compose -f docker-compose.dev.yml up --build -d
	@$(MAKE) health
	@echo "${GREEN}🎉 Development environment started!${NC}"
	@echo "${CYAN}Frontend: http://localhost:5173${NC}"
	@echo "${CYAN}Backend: http://localhost:8082${NC}"

prod: ## Start production environment
	@echo "${YELLOW}🏭 Starting production environment...${NC}"
	@docker-compose down
	@docker-compose up --build -d
	@$(MAKE) health
	@echo "${GREEN}🎉 Production environment started!${NC}"
	@echo "${CYAN}Console: http://localhost:8082${NC}"

logs: ## Show logs
	@echo "${BLUE}📋 Showing logs...${NC}"
ifeq ($(ENV),production)
	@docker-compose logs -f
else
	@docker-compose -f docker-compose.dev.yml logs -f
endif

stop: ## Stop all services
	@echo "${YELLOW}⏹️  Stopping services...${NC}"
ifeq ($(ENV),production)
	@docker-compose down
else
	@docker-compose -f docker-compose.dev.yml down
endif
	@echo "${GREEN}✅ Services stopped${NC}"

restart: stop ## Restart services
	@echo "${YELLOW}🔄 Restarting services...${NC}"
ifeq ($(ENV),production)
	@$(MAKE) prod
else
	@$(MAKE) dev
endif

health: ## Check service health
	@echo "${BLUE}🏥 Checking service health...${NC}"
	@for i in {1..30}; do \
		if curl -f http://localhost:8082/api/v1/health >/dev/null 2>&1; then \
			echo "${GREEN}✅ Health check passed${NC}"; \
			exit 0; \
		fi; \
		echo "⏳ Attempt $$i/30 - waiting for service..."; \
		sleep 2; \
	done; \
	echo "${RED}❌ Health check failed${NC}"; \
	exit 1

status: ## Show service status
	@echo "${BLUE}📊 Service Status:${NC}"
	@echo "=================="
ifeq ($(ENV),production)
	@docker-compose ps
else
	@docker-compose -f docker-compose.dev.yml ps
endif

# Development helpers
dev-backend: ## Run backend in development mode
	@echo "${BLUE}🔧 Starting backend in development mode...${NC}"
	@INFRA_CORE_ENV=development go run cmd/console/main.go

dev-frontend: ## Run frontend in development mode
	@echo "${BLUE}🔧 Starting frontend in development mode...${NC}"
	@cd ui && npm run dev

install-deps: ## Install dependencies
	@echo "${BLUE}📦 Installing dependencies...${NC}"
	@go mod download
	@cd ui && npm install
	@echo "${GREEN}✅ Dependencies installed${NC}"

# Database operations
db-migrate: ## Run database migrations
	@echo "${BLUE}🗃️  Running database migrations...${NC}"
	@go run cmd/db-test/main.go

# Deployment
deploy-linux: ## Deploy to Linux server
	@echo "${YELLOW}🚀 Deploying to Linux...${NC}"
	@chmod +x deploy.sh
	@./deploy.sh $(ENV)

deploy-windows: ## Deploy on Windows
	@echo "${YELLOW}🚀 Deploying on Windows...${NC}"
	@powershell -ExecutionPolicy Bypass -File deploy.ps1 -Environment $(ENV)

# Docker helpers
docker-build: ## Build Docker image
	@echo "${BLUE}🐳 Building Docker image...${NC}"
	@docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) .
	@echo "${GREEN}✅ Docker image built: $(DOCKER_IMAGE):$(DOCKER_TAG)${NC}"

docker-push: ## Push Docker image (requires registry config)
	@echo "${BLUE}🐳 Pushing Docker image...${NC}"
	@docker push $(DOCKER_IMAGE):$(DOCKER_TAG)

# Info commands
version: ## Show version information
	@echo "${CYAN}InfraCore Version Information${NC}"
	@echo "============================"
	@echo "Go version: $(shell go version)"
	@echo "Node version: $(shell node --version 2>/dev/null || echo 'Not installed')"
	@echo "Docker version: $(shell docker --version 2>/dev/null || echo 'Not installed')"
	@echo "Environment: $(ENV)"

config: ## Show current configuration
	@echo "${CYAN}Current Configuration${NC}"
	@echo "===================="
	@echo "Environment: $(ENV)"
	@echo "Docker Image: $(DOCKER_IMAGE):$(DOCKER_TAG)"
	@echo "Config files:"
ifeq ($(ENV),production)
	@ls -la configs/production.yaml 2>/dev/null || echo "  Production config not found"
else
	@ls -la configs/development.yaml 2>/dev/null || echo "  Development config not found"
endif