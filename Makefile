# Variables
BINARY=baihu
GOBUILD=go build
GOCLEAN=go clean
GOGET=go get
GOMOD=go mod
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date '+%Y/%m/%d %H:%M:%S')
LDFLAGS=-ldflags="-s -w -X 'baihu/internal/constant.Version=$(VERSION)' -X 'baihu/internal/constant.BuildTime=$(BUILD_TIME)'"

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

# Build agent for all platforms (local development)
build-agent:
	@mkdir -p data/agent
	@echo "$(VERSION)" > data/agent/version.txt
	@echo "Building and packaging agents..."
	cd agent && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w -X 'main.Version=$(VERSION)' -X 'main.BuildTime=$(BUILD_TIME)'" -o baihu-agent . && \
		tar -czvf ../data/agent/baihu-agent-linux-amd64.tar.gz baihu-agent config.example.ini && rm baihu-agent
	cd agent && CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="-s -w -X 'main.Version=$(VERSION)' -X 'main.BuildTime=$(BUILD_TIME)'" -o baihu-agent . && \
		tar -czvf ../data/agent/baihu-agent-linux-arm64.tar.gz baihu-agent config.example.ini && rm baihu-agent
	cd agent && CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags="-s -w -X 'main.Version=$(VERSION)' -X 'main.BuildTime=$(BUILD_TIME)'" -o baihu-agent.exe . && \
		tar -czvf ../data/agent/baihu-agent-windows-amd64.tar.gz baihu-agent.exe config.example.ini && rm baihu-agent.exe
	cd agent && CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w -X 'main.Version=$(VERSION)' -X 'main.BuildTime=$(BUILD_TIME)'" -o baihu-agent . && \
		tar -czvf ../data/agent/baihu-agent-darwin-amd64.tar.gz baihu-agent config.example.ini && rm baihu-agent
	cd agent && CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w -X 'main.Version=$(VERSION)' -X 'main.BuildTime=$(BUILD_TIME)'" -o baihu-agent . && \
		tar -czvf ../data/agent/baihu-agent-darwin-arm64.tar.gz baihu-agent config.example.ini && rm baihu-agent
	@echo "Agent packages built in data/agent/"
	@ls -lh data/agent/*.tar.gz

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
	docker build -t $(BINARY) .

# Docker run
docker-run:
	docker run -p 8052:8052 $(BINARY)

# Docker compose up
docker-up:
	docker-compose up -d

# Docker compose down
docker-down:
	docker-compose down

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