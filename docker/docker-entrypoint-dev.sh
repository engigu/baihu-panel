#!/bin/bash
set -e
export LANG=C.UTF-8
export LC_ALL=C.UTF-8
export MISE_HIDE_UPDATE_WARNING=1

# 加载公共 Entrypoint 库
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
COMMON_SH="${SCRIPT_DIR}/docker-entrypoint-common.sh"
[ ! -f "$COMMON_SH" ] && COMMON_SH="/app/docker-entrypoint-common.sh"
[ ! -f "$COMMON_SH" ] && COMMON_SH="/usr/local/bin/docker-entrypoint-common.sh"
. "$COMMON_SH"

show_banner "开发版 (Dev Mode)"

ensure_cache_ownership() {
    local dir="$1"
    local target_uid="$2"
    local target_gid="$3"
    if [ ! -d "$dir" ]; then
        mkdir -p "$dir"
        chown "$target_uid:$target_gid" "$dir"
        return
    fi
    local current_uid=$(stat -c '%u' "$dir")
    local dirty_file=$(find "$dir" -mindepth 1 ! -uid "$target_uid" -print -quit 2>/dev/null)
    if [ "$current_uid" == "$target_uid" ] && [ -z "$dirty_file" ]; then
        log_info "缓存属主状态 : ${dir} (已匹配 UID: ${target_uid})"
    else
        if [ -n "$dirty_file" ]; then
            log_warn "检测到异常属主文件: ${dirty_file}"
        else
            log_info "用户变更探测 : (${current_uid} -> ${target_uid})，重置缓存..."
        fi
        find "$dir" -mindepth 1 -delete 2>/dev/null || rm -rf "$dir"/*
        chown "$target_uid:$target_gid" "$dir"
        log_success "缓存目录重置完毕 -> ${dir} (属主: ${target_uid})"
    fi
}

log_step "初始化开发环境用户与目录权限"
DEV_UID=${DEV_UID:-1000}
DEV_GID=${DEV_GID:-1000}
EXISTING_USER=$(getent passwd "$DEV_UID" | cut -d: -f1 | head -n 1)

if [ -n "$EXISTING_USER" ]; then
    TARGET_USER="$EXISTING_USER"
    log_info "开发用户状态 : 复用已有用户 '${TARGET_USER}' (UID: $DEV_UID)"
else
    TARGET_USER="devuser"
    log_info "开发用户状态 : 创建全新用户 '${TARGET_USER}' (UID: $DEV_UID, GID: $DEV_GID)"
    groupadd -o -g "$DEV_GID" devgroup
    useradd -o -u "$DEV_UID" -g "$DEV_GID" -m -s /bin/bash "$TARGET_USER"
fi

if [ "$TARGET_USER" != "root" ]; then
    echo "$TARGET_USER ALL=(ALL) NOPASSWD: ALL" > /etc/sudoers.d/$TARGET_USER
    chmod 0440 /etc/sudoers.d/$TARGET_USER
fi

export npm_config_cache=/var/cache/npm
mkdir -p /var/cache/npm /app/web/node_modules
chown "$DEV_UID:$DEV_GID" /var/cache/npm /app/web/node_modules

if [ "$DEV_UID" != "0" ]; then
    ensure_cache_ownership "/app/envs" "$DEV_UID" "$DEV_GID"
    ensure_cache_ownership "/app/web/node_modules" "$DEV_UID" "$DEV_GID"
    ensure_cache_ownership "/var/cache/npm" "$DEV_UID" "$DEV_GID"
    ensure_cache_ownership "/go" "$DEV_UID" "$DEV_GID"
fi
log_success "开发用户与本地缓存权限配置完毕"

sync_mise_environment "/opt/mise-dev" "$DEV_UID:$DEV_GID"
setup_runtime_env
verify_runtimes
setup_cli_and_completion "/home/$TARGET_USER/.bashrc"
init_builtin_packages

log_step "启动开发环境服务进程"
log_info "执行身份用户 : ${TARGET_USER} (UID: $DEV_UID, GID: $DEV_GID)"
log_info "运行目标命令 : $*"
log_success "开发环境初始化就绪，正在启动命令 -> gosu $DEV_UID:$DEV_GID $*"

exec gosu "$DEV_UID:$DEV_GID" "$@"
