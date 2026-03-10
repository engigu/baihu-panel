# Variables
BINARY=bin/baihu
GOBUILD=go build
GOCLEAN=go clean
GOGET=go get
GOMOD=go mod
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date '+%Y/%m/%d %H:%M:%S')
LDFLAGS=-ldflags="-s -w -X 'github.com/engigu/baihu-panel/internal/constant.Version=$(VERSION)' -X 'github.com/engigu/baihu-panel/internal/constant.BuildTime=$(BUILD_TIME)'"

TAGS_WEB=-tags web

DEV_UID ?= $(shell id -u 2>/dev/null || echo 1000)
DEV_GID ?= $(shell id -g 2>/dev/null || echo 1000)
export DEV_UID
export DEV_GID

# Default target
all: build

# Build frontend
build-web:
	cd web && npm ci && npm run build

# Build the application (requires frontend to be built first)
build:
	@mkdir -p bin
	CGO_ENABLED=0 $(GOBUILD) $(LDFLAGS) -o $(BINARY) main.go

# Build release version (Frontend + Backend with embedded assets)
release: build-web
	@mkdir -p bin
	rm -rf internal/static/dist
	cp -r web/dist internal/static/dist
	CGO_ENABLED=0 $(GOBUILD) $(LDFLAGS) $(TAGS_WEB) -o $(BINARY) main.go

# Alias for backward compatibility
build-all: release

# Build agent for all platforms
build-agent: build-agent-linux-amd64 build-agent-linux-arm64 build-agent-windows-amd64 build-agent-darwin-amd64 build-agent-darwin-arm64
	@echo "All agent packages built in data/agent/"
	@ls -lh data/agent/baihu-agent-*

AGENT_LDFLAGS=-s -w -X 'main.Version=$(VERSION)' -X 'main.BuildTime=$(BUILD_TIME)'

build-agent-linux-amd64:
	@mkdir -p data/agent
	@echo "$(VERSION)" > data/agent/version.txt
	cd agent && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="$(AGENT_LDFLAGS)" -o ../data/agent/baihu-agent-linux-amd64 .

build-agent-linux-arm64:
	@mkdir -p data/agent
	@echo "$(VERSION)" > data/agent/version.txt
	cd agent && CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="$(AGENT_LDFLAGS)" -o ../data/agent/baihu-agent-linux-arm64 .

build-agent-windows-amd64:
	@mkdir -p data/agent
	@echo "$(VERSION)" > data/agent/version.txt
	cd agent && CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags="$(AGENT_LDFLAGS)" -o ../data/agent/baihu-agent-windows-amd64.exe .

build-agent-darwin-amd64:
	@mkdir -p data/agent
	@echo "$(VERSION)" > data/agent/version.txt
	cd agent && CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags="$(AGENT_LDFLAGS)" -o ../data/agent/baihu-agent-darwin-amd64 .

build-agent-darwin-arm64:
	@mkdir -p data/agent
	@echo "$(VERSION)" > data/agent/version.txt
	cd agent && CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags="$(AGENT_LDFLAGS)" -o ../data/agent/baihu-agent-darwin-arm64 .

# Clean built files
clean:
	$(GOCLEAN)
	rm -rf bin/
	rm -rf internal/static/dist
	rm -rf web/dist

# Clean everything: local artifacts and Docker development environment (including volumes)
clean-all: clean docker-dev-clean
	rm -rf web/node_modules
	@echo "All local artifacts and Docker dev caches have been completely wiped."

# Run the application
run:
	@mkdir -p bin
	$(GOBUILD) -o $(BINARY) main.go
	./$(BINARY) server

# Development run with hot reload (both frontend and backend)
dev:
	@command -v concurrently > /dev/null 2>&1 || npm install -g concurrently
	@mkdir -p envs web/node_modules
	concurrently --kill-others \
		"go tool air" \
		"cd web && npm ci && npm run dev"

# Run agent with hot reload
agent-dev:
	go tool air -c agent.air.toml

# Run agent
agent-run:
	@mkdir -p bin
	$(GOBUILD) -o bin/baihu-agent ./agent
	./bin/baihu-agent run -c ../agent/config.ini

# Install dependencies
deps:
	$(GOMOD) tidy

# Generate swagger documentation
swag:
	@mkdir -p docs/public
	go run github.com/swaggo/swag/cmd/swag@latest init -g main.go -o ./docs/public --ot json,yaml

docs-dev:
	cd docs && npm run docs:dev

docs-build:
	cd docs && npm run docs:build

# Docker build
docker-build:
	docker build -t baihu:dev -f docker/Dockerfile .

# Docker run
docker-run:
	docker run -p 8052:8052 baihu:dev

# Docker compose up
docker-up:
	docker compose up -d

# Docker compose down
docker-down:
	docker compose down

# Start isolated Docker dev environment (foreground with logs, Ctrl+C to stop)
docker-dev:
	@command -v concurrently > /dev/null 2>&1 || npm install -g concurrently
	@mkdir -p envs web/node_modules
	docker compose -f docker-compose.dev.yml up --build

# Start isolated Docker dev environment (background)
docker-dev-d:
	docker compose -f docker-compose.dev.yml up -d --build

# Stop Docker dev environment (preserves cached volumes for fast restart)
docker-dev-down:
	docker compose -f docker-compose.dev.yml down

# Stop and completely clean Docker dev environment (removes all cached volumes)
# Use this if your environment is broken or you want a fresh start
docker-dev-clean:
	docker compose -f docker-compose.dev.yml down -v

# Help
help:
	@echo "Available targets:"
	@echo "  all              - Build backend only (default)"
	@echo "  build            - Build backend binary (no UI embedded)"
	@echo "  release          - Build full release binary (with UI embedded)"
	@echo "  build-web        - Build frontend assets only"
	@echo "  build-agent      - Build agent packages (tar.gz) for all platforms"
	@echo "  clean            - Clean built files"
	@echo "  clean-all        - Clean local files and Docker dev environment (including volumes)"
	@echo "  run              - Run the application locally"
	@echo "  dev              - Run local development with hot reload"
	@echo "  deps             - Install Go dependencies"
	@echo "  docker-build     - Build production Docker image"
	@echo "  docker-run       - Run production Docker container"
	@echo "  docker-up        - Start production Docker Compose stack"
	@echo "  docker-down      - Stop production Docker Compose stack"
	@echo "  docker-dev       - Start isolated Docker dev environment (foreground)"
	@echo "  docker-dev-d     - Start isolated Docker dev environment (background)"
	@echo "  docker-dev-down  - Stop Docker dev environment (keep caches)"
	@echo "  docker-dev-clean - Stop and clean Docker dev environment (remove caches)"
	@echo "  swag             - Generate swagger documentation and sync with docs"
	@echo "  docs-dev         - Run documentation development server"
	@echo "  docs-build       - Build documentation"
	@echo "  help             - Show this help message"
