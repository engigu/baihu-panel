# ================================
# Stage 1: Build frontend
# ================================
FROM node:20-alpine AS frontend-builder

WORKDIR /app/web

# Copy package files
COPY web/package*.json ./

# Install dependencies
RUN npm ci

# Copy frontend source
COPY web/ ./

# Build frontend
RUN npm run build

# ================================
# Stage 2: Build backend
# ================================
FROM --platform=$BUILDPLATFORM golang:1.24-alpine AS backend-builder

ARG TARGETOS
ARG TARGETARCH
ARG VERSION
ARG BUILD_TIME

WORKDIR /app

# Go mod files
COPY go.mod go.sum ./
RUN go env -w GOPROXY=https://goproxy.cn,direct && go mod download

# Copy backend source
COPY . .

# Copy frontend dist
COPY --from=frontend-builder /app/web/dist ./internal/static/dist

# Build Go binary - 如果 VERSION/BUILD_TIME 为空则使用默认值
RUN VERSION_VAL="${VERSION:-dev-$(date '+%Y%m%d%H%M%S')}" && \
    BUILD_TIME_VAL="${BUILD_TIME:-$(date '+%Y-%m-%d %H:%M:%S')}" && \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -ldflags="-s -w -X baihu/internal/constant.Version=${VERSION_VAL} -X 'baihu/internal/constant.BuildTime=${BUILD_TIME_VAL}'" \
    -o baihu .

# ================================
# Stage 3: Build Agent (all platforms)
# ================================
FROM --platform=$BUILDPLATFORM golang:1.24-alpine AS agent-builder

ARG VERSION
ARG BUILD_TIME

WORKDIR /app

# Copy agent source and config example
COPY agent/ ./agent/

# Download dependencies
WORKDIR /app/agent
RUN go env -w GOPROXY=https://goproxy.cn,direct && go mod download

# Build agent for all platforms and package as tar.gz (sequential to avoid file conflicts)
RUN VERSION_VAL="${VERSION:-dev-$(date '+%Y%m%d%H%M%S')}" && \
    BUILD_TIME_VAL="${BUILD_TIME:-$(date '+%Y-%m-%d %H:%M:%S')}" && \
    mkdir -p /opt/agent && \
    echo "${VERSION_VAL}" > /opt/agent/version.txt && \
    # Linux amd64
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w -X 'main.Version=${VERSION_VAL}' -X 'main.BuildTime=${BUILD_TIME_VAL}'" -o baihu-agent . && \
    tar -czvf /opt/agent/baihu-agent-linux-amd64.tar.gz baihu-agent config.example.ini && rm baihu-agent && \
    # Linux arm64
    # CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="-s -w -X 'main.Version=${VERSION_VAL}' -X 'main.BuildTime=${BUILD_TIME_VAL}'" -o baihu-agent . && \
    # tar -czvf /opt/agent/baihu-agent-linux-arm64.tar.gz baihu-agent config.example.ini && rm baihu-agent && \
    # # Windows amd64
    # CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags="-s -w -X 'main.Version=${VERSION_VAL}' -X 'main.BuildTime=${BUILD_TIME_VAL}'" -o baihu-agent.exe . && \
    # tar -czvf /opt/agent/baihu-agent-windows-amd64.tar.gz baihu-agent.exe config.example.ini && rm baihu-agent.exe && \
    # # Darwin amd64
    # CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w -X 'main.Version=${VERSION_VAL}' -X 'main.BuildTime=${BUILD_TIME_VAL}'" -o baihu-agent . && \
    # tar -czvf /opt/agent/baihu-agent-darwin-amd64.tar.gz baihu-agent config.example.ini && rm baihu-agent && \
    # # Darwin arm64
    # CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w -X 'main.Version=${VERSION_VAL}' -X 'main.BuildTime=${BUILD_TIME_VAL}'" -o baihu-agent . && \
    # tar -czvf /opt/agent/baihu-agent-darwin-arm64.tar.gz baihu-agent config.example.ini && rm baihu-agent
    echo "Agent build completed"
# ================================
# Stage 4: Final image
# ================================
FROM debian:bookworm-slim

ENV TZ=Asia/Shanghai
ENV PATH="/app/envs/node/bin:/app/envs/python/bin:$PATH"
ENV NODE_PATH="/app/envs/node/lib/node_modules"

# 安装必要系统工具 + Node + Python
RUN sed -i 's@deb.debian.org@mirrors.tuna.tsinghua.edu.cn@g' /etc/apt/sources.list.d/debian.sources \
    && sed -i 's@security.debian.org@mirrors.tuna.tsinghua.edu.cn@g' /etc/apt/sources.list.d/debian.sources \
    && echo "${TZ}" > /etc/timezone \
    && ln -sf /usr/share/zoneinfo/${TZ} /etc/localtime \
    && apt-get update \
    && apt-get install -y --no-install-recommends \
         tzdata git gcc curl wget vim nodejs htop npm python3 python3-venv python3-pip \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy Go binary
COPY --from=backend-builder /app/baihu .

# Copy frontend dist folder
COPY --from=frontend-builder /app/web/dist ./internal/static/dist

# Copy configs and entrypoint
COPY --from=backend-builder /app/configs ./configs
COPY docker-entrypoint.sh .

# Copy sync.py to /opt
COPY custom/sync.py /opt/sync.py

# Copy agent binaries to /opt/agent
COPY --from=agent-builder /opt/agent /opt/agent

RUN chmod +x /opt/sync.py \
    && chmod +x docker-entrypoint.sh \
    && touch "dont-not-delete-anythings" \
    && echo "set encoding=utf-8" >> /etc/vim/vimrc

EXPOSE 8052

ENTRYPOINT ["./docker-entrypoint.sh"]
