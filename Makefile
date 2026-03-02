# Variables
BINARY=bin/baihu
GOBUILD=go build
GOCLEAN=go clean
GOGET=go get
GOMOD=go mod
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date '+%Y/%m/%d %H:%M:%S')
LDFLAGS=-ldflags="-s -w -X 'github.com/engigu/baihu-panel/internal/constant.Version=$(VERSION)' -X 'github.com/engigu/baihu-panel/internal/constant.BuildTime=$(BUILD_TIME)'"

# Default target
all: build

# Build frontend
build-web:
	cd web && npm ci && npm run build
	rm -rf internal/static/dist
	cp -r web/dist internal/static/dist

# Build the application (requires frontend to be built first)
build:
	@mkdir -p bin
	CGO_ENABLED=0 $(GOBUILD) $(LDFLAGS) -o $(BINARY) main.go

# Build all (frontend + backend)
build-all: build-web build

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
	mkdir -p internal/static/dist
	touch internal/static/dist/.gitkeep
	rm -rf web/dist
	mkdir -p web/dist
	touch web/dist/.gitkeep

# Run the application
run:
	@mkdir -p bin
	$(GOBUILD) -o $(BINARY) main.go
	./$(BINARY) server

# Development run with hot reload (both frontend and backend)
dev:
	@command -v concurrently > /dev/null 2>&1 || npm install -g concurrently
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

# Docker build
docker-build:
	docker build -t baihu:dev -f docker/Dockerfile .

# Docker run
docker-run:
	docker run -p 8052:8052 baihu:dev

# Docker compose up
docker-up:
	docker-compose up -d

# Docker compose down
docker-down:
	docker-compose down

# Build and run in development mode (foreground with logs)
docker-dev:
	docker-compose down
	docker-compose build
	docker-compose up

# Help
help:
	@echo "Available targets:"
	@echo "  all            - Build the application (default)"
	@echo "  build          - Build the application"
	@echo "  build-agent    - Build agent packages (tar.gz) for all platforms"
	@echo "  clean          - Clean built files"
	@echo "  run            - Run the application"
	@echo "  deps           - Install dependencies"
	@echo "  docker-build   - Build Docker image"
	@echo "  docker-run     - Run Docker container"
	@echo "  docker-up      - Start Docker Compose stack"
	@echo "  docker-down    - Stop Docker Compose stack"
	@echo "  docker-dev     - Build and run Docker Compose in foreground"
	@echo "  help           - Show this help message"
