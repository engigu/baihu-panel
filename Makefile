# Variables
BINARY=baihu
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
	CGO_ENABLED=0 $(GOBUILD) $(LDFLAGS) -o $(BINARY) main.go

# Build all (frontend + backend)
build-all: build-web build

# Build agent for all platforms
build-agent: build-agent-linux-amd64 build-agent-linux-arm64 build-agent-windows-amd64 build-agent-darwin-amd64 build-agent-darwin-arm64
	@echo "All agent packages built in data/agent/"
	@ls -lh data/agent/*.tar.gz

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
	rm -f $(BINARY)
	rm -rf internal/static/dist
	mkdir -p internal/static/dist
	touch internal/static/dist/.gitkeep

# Run the application
run:
	$(GOBUILD) -o $(BINARY) main.go
	./$(BINARY)

# Development run (without embedding frontend)
dev:
	$(GOBUILD) -o $(BINARY) main.go
	./$(BINARY)

# Install dependencies
deps:
	$(GOMOD) tidy

# Docker build
docker-build:
	docker build -t $(BINARY) -f docker/Dockerfile .

# Docker run
docker-run:
	docker run -p 8052:8052 $(BINARY)

# Docker compose up
docker-up:
	docker-compose -f docker/docker-compose.yml up -d

# Docker compose down
docker-down:
	docker-compose -f docker/docker-compose.yml down

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
	@echo "  help           - Show this help message"