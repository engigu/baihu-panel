#!/bin/sh
set -e

PYTHON_VENV_DIR="/app/envs/python"
NODE_ENV_DIR="/app/envs/node"

echo "[entrypoint] Starting environment initialization..."

# ============================
# 创建基础目录
# ============================
mkdir -p \
  /app/data \
  /app/data/scripts \
  /app/configs \
  /app/envs \
  "$NODE_ENV_DIR"

# ============================
# Python 虚拟环境
# ============================
if [ ! -x "$PYTHON_VENV_DIR/bin/python" ]; then
    echo "[entrypoint] Creating Python virtual environment..."
    python3 -m venv "$PYTHON_VENV_DIR"
    "$PYTHON_VENV_DIR/bin/pip" install --upgrade pip setuptools wheel
    "$PYTHON_VENV_DIR/bin/pip" config set global.index-url https://pypi.tuna.tsinghua.edu.cn/simple
else
    echo "[entrypoint] Python venv exists"
fi

# ============================
# Node 环境（npm prefix 方案，核心）
# ============================
echo "[entrypoint] Initializing Node npm prefix..."

# npm 全局安装目录 → /app/envs/node
npm config set prefix "$NODE_ENV_DIR"

# npm 行为优化（防内存暴涨）
npm config set registry https://registry.npmmirror.com
npm config set audit false
npm config set fund false
npm config set progress false
npm config set maxsockets 2

# Node 内存限制（非常关键）
export NODE_OPTIONS="--max-old-space-size=256"

# ============================
# 环境变量注入（等价 activate）
# ============================
export PATH="$PYTHON_VENV_DIR/bin:$NODE_ENV_DIR/bin:$PATH"
export NODE_PATH="$NODE_ENV_DIR/lib/node_modules"

# ============================
# 打印确认（强烈建议保留）
# ============================
echo "[entrypoint] python: $(which python)"
echo "[entrypoint] pip: $(which pip)"
echo "[entrypoint] node: $(which node)"
echo "[entrypoint] npm prefix: $(npm config get prefix)"
echo "[entrypoint] npm root: $(npm root -g)"

# ============================
# 启动应用
# ============================
cd /app
exec ./baihu