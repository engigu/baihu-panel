#!/usr/bin/env python3
# -*- coding: utf-8 -*-

import argparse
import os
import subprocess
import sys
import shutil
import urllib.request
import urllib.error


def run(cmd, env=None, cwd=None):
    """执行命令并打印输出"""
    print(">>", " ".join(cmd))
    result = subprocess.run(
        cmd,
        cwd=cwd,
        env=env,
        stdout=sys.stdout,
        stderr=sys.stderr,
    )
    if result.returncode != 0:
        sys.exit(result.returncode)


def build_proxy_url(url, proxy_type, proxy_url):
    """构建代理 URL"""
    if not proxy_type or proxy_type == "none":
        return url
    
    proxy_base = ""
    if proxy_type == "ghproxy":
        proxy_base = "https://gh-proxy.com/"
    elif proxy_type == "mirror":
        proxy_base = "https://mirror.ghproxy.com/"
    elif proxy_type == "custom" and proxy_url:
        proxy_base = proxy_url.rstrip("/") + "/"
    
    if proxy_base and url.startswith("http"):
        return proxy_base + url
    
    return url


def sync_git_file(args, repo_url, env):
    """从 Git 仓库同步单个文件（通过 raw URL 下载）"""
    source_url = args.source_url
    file_path = args.path
    branch = args.branch or "main"
    dest = args.target_path
    
    # 构建 raw 文件 URL
    # GitHub: https://github.com/user/repo -> https://raw.githubusercontent.com/user/repo/branch/path
    # GitLab: https://gitlab.com/user/repo -> https://gitlab.com/user/repo/-/raw/branch/path
    # Gitee:  https://gitee.com/user/repo -> https://gitee.com/user/repo/raw/branch/path
    
    raw_url = None
    if "github.com" in source_url:
        # GitHub
        base = source_url.replace("github.com", "raw.githubusercontent.com").rstrip(".git")
        raw_url = f"{base}/{branch}/{file_path}"
    elif "gitlab.com" in source_url:
        # GitLab
        base = source_url.rstrip(".git")
        raw_url = f"{base}/-/raw/{branch}/{file_path}"
    elif "gitee.com" in source_url:
        # Gitee
        base = source_url.rstrip(".git")
        raw_url = f"{base}/raw/{branch}/{file_path}"
    else:
        # 通用：尝试 GitHub 风格
        base = source_url.rstrip(".git")
        raw_url = f"{base}/raw/{branch}/{file_path}"
    
    # 应用代理
    raw_url = build_proxy_url(raw_url, args.proxy, args.proxy_url)
    
    print(f"下载单文件: {raw_url}")
    print(f"目标路径: {dest}")
    
    # 确保目标目录存在
    parent_dir = os.path.dirname(dest)
    if parent_dir:
        os.makedirs(parent_dir, exist_ok=True)
    
    # 创建请求
    req = urllib.request.Request(raw_url)
    
    # 添加认证 Token
    if args.auth_token:
        req.add_header("Authorization", f"token {args.auth_token}")
    
    req.add_header("User-Agent", "Mozilla/5.0 (compatible; sync.py)")
    
    try:
        with urllib.request.urlopen(req, timeout=300) as response:
            content = response.read()
            
            with open(dest, "wb") as f:
                f.write(content)
            
            print(f"文件大小: {len(content)} 字节")
            print("同步完成")
    except urllib.error.HTTPError as e:
        print(f"下载失败, HTTP 状态码: {e.code}")
        sys.exit(1)
    except urllib.error.URLError as e:
        print(f"下载失败: {e.reason}")
        sys.exit(1)


def is_raw_file_url(url):
    """检测是否是 raw 文件 URL"""
    raw_patterns = [
        "raw.githubusercontent.com",
        "/raw/",
        "/-/raw/",
        "/blob/",
    ]
    return any(pattern in url for pattern in raw_patterns)


def get_repo_name(url):
    """从 Git URL 中提取仓库名"""
    # 去掉末尾的 .git
    url = url.rstrip("/").rstrip(".git")
    # 提取最后一部分作为仓库名
    return os.path.basename(url)


