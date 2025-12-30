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
# Stage 0: Generate build info
# ================================
FROM alpine:3.19 AS build-info

ARG VERSION
ARG BUILD_TIME

# 生成版本信息文件，确保主服务和 Agent 使用相同的值（使用东八区时间）
RUN VERSION_VAL="${VERSION:-dev-$(TZ=Asia/Shanghai date '+%Y%m%d%H%M%S')}" && \
    BUILD_TIME_VAL="${BUILD_TIME:-$(TZ=Asia/Shanghai date '+%Y-%m-%d %H:%M:%S')}" && \
    mkdir -p /build-info && \
    echo "${VERSION_VAL}" > /build-info/version.txt && \
    echo "${BUILD_TIME_VAL}" > /build-info/build_time.txt

# ================================
# Stage 2: Build backend
# ================================
FROM --platform=$BUILDPLATFORM golang:1.24-alpine AS backend-builder

ARG TARGETOS
ARG TARGETARCH

WORKDIR /app

# Copy build info
COPY --from=build-info /build-info /build-info

# Go mod files
COPY go.mod go.sum ./
RUN go env -w GOPROXY=https://goproxy.cn,direct && go mod download

# Copy backend source
COPY . .

# Copy frontend dist
COPY --from=frontend-builder /app/web/dist ./internal/static/dist

# Build Go binary
RUN VERSION_VAL=$(cat /build-info/version.txt) && \
    BUILD_TIME_VAL=$(cat /build-info/build_time.txt) && \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -ldflags="-s -w -X baihu/internal/constant.Version=${VERSION_VAL} -X 'baihu/internal/constant.BuildTime=${BUILD_TIME_VAL}'" \
    -o baihu .

# ================================
# Stage 3: Build Agent (all platforms)
# ================================
FROM --platform=$BUILDPLATFORM golang:1.24-alpine AS agent-builder

WORKDIR /app

# Copy build info
COPY --from=build-info /build-info /build-info

# Copy agent source and config example
COPY agent/ ./agent/

# Download dependencies
WORKDIR /app/agent
RUN go env -w GOPROXY=https://goproxy.cn,direct && go mod download

# Build agent for all platforms and package as tar.gz (parallel build)
RUN VERSION_VAL=$(cat /build-info/version.txt) && \
    BUILD_TIME_VAL=$(cat /build-info/build_time.txt) && \
    LDFLAGS="-s -w -X 'main.Version=${VERSION_VAL}' -X 'main.BuildTime=${BUILD_TIME_VAL}'" && \
    mkdir -p /opt/agent && \
    echo "${VERSION_VAL}" > /opt/agent/version.txt && \
    # 并行构建所有平台
    (CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="${LDFLAGS}" -o baihu-agent-linux-amd64 . &) && \
    (CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="${LDFLAGS}" -o baihu-agent-linux-arm64 . &) && \
    (CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags="${LDFLAGS}" -o baihu-agent-darwin-amd64 . &) && \
    (CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags="${LDFLAGS}" -o baihu-agent-darwin-arm64 . &) && \
    # (CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags="${LDFLAGS}" -o baihu-agent-windows-amd64.exe . &) && \
    wait && \
    # 打包各平台
    for platform in linux-amd64 linux-arm64 darwin-amd64 darwin-arm64; do \
        mv baihu-agent-${platform} baihu-agent && \
        tar -czvf /opt/agent/baihu-agent-${platform}.tar.gz baihu-agent config.example.ini && \
        rm baihu-agent; \
    done && \
    # Windows 单独处理（exe 后缀）
    # mv baihu-agent-windows-amd64.exe baihu-agent.exe && \
    # tar -czvf /opt/agent/baihu-agent-windows-amd64.tar.gz baihu-agent.exe config.example.ini && \
    # rm baihu-agent.exe && \
    echo "Agent build completed for all platforms"
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
