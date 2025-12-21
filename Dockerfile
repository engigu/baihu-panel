# Stage 1: Build frontend
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

# Stage 2: Build backend
FROM --platform=$BUILDPLATFORM golang:1.24-alpine AS backend-builder

ARG TARGETOS
ARG TARGETARCH

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go env -w GOPROXY=https://goproxy.cn,direct && go mod download

# Copy source code
COPY . .

# Copy built frontend to embed location
COPY --from=frontend-builder /app/web/dist ./internal/static/dist

# Build Go binary with cross-compilation (native speed, no QEMU)
ARG VERSION=dev
ARG BUILD_TIME
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -ldflags="-s -w -X baihu/internal/constant.Version=${VERSION} -X 'baihu/internal/constant.BuildTime=${BUILD_TIME}'" -o baihu .

# Stage 3: Final image based on Dockerfile.debian
FROM debian:bookworm-slim

ENV TZ=Asia/Shanghai
ENV CONDA_DIR=/opt/miniforge3
ENV PATH=${CONDA_DIR}/bin:$PATH

RUN sed -i 's@deb.debian.org@mirrors.tuna.tsinghua.edu.cn@g' /etc/apt/sources.list.d/debian.sources \
    && sed -i 's@security.debian.org@mirrors.tuna.tsinghua.edu.cn@g' /etc/apt/sources.list.d/debian.sources \
    && echo "${TZ}" > /etc/timezone \
    && ln -sf /usr/share/zoneinfo/${TZ} /etc/localtime \
    && apt update \
    && apt install -y tzdata git gcc curl wget vim ca-certificates procps htop e2fsprogs \
    && rm -rf /var/lib/apt/lists/* \
    && curl -sSL https://gh-proxy.com/https://github.com/conda-forge/miniforge/releases/latest/download/Miniforge3-$(uname)-$(uname -m).sh -o /tmp/miniforge.sh \
    && bash /tmp/miniforge.sh -b -p ${CONDA_DIR} \
    && rm -f /tmp/miniforge.sh \
    && "${CONDA_DIR}"/bin/conda config --set show_channel_urls yes \
    && "${CONDA_DIR}"/bin/conda config --set channel_priority strict \
    && "${CONDA_DIR}"/bin/conda init \
    && "${CONDA_DIR}"/bin/conda clean -afy

WORKDIR /app

# Copy binary from builder
COPY --from=backend-builder /app/baihu .

# Copy config files
COPY --from=backend-builder /app/configs ./configs

# Copy entrypoint script
COPY docker-entrypoint.sh .
RUN chmod +x docker-entrypoint.sh

# Expose port
EXPOSE 8052

# Run with entrypoint
CMD ["./docker-entrypoint.sh"]
