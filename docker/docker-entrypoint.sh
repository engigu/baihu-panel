#!/bin/sh
set -e
export LANG=C.UTF-8
export LC_ALL=C.UTF-8

export MISE_HIDE_UPDATE_WARNING=1

# 日志输出格式
COLOR_PREFIX="\033[1;36m[Entrypoint]\033[0m"
log() {
  printf "${COLOR_PREFIX} %s\n" "$1"
}

MISE_DIR="/app/envs/mise"

log "Starting environment initialization..."

# ============================
# 创建基础目录
# ============================
mkdir -p \
  /app/data \
  /app/data/scripts \
  /app/configs \
  /app/envs

if [ -d "/app/example" ]; then
  mkdir -p /app/data/scripts/example
  rsync -a --ignore-existing /app/example/ /app/data/scripts/example/ || true
  log "Example scripts synced to /app/data/scripts/example"
else
  log "No example directory found, skipping example sync"
fi

# ============================
# Mise 环境初始化 (仅容器重建或镜像版本更新时同步)
# ============================
mkdir -p "$MISE_DIR"
if [ -d "/opt/mise-base" ]; then
  CURRENT_VER=$(cat /build-info/version.txt 2>/dev/null || echo "")
  INSTALLED_VER=$(cat "$MISE_DIR/.installed_version" 2>/dev/null || echo "")

  NEED_SYNC=0
  if [ ! -f "/tmp/.mise_container_synced" ]; then
    NEED_SYNC=1
  elif [ -n "$CURRENT_VER" ] && [ "$CURRENT_VER" != "$INSTALLED_VER" ]; then
    NEED_SYNC=1
  elif [ ! -f "$MISE_DIR/config.toml" ]; then
    NEED_SYNC=1
  fi

  if [ "$NEED_SYNC" -eq 1 ]; then
    log "Syncing mise environment from base (rebuild or image update)..."
    rsync -a --ignore-existing /opt/mise-base/ "$MISE_DIR/" || true
    touch /tmp/.mise_container_synced
    [ -n "$CURRENT_VER" ] && echo "$CURRENT_VER" > "$MISE_DIR/.installed_version"
    log "Mise environment synced"
  else
    log "Mise environment already synced for this container, skipping rsync"
  fi
else
  log "No base mise environment found, skipping sync"
fi

# ============================
# 环境变量注入
# ============================
export MISE_DATA_DIR="$MISE_DIR"
export MISE_CONFIG_DIR="$MISE_DIR"
export PATH="$MISE_DIR/shims:$MISE_DIR/bin:$PATH"

log "Mise PATH configured, verifying runtimes..."

# 默认启用 Python 镜像源
export PIP_INDEX_URL=${PIP_INDEX_URL:-https://pypi.org/simple}

# Node 内存限制
export NODE_OPTIONS="--max-old-space-size=256"
export PYTHONPATH=/app/data/scripts:$PYTHONPATH

# ============================
# 打印确认 (增加超时防护，防止这里卡死)
# ============================
log "Checking mise..."
log "  - mise: $(mise --version 2>/dev/null | head -n 1 || echo "not found")"

log "Checking python..."
log "  - python: $(python --version 2>&1 | head -n 1 || echo "not found")"

log "Checking node..."
log "  - node: $(node --version 2>&1 | head -n 1 || echo "not found")"

log "Checking npm..."
log "  - npm: $(npm --version 2>&1 | head -n 1 || echo "not found")"

# ============================
# 将 baihu 注册到全局命令并配置 Tab 自动补全
# ============================
ln -sf /app/baihu /usr/local/bin/baihu

for rcfile in /etc/bash.bashrc /etc/bashrc /root/.bashrc; do
  if [ -f "$rcfile" ] || [ "$rcfile" = "/root/.bashrc" ]; then
    if ! grep -q "baihu completion" "$rcfile" 2>/dev/null; then
      echo 'eval "$(baihu completion bash 2>/dev/null)"' >> "$rcfile" 2>/dev/null || true
    fi
  fi
done

# ============================
# 释放启动读盘产生的 Page Cache
# ============================
baihu dropcache /opt/mise-base "$MISE_DIR" 2>/dev/null || true

# ============================
# 启动应用
# ============================
printf "\n\033[1;32m>>> Environment setup complete. Starting Baihu Server...\033[0m\n\n"

cd /app
exec baihu server
