#!/bin/bash

# ==============================================================================
# PHP 环境构建依赖安装脚本
# ==============================================================================
# 说明：
# 如果你需要使用 PHP 环境，由于 mise 安装某些 PHP 版本时需要从源码编译，
# 建议在面板的“任务训练”中创建一个“开机任务”，并关联此脚本，
# 以确保容器重启后依然具备编译 PHP 所需的系统依赖。
# ==============================================================================

set -e

echo "正在检测 PHP 编译依赖..."

# Debian 系统依赖包列表
PACKAGES="autoconf bison build-essential libssl-dev libxml2-dev pkg-config libsqlite3-dev zlib1g-dev libcurl4-openssl-dev libpng-dev libjpeg-dev libonig-dev libzip-dev"
CHECK_CMD="dpkg -l"
INSTALL_CMD="apt-get install -y"
UPDATE_CMD="apt-get update"

MISSING_PACKAGES=""
for pkg in $PACKAGES; do
    if ! $CHECK_CMD "$pkg" >/dev/null 2>&1; then
        MISSING_PACKAGES="$MISSING_PACKAGES $pkg"
    fi
done

if [ -z "$MISSING_PACKAGES" ]; then
    echo "所有依赖已就绪，跳过安装。"
    exit 0
fi

echo "检测到缺失依赖: $MISSING_PACKAGES"

# 执行安装
if [ -n "$UPDATE_CMD" ]; then
    $UPDATE_CMD
fi

$INSTALL_CMD $MISSING_PACKAGES

echo "PHP 编译依赖安装完成！"
