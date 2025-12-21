#!/bin/sh

# 创建必要的目录
mkdir -p /app/data /app/data/scripts /app/configs

# 如果数据库文件存在，设置不可变属性防止误删
if [ -f /app/data/ql.db ]; then
    chattr +i /app/data/ql.db 2>/dev/null || true
fi

# 如果 configs 目录存在，设置不可变属性
if [ -d /app/configs ]; then
    chattr +i /app/configs 2>/dev/null || true
fi

# 如果 scripts 目录存在，设置不可变属性
if [ -d /app/data/scripts ]; then
    chattr +i /app/data/scripts 2>/dev/null || true
fi

# 启动应用
exec ./baihu
