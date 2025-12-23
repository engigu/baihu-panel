#!/bin/sh

PYTHON_VENV_DIR="/app/envs/python"

# ============================
# 创建必要目录
# ============================
mkdir -p /app/data /app/data/scripts /app/configs /app/envs

# ============================
# 创建 Python 虚拟环境（如果不存在）
# ============================
if [ ! -d "$PYTHON_VENV_DIR" ]; then
    echo "Creating Python virtual environment..."
    python3 -m venv "$PYTHON_VENV_DIR"
    "$PYTHON_VENV_DIR/bin/pip" config set global.index-url https://pypi.tuna.tsinghua.edu.cn/simple
    "$PYTHON_VENV_DIR/bin/pip" install --upgrade pip
    echo "Python virtual environment created at $PYTHON_VENV_DIR"
else
    echo "Python virtual environment already exists at $PYTHON_VENV_DIR"
fi

# ============================
# 激活 Python 虚拟环境
# ============================
export PATH="$PYTHON_VENV_DIR/bin:$PATH"

# ============================
# 启动应用
# ============================
exec ./baihu
