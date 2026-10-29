#!/bin/sh
set -e

# ============================================================
# 白虎面板 (Baihu Panel) 容器入口公共函数库 (Entrypoint Common)
# 规范严格参考 build-go.sh 日志样式与格式化对齐 (紧凑精简版)
# ============================================================

# ANSI 颜色与文本样式
C_RESET="\033[0m"
C_BOLD="\033[1m"
C_CYAN="\033[1;36m"
C_GREEN="\033[1;32m"
C_YELLOW="\033[1;33m"
C_BLUE="\033[1;34m"

# 标准日志函数 (紧凑输出，无多余空行)
log_info() {
  printf "${C_CYAN}[Entrypoint]${C_RESET} %s\n" "$*"
}

log_step() {
  printf "${C_BLUE}==>${C_RESET} ${C_BOLD}%s${C_RESET}\n" "$*"
}

log_success() {
  printf "${C_GREEN}✔ %s${C_RESET}\n" "$*"
}

log_warn() {
  printf "${C_YELLOW}⚠ %s${C_RESET}\n" "$*"
}

# 兼容原有的 log 函数
log() {
  log_info "$*"
}

# ============================================================
# 1. 版本与系统信息提取与环境横幅 (Banner)
# ============================================================
init_panel_version() {
  PANEL_VER=$(cat /build-info/version.txt 2>/dev/null || true)
  BUILD_TIME=$(cat /build-info/build_time.txt 2>/dev/null || true)

  if [ -z "$PANEL_VER" ] && [ -x /app/baihu ]; then
    PANEL_VER=$(/app/baihu version 2>/dev/null | awk '{print $2}' || true)
  fi
  if [ -z "$BUILD_TIME" ] && [ -x /app/baihu ]; then
    BUILD_TIME=$(/app/baihu version 2>/dev/null | awk -F'Build time: ' '{print $2}' | tr -d ')' || true)
  fi

  PANEL_VER="${PANEL_VER:-unknown}"
  BUILD_TIME="${BUILD_TIME:-unknown}"

  _os="$(uname -s 2>/dev/null | tr '[:upper:]' '[:lower:]')"
  _raw_arch="$(uname -m 2>/dev/null || echo 'unknown')"
  case "$_raw_arch" in
    x86_64)  _arch="amd64" ;;
    aarch64) _arch="arm64" ;;
    armv7l)  _arch="arm" ;;
    *)       _arch="$_raw_arch" ;;
  esac
  SYS_ARCH="${_os:-linux}/${_arch}"
}

show_banner() {
  _mode="${1:-标准版 (Standard)}"
  init_panel_version

  echo "============================================================"
  echo "           白虎面板 (Baihu Panel) 容器运行环境"
  echo "============================================================"
  log_info "容器运行模式 : ${_mode}"
  log_info "面板程序版本 : ${PANEL_VER}"
  log_info "构建时间戳   : ${BUILD_TIME}"
  log_info "系统平台架构 : ${SYS_ARCH}"
  echo "============================================================"
}

# ============================================================
# 2. 初始化数据与脚本工作目录
# ============================================================
init_directories() {
  log_step "初始化基础工作目录"
  log_info "运行数据路径 : /app/data, /app/configs, /app/envs"

  mkdir -p \
    /app/data \
    /app/data/scripts \
    /app/configs \
    /app/envs

  if [ -d "/app/example" ]; then
    mkdir -p /app/data/scripts/example
    rsync -a --ignore-existing /app/example/ /app/data/scripts/example/ || true
    log_success "基础工作目录准备完毕 -> /app/data (已同步示例脚本: scripts/example)"
  else
    log_success "基础工作目录准备完毕 -> /app/data"
  fi
}

# ============================================================
# 3. 检查并同步 Mise 基础运行底座
# ============================================================
sync_mise_environment() {
  _base_dir="${1:-/opt/mise-base}"
  _chown_opt="${2:-}"
  _target_dir="${3:-/app/envs/mise}"

  log_step "检查并同步 Mise 运行底座"
  log_info "母盘底座映射 : ${_base_dir} -> ${_target_dir}"
  mkdir -p "$_target_dir"

  if [ -d "$_base_dir" ]; then
    _curr_ver="${PANEL_VER:-$(cat /build-info/version.txt 2>/dev/null || echo '')}"
    _installed_ver=$(cat "${_target_dir}/.installed_version" 2>/dev/null || echo "")

    _need_sync=0
    if [ ! -f "/tmp/.mise_container_synced" ]; then
      _need_sync=1
    elif [ -n "$_curr_ver" ] && [ "$_curr_ver" != "unknown" ] && [ "$_curr_ver" != "$_installed_ver" ]; then
      _need_sync=1
    elif [ ! -f "${_target_dir}/config.toml" ]; then
      _need_sync=1
    fi

    if [ "$_need_sync" -eq 1 ]; then
      log_info "正在从母盘同步运行环境 (全新部署或镜像升级)..."
      if [ -n "$_chown_opt" ]; then
        rsync -a --chown="$_chown_opt" --ignore-existing "${_base_dir}/" "${_target_dir}/" || true
      else
        rsync -a --ignore-existing "${_base_dir}/" "${_target_dir}/" || true
      fi
      touch /tmp/.mise_container_synced
      [ -n "$_curr_ver" ] && echo "$_curr_ver" > "${_target_dir}/.installed_version"
      log_success "Mise 基础运行环境同步完毕 -> ${_target_dir} (版本: ${_curr_ver:-最新})"
    else
      log_success "Mise 基础运行环境已是最新 -> ${_target_dir} (跳过同步)"
    fi
  else
    log_info "未检测到底座母盘 (${_base_dir})，跳过同步"
  fi
}

