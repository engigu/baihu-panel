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

log "Starting environment initialization (Minimal Mode)..."

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
# Mise 环境初始化
# ============================
mkdir -p "$MISE_DIR"
if [ -d "/opt/mise-base" ]; then
  log "Syncing mise environment from base..."
  rsync -a --ignore-existing /opt/mise-base/ "$MISE_DIR/" || true
  log "Mise environment synced"
else
  log "No base mise environment found, skipping sync"
fi

# ============================
# 环境变量注入
# ============================
export MISE_DATA_DIR="$MISE_DIR"
export MISE_CONFIG_DIR="$MISE_DIR"
export PATH="$MISE_DIR/shims:$MISE_DIR/bin:$PATH"

# 默认启用 Python 镜像源
export PIP_INDEX_URL=${PIP_INDEX_URL:-https://pypi.org/simple}

# Node 内存限制
export NODE_OPTIONS="--max-old-space-size=256"
export PYTHONPATH=/app/data/scripts:$PYTHONPATH

# ============================
# 打印确认
# ============================
log "mise version: $(mise --version 2>/dev/null | head -n 1)"
[ -x "$(command -v python)" ] && log "python: $(python --version 2>&1 | head -n 1) at $(which python)" || log "python: not installed"
[ -x "$(command -v node)" ] && log "node: $(node --version 2>&1 | head -n 1) at $(which node)" || log "node: not installed"
[ -x "$(command -v npm)" ] && log "npm: $(npm --version 2>&1 | head -n 1) at $(which npm)" || log "npm: not installed"

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
# 启动应用
# ============================
printf "\n\033[1;32m>>> Environment setup complete. Starting Baihu Server...\033[0m\n\n"

cd /app
exec baihu server
