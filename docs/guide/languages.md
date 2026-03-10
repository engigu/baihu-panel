# 编程语言与依赖管理

白虎面板深度集成了 **Mise** 运行时管理器，这使得它具备多版本语言环境的高灵活性和隔离性。

## 脚本运行环境

白虎面板原生支持以下脚本的定时执行：
- **Python3**, **Node.js**, **Bash** (内置环境)
- 通过 **Mise** 扩展：支持几乎所有主流编程语言的动态安装与切换。

## 依赖管理支持

系统内置了高度集成的跨语言依赖管理器，支持自动化安装和管理以下语言的依赖项，并确保在容器内全局可用：

| 语言 | 包管理器 | 功能说明 |
| :--- | :--- | :--- |
| **Python** | pip | 自动使用内置虚拟环境，支持清华源 |
| **Node.js** | npm | 全局安装模式，自动配置 npmmirror 镜像 |
| **Go** | go install | 通过 `go install` 安装二进制工具 |
| **Rust** | cargo | 通过 `cargo install` 安装 Rust 依赖 |
| **Ruby** | gem | 支持 `gem install` 本地安装 |
| **Bun** | bun | 支持 `bun add -g` 全局模式 |
| **PHP** | composer | 支持 `composer global require` |
| **Deno** | deno | 支持 `deno install -g` |
| **.NET** | dotnet | 支持 `dotnet tool install -g` |
| **Elixir/Erlang** | mix | 支持 `mix archive.install` |
| **Lua** | luarocks | 通过 `luarocks` 管理 Lua 包 |
| **Nim** | nimble | 支持 `nimble install` |
| **Dart/Flutter** | pub | 支持 `pub global activate` |
| **Perl** | cpanm | 简单的 `cpanm` 安装支持 |
| **Crystal** | shards | `shards` 项目级别或工具安装 |

## 使用方法

### 1. 安装环境
进入「编程语言」页面，使用 `mise` 一键安装所需的语言及版本。

### 2. 依赖管理
在已安装列表点击「依赖管理」，输入名称（可选版本）即可自动在对应环境内完成安装。

### 3. 多版本切换
对于复杂的项目，您可以通过面板配置不同的任务版本镜像，系统基于 `mise exec` 实现了完善的环境隔离，不同版本的依赖包互不冲突。

---

## 隔离机制说明

-   白虎面板通过动态注入 `PATH` 环境和 `mise shims` 将语言环境暴露给系统。
-   每个任务在执行前都会根据任务配置自动加载对应的运行时环境变量。
-   **运行时激活**：自动将 `MISE_DATA_DIR` 等环境变量指向宿主机的持久化挂载目录，确保护持久化可用。