def sync_git(args):
    """Git 仓库同步"""
    env = os.environ.copy()

    # 如果 source_url 是 raw 文件 URL，自动切换到 URL 下载模式
    if is_raw_file_url(args.source_url):
        print("检测到 raw 文件 URL，自动切换到 URL 下载模式")
        sync_url(args)
        return

    # 设置 HTTP 代理
    if args.http_proxy:
        env["http_proxy"] = args.http_proxy
        env["https_proxy"] = args.http_proxy

    # 构建仓库 URL（带代理）
    repo_url = build_proxy_url(args.source_url, args.proxy, args.proxy_url)
    
    # 如果有认证 Token，将其嵌入 URL
    if args.auth_token and repo_url.startswith("https://"):
        repo_url = repo_url.replace("https://", f"https://{args.auth_token}@")

    dest = args.target_path
    branch = args.branch or "main"

    # 如果指定了 path 且是单文件模式，使用 raw URL 下载
    if args.path and args.single_file:
        sync_git_file(args, repo_url, env)
        return

    # 如果目标路径是已存在的目录且不是 git 仓库，自动追加仓库名作为子目录
    git_dir = os.path.join(dest, ".git")
    if os.path.isdir(dest) and not os.path.exists(git_dir):
        repo_name = get_repo_name(args.source_url)
        dest = os.path.join(dest, repo_name)
        print(f"目标路径自动追加仓库名: {dest}")
        git_dir = os.path.join(dest, ".git")

    # 检查目标目录是否已存在 git 仓库
    is_existing_repo = os.path.exists(git_dir)

    if is_existing_repo:
        # 已存在仓库，执行 git pull
        print(f"检测到已存在仓库，执行 git pull")
        
        # 先切换分支
        if branch:
            try:
                run(["git", "checkout", branch], cwd=dest, env=env)
            except:
                pass
        
        run(["git", "pull"], cwd=dest, env=env)
    else:
        # 新仓库，执行 git clone
        print(f"执行 git clone")
        
        # 确保父目录存在
        parent_dir = os.path.dirname(dest)
        if parent_dir:
            os.makedirs(parent_dir, exist_ok=True)

        # 如果目标目录已存在且不为空，报错提示
        if os.path.exists(dest) and os.listdir(dest):
            print(f"错误: 目标目录 '{dest}' 已存在且不为空，无法执行 git clone")
            print("提示: 请清空目标目录或指定一个新目录")
            sys.exit(1)

        # 稀疏 clone（如果指定了 path）
        if args.path:
            run([
                "git", "clone",
                "--depth", "1",
                "--filter=blob:none",
                "--no-checkout",
                "-b", branch,
                repo_url,
                dest
            ], env=env)

            run(["git", "sparse-checkout", "init", "--cone"], cwd=dest, env=env)
            run(["git", "sparse-checkout", "set", args.path], cwd=dest, env=env)
            run(["git", "checkout"], cwd=dest, env=env)
        else:
            # 普通 clone
            run([
                "git", "clone",
                "--depth", "1",
                "-b", branch,
                repo_url,
                dest
            ], env=env)

    print("同步完成")


def sync_url(args):
    """URL 文件下载"""
    # 构建下载 URL（带代理）
    download_url = build_proxy_url(args.source_url, args.proxy, args.proxy_url)
    
    print(f"下载地址: {download_url}")
    
    dest = args.target_path
    
    # 如果目标路径是目录或以 / 结尾，从 URL 中提取文件名
    if os.path.isdir(dest) or dest.endswith("/"):
        # 从 URL 中提取文件名
        url_path = args.source_url.split("?")[0]  # 去掉查询参数
        filename = os.path.basename(url_path)
        if not filename:
            filename = "downloaded_file"
        dest = os.path.join(dest, filename)
        print(f"目标文件: {dest}")
    
    # 确保目标目录存在
    parent_dir = os.path.dirname(dest)
    if parent_dir:
        os.makedirs(parent_dir, exist_ok=True)

    # 创建请求
    req = urllib.request.Request(download_url)
    
    # 添加认证 Token
    if args.auth_token:
        req.add_header("Authorization", f"token {args.auth_token}")
    
    # 添加 User-Agent
    req.add_header("User-Agent", "Mozilla/5.0 (compatible; sync.py)")

    try:
        with urllib.request.urlopen(req, timeout=300) as response:
            content = response.read()
            
            with open(dest, "wb") as f:
                f.write(content)
            
            print(f"目标路径: {dest}")
            print(f"文件大小: {len(content)} 字节")
            print("同步完成")
    except urllib.error.HTTPError as e:
        print(f"下载失败, HTTP 状态码: {e.code}")
        sys.exit(1)
    except urllib.error.URLError as e:
        print(f"下载失败: {e.reason}")
        sys.exit(1)


def main():
    parser = argparse.ArgumentParser(description="仓库/文件同步工具")

    parser.add_argument("--source-type", choices=["git", "url"], default="git",
                        help="源类型: git(Git仓库) 或 url(URL下载)")
    parser.add_argument("--source-url", required=True, 
                        help="源地址（Git仓库URL或文件URL）")
    parser.add_argument("--target-path", required=True, 
                        help="目标路径")
    parser.add_argument("--branch", default="main", 
                        help="Git 分支名（仅 git 类型有效）")
    parser.add_argument("--path", 
                        help="仅拉取指定文件或目录（仅 git 类型有效）")
    parser.add_argument("--single-file", action="store_true",
                        help="单文件模式，直接下载指定文件而非 sparse-checkout（需配合 --path 使用）")
    parser.add_argument("--proxy", choices=["none", "ghproxy", "mirror", "custom"], default="none",
                        help="代理类型")
    parser.add_argument("--proxy-url", 
                        help="自定义代理地址（仅 proxy=custom 时有效）")
    parser.add_argument("--auth-token", 
                        help="认证 Token（用于私有仓库）")
    parser.add_argument("--http-proxy", 
                        help="HTTP 代理（如 http://127.0.0.1:7890）")

    args = parser.parse_args()

    # 打印原始命令行参数
    print("参数:", " ".join(sys.argv[1:]))

    if args.source_type == "git":
        sync_git(args)
    else:
        sync_url(args)


if __name__ == "__main__":
    main()
