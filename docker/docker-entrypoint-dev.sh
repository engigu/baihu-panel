#!/bin/bash
set -e
export LANG=C.UTF-8
export LC_ALL=C.UTF-8

export MISE_HIDE_UPDATE_WARNING=1

COLOR_PREFIX="\033[1;36m[Entrypoint]\033[0m"
log() {
    printf "${COLOR_PREFIX} %s\n" "$1"
}

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
        log "Cache hit: $dir owned by $target_uid. Reusing."
    else
        if [ -n "$dirty_file" ]; then
            log "Dirty cache detected in $dir (Found root-owned files like $dirty_file)."
        else
            log "User changed ($current_uid -> $target_uid). Resetting cache in $dir..."
        fi

        log "Nuking directory to ensure clean state..."

        find "$dir" -mindepth 1 -delete 2>/dev/null || rm -rf "$dir"/*

        chown "$target_uid:$target_gid" "$dir"
        log "Cache reset complete. Ownership transferred to $target_uid."
    fi
}

log "Initializing Development Environment..."

DEV_UID=${DEV_UID:-1000}
DEV_GID=${DEV_GID:-1000}

EXISTING_USER=$(getent passwd "$DEV_UID" | cut -d: -f1 | head -n 1)

if [ -n "$EXISTING_USER" ]; then
    TARGET_USER="$EXISTING_USER"
    log "UID $DEV_UID already exists as user '$TARGET_USER'. Using existing user."
else
    TARGET_USER="devuser"
    log "Creating user '$TARGET_USER' (UID: $DEV_UID, GID: $DEV_GID)..."
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

MISE_DIR="/app/envs/mise"
mkdir -p "$MISE_DIR"
log "Syncing mise environment from base..."
rsync -a --chown="$DEV_UID:$DEV_GID" --ignore-existing /opt/mise-dev/ "$MISE_DIR" || true
log "Mise environment synced"

export MISE_DATA_DIR="$MISE_DIR"
export MISE_CONFIG_DIR="$MISE_DIR"
export PATH="$MISE_DIR/shims:$MISE_DIR/bin:$PATH"

export PIP_INDEX_URL=${PIP_INDEX_URL:-https://pypi.org/simple}

log "mise version: $(mise --version 2>/dev/null | head -n 1)"
log "python: $(python --version 2>&1 | head -n 1) at $(which python)"
log "node: $(node --version 2>&1 | head -n 1) at $(which node)"
log "npm: $(npm --version 2>&1 | head -n 1) at $(which npm)"

log "Environment ready! Starting command as user '$TARGET_USER' (UID: $DEV_UID)..."
exec gosu "$DEV_UID:$DEV_GID" "$@"
