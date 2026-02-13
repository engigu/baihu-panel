#!/bin/sh
set -e
export LANG=C.UTF-8
export LC_ALL=C.UTF-8

MISE_DIR="/app/envs/mise"

echo "[entrypoint] Starting environment initialization..."

# ============================
# 创建基础目录
# ============================
mkdir -p \
  /app/data \
  /app/data/scripts \
  /app/configs \
  /app/envs

# ============================
# Mise 环境初始化
# ============================
# 始终尝试同步基础环境（以补充用户挂载卷中可能缺失的文件，如 config.toml）
echo "[entrypoint][Mise] Syncing mise environment from base..."
mkdir -p "$MISE_DIR"
# 使用 rsync 同步: -a 归档模式, --ignore-existing 不覆盖已存在文件
rsync -a --ignore-existing /opt/mise-base/ "$MISE_DIR/" || true
echo "[entrypoint][Mise] Mise environment synced"

# ============================
# 环境变量注入
# ============================
export MISE_DATA_DIR="$MISE_DIR"
export MISE_CONFIG_DIR="$MISE_DIR"
export PATH="$MISE_DIR/shims:$MISE_DIR/bin:$PATH"

# 默认启用 Python 镜像源
export PIP_INDEX_URL=https://pypi.tuna.tsinghua.edu.cn/simple

# Node 内存限制
export NODE_OPTIONS="--max-old-space-size=256"
export PYTHONPATH=/app/data/scripts:$PYTHONPATH

# ============================
# 打印确认
# ============================
echo "[entrypoint][Mise] mise version: $(mise --version)"
echo "[entrypoint][Python3] python: $(python --version) at $(which python)"
echo "[entrypoint][Nodejs] node: $(node --version) at $(which node)"
echo "[entrypoint][Nodejs] npm: $(npm --version) at $(which npm)"

# ============================
# 启动应用
# ============================
cd /app
exec ./baihu