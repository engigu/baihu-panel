ARG BASE_TAG=base
# ================================
# Stage 1: Build frontend
# ================================
FROM --platform=$BUILDPLATFORM node:20-alpine AS frontend-builder

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
FROM --platform=$BUILDPLATFORM golang:1.25 AS backend-builder

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

# Copy frontend dist (needed for embedding)
COPY --from=frontend-builder /app/web/dist ./internal/static/dist

# Generate Swagger
RUN go run github.com/swaggo/swag/cmd/swag@latest init -g main.go -o ./openapi_docs

# Build Go binary
RUN VERSION_VAL=$(cat /build-info/version.txt) && \
    BUILD_TIME_VAL=$(cat /build-info/build_time.txt) && \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -tags web -ldflags="-s -w -X github.com/engigu/baihu-panel/internal/constant.Version=${VERSION_VAL} -X 'github.com/engigu/baihu-panel/internal/constant.BuildTime=${BUILD_TIME_VAL}'" \
    -o baihu .

# ================================
# Stage 3: Build Agent (all platforms)
# ================================
FROM --platform=$BUILDPLATFORM golang:1.25 AS agent-builder


WORKDIR /app

# Copy build info
COPY --from=build-info /build-info /build-info

# Go mod files
COPY go.mod go.sum ./
RUN go env -w GOPROXY=https://goproxy.cn,direct && go mod download

# Copy source needed for agent
COPY internal/ ./internal/
COPY agent/ ./agent/

# Build agent for all platforms and package as tar.gz
WORKDIR /app/agent

RUN VERSION_VAL=$(cat /build-info/version.txt) && \
    BUILD_TIME_VAL=$(cat /build-info/build_time.txt) && \
    LDFLAGS="-s -w -X 'main.Version=${VERSION_VAL}' -X 'main.BuildTime=${BUILD_TIME_VAL}'" && \
    mkdir -p /opt/agent && \
    echo "${VERSION_VAL}" > /opt/agent/version.txt && \
    # Helper to build and compress
    build_agent() { \
        local os=$1; local arch=$2; local suffix=$3; \
        CGO_ENABLED=0 GOOS=$os GOARCH=$arch go build -ldflags="${LDFLAGS}" -o baihu-agent$suffix . && \
        tar -czvf /opt/agent/baihu-agent-$os-$arch.tar.gz baihu-agent$suffix config.example.ini && \
        rm baihu-agent$suffix; \
    } && \
    build_agent linux amd64 "" && \
    build_agent linux arm64 "" && \
    build_agent darwin amd64 "" && \
    build_agent darwin arm64 "" && \
    echo "Agent build completed for all platforms"
# ================================
# Stage 4: Final image
# ================================
ARG BASE_TAG
FROM ghcr.io/engigu/baihu:${BASE_TAG}

ENV TZ=Asia/Shanghai
ENV LANG=C.UTF-8
ENV LC_ALL=C.UTF-8
ENV MISE_DATA_DIR=/app/envs/mise
ENV MISE_CONFIG_DIR=/app/envs/mise
ENV PATH="/app/envs/mise/shims:/app/envs/mise/bin:$PATH"

# # 安装必要系统工具 + Node + Python
# RUN sed -i 's@deb.debian.org@mirrors.tuna.tsinghua.edu.cn@g' /etc/apt/sources.list.d/debian.sources \
#     && sed -i 's@security.debian.org@mirrors.tuna.tsinghua.edu.cn@g' /etc/apt/sources.list.d/debian.sources \
#     && echo "${TZ}" > /etc/timezone \
#     && ln -sf /usr/share/zoneinfo/${TZ} /etc/localtime \
#     && apt-get update \
#     && apt-get install -y --no-install-recommends \
#          tzdata git gcc curl wget vim nodejs htop npm python3 python3-venv python3-pip \
#     && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy Go binary
COPY --from=backend-builder /app/baihu .

# Copy configs and entrypoint
COPY --from=backend-builder /app/configs ./configs
COPY docker/docker-entrypoint.sh .



# Copy agent binaries to /opt/agent
COPY --from=agent-builder /opt/agent /opt/agent

RUN chmod +x docker-entrypoint.sh \
    && echo "set encoding=utf-8" >> /etc/vim/vimrc \
    && ln -sf /app/baihu /usr/local/bin/baihu

EXPOSE 8052

ENTRYPOINT ["./docker-entrypoint.sh"]