# ============================================================
# 4. 配置运行时环境与系统 PATH
# ============================================================
setup_runtime_env() {
  _mise_dir="${1:-/app/envs/mise}"

  log_step "配置运行时环境与系统 PATH"
  export MISE_DATA_DIR="$_mise_dir"
  export MISE_CONFIG_DIR="$_mise_dir"
  export PATH="${_mise_dir}/shims:${_mise_dir}/bin:$PATH"

  # 默认启用 Python 镜像源
  export PIP_INDEX_URL=${PIP_INDEX_URL:-https://pypi.org/simple}

  # Node 内存限制与全局脚本模块路径
  export NODE_OPTIONS="--max-old-space-size=256"
  export PYTHONPATH=/app/data/scripts:$PYTHONPATH

  log_info "运行时 Shims : ${PATH%%:*}"
  log_info "Python 镜像源: ${PIP_INDEX_URL}"
  log_success "运行时环境变量与系统 PATH 配置完毕"
}

# ============================================================
# 5. 验证预装语言与运行时环境
# ============================================================
verify_runtimes() {
  log_step "验证核心运行时组件"

  _mise_ver="未安装"
  if [ -x "$(command -v mise)" ]; then
    _mise_ver="$(mise --version 2>/dev/null | head -n 1 || echo '未检测到版本')"
  fi

  _python_ver="未安装"
  if [ -x "$(command -v python)" ]; then
    _python_ver="$(python --version 2>&1 | head -n 1 || echo '未检测到版本')"
  fi

  _node_ver="未安装"
  if [ -x "$(command -v node)" ]; then
    _node_ver="$(node --version 2>&1 | head -n 1 || echo '未检测到版本')"
  fi

  _npm_ver="未安装"
  if [ -x "$(command -v npm)" ]; then
    _npm_ver="$(npm --version 2>&1 | head -n 1 || echo '未检测到版本')"
  fi

  log_info "Mise 版本   : ${_mise_ver}"
  log_info "Python 环境 : ${_python_ver}"
  log_info "Node.js 环境: ${_node_ver}"
  log_info "NPM 包管理器: ${_npm_ver}"
  log_success "运行时核心组件自检完毕"
}

# ============================================================
# 6. 注册全局 CLI 命令与自动补全
# ============================================================
setup_cli_and_completion() {
  _extra_rc="${1:-}"

  log_step "注册全局 CLI 命令与自动补全"
  if [ -x "/app/baihu" ]; then
    ln -sf /app/baihu /usr/local/bin/baihu
    log_info "全局命令软链 : /app/baihu -> /usr/local/bin/baihu"
  fi

  _rc_targets="/etc/bash.bashrc /etc/bashrc /root/.bashrc"
  [ -n "$_extra_rc" ] && _rc_targets="${_rc_targets} ${_extra_rc}"

  for _rcfile in $_rc_targets; do
    if [ -f "$_rcfile" ] || [ -d "$(dirname "$_rcfile" 2>/dev/null)" ]; then
      if ! grep -q "baihu completion" "$_rcfile" 2>/dev/null; then
        echo 'eval "$(baihu completion bash 2>/dev/null)"' >> "$_rcfile" 2>/dev/null || true
      fi
    fi
  done
  log_success "全局 CLI 命令及自动补全配置完毕"
}

# ============================================================
# 7. 检查并初始化内建助手库 (Built-in SDK)
# ============================================================
init_builtin_packages() {
  log_step "检查并初始化内建助手库"

  _flag_file="/tmp/.baihu_builtin_installed"
  _lock_file="/tmp/.baihu_builtin_installing"
  _curr_ver="${PANEL_VER:-$(cat /build-info/version.txt 2>/dev/null || echo '')}"

  if [ -f "$_flag_file" ]; then
    _installed_ver=$(cat "$_flag_file" 2>/dev/null || echo "")
    if [ -z "$_curr_ver" ] || [ "$_curr_ver" = "unknown" ] || [ "$_curr_ver" = "$_installed_ver" ]; then
      log_success "内建助手库已就绪 -> 跳过安装 (命中临时标记)"
      return 0
    fi
  fi

  if [ -f "$_lock_file" ]; then
    log_info "检测到内建助手库正在后台安装中，跳过重复触发"
    return 0
  fi

  _builtin_dir=""
  if [ -d "/www/builtin" ]; then
    _builtin_dir="/www/builtin"
  elif [ -d "/app/builtin" ]; then
    _builtin_dir="/app/builtin"
  fi

  _baihu_bin=""
  if [ -x "/app/baihu" ]; then
    _baihu_bin="/app/baihu"
  elif [ -x "/usr/local/bin/baihu" ]; then
    _baihu_bin="/usr/local/bin/baihu"
  fi

  if [ -n "$_baihu_bin" ] && [ -n "$_builtin_dir" ]; then
    log_info "派发后台异步安装内建助手库任务 (主服务无须等待启动)..."
    touch "$_lock_file"
    (
      if "$_baihu_bin" builtininstall >/tmp/baihu_builtininstall.log 2>&1; then
        echo "${_curr_ver:-installed}" > "$_flag_file"
      fi
      rm -f "$_lock_file"
    ) &
    log_success "内建助手库异步初始化已派发 (日志: /tmp/baihu_builtininstall.log)"
  else
    log_info "未检测到内建库目录或 CLI 工具，跳过自动安装"
  fi
}

# ============================================================
# 8. 启动主服务
# ============================================================
start_baihu_server() {
  log_step "启动白虎面板服务"
  log_info "服务工作目录 : /app"
  log_success "容器初始化全部就绪，正在启动服务进程 -> tini -- baihu server"

  cd /app
  # tini 作为 1 号进程回收孤儿进程，避免僵尸进程累积
  exec tini -- baihu server
}
