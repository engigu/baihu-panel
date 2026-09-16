# 白虎面板应用规范设计 (App Specification)

## 概述

传统的自动化调度平台（如青龙等）长期采用**“仓库 + 裸脚本扫描”**模式：直接拉取一个 Git 仓库，靠正则硬扫脚本头部的注释（如 `// cron "0 0 * * *"`）来生成任务。

这种旧模式存在诸多严重弊端：
- **规则与代码死锁**：上游脚本作者如果不更新、不 Push 代码，调度规则、重试次数和执行参数就无法更新；
- **配置极其繁琐**：用户需要到处翻找 README 猜测环境变量名，错填漏填频繁导致任务报错；
- **依赖黑盒报错**：缺少明确的前置环境契约，经常在脚本运行到一半时报 `ModuleNotFoundError`；
- **任务泥沙俱下**：一个仓库几十个脚本一股脑全塞进任务列表，缺乏场景化（Scenario）选择能力。

为此，白虎面板正式提出**「应用模式规范 (Baihu App Specification)」**，将传统脚本堆砌升级为现代化的**声明式应用工程**。

---

## 一、 标准文件组织

白虎应用描述文件统一命名为 `baihu-app.yaml`。

```text
my-app/
├── baihu-app.yaml          # [核心] 应用主描述文件 (App Manifest)
├── templates/              # [可选] 默认配置文件模板
├── scripts/                # [可选] 内置本地脚本（若不使用远程仓库）
└── README.md               # [可选] 应用说明文档
```

> **纯编排模式**：应用甚至可以是一个无代码仓库的独立 YAML 文件（托管于 Gist、CDN 或订阅源），仅负责定义规则、环境、场景与上游源码的映射关系。

---

## 二、 完整 Manifest 结构详解

```yaml
# ==============================================================================
# 白虎面板应用规范定义文件 (Baihu Application Specification)
# 文件名标准: baihu-app.yaml
# ==============================================================================

spec_version: "v1"                    # 规范版本号
id: "bilibili-assistant"              # 全局唯一标识符（字母、数字、中划线）
name: "B站全自动化助手"                # 应用显示名称
version: "1.2.0"                      # 应用语义化版本号
author: "Baihu Community"             # 作者或维护组织
category: "福利签到"                   # 分类标签
description: "每日经验获取、大会员权益礼包领取、天选抽奖等一站式自动化任务编排"
icon: "https://assets.example.com/icons/bilibili.png"
homepage: "https://github.com/example/bilibili-assistant"

# ------------------------------------------------------------------------------
# 1. 脚本代码源列表 (Sources) —— 支持定义多个源，100% 复用 baihu reposync 参数规范
# ------------------------------------------------------------------------------
# 支持配置多个独立的代码源（例如：主业务仓库 + 外部公共工具库 + 单文件补丁直链）
# 单源场景下亦兼容单个 source: { ... } 对象
sources:
  - id: "main"                        # [必填] 源唯一标识符（在 tasks 中通过 source: "main" 关联）
    source_type: "git"                # 对应 --source-type: git (Git仓库) 或 url (单文件下载)
    source_url: "https://github.com/upstream-author/bili-scripts.git" # 对应 --source-url
    branch: "main"                    # 对应 --branch: 分支名（留空自动探测）
    path: ""                          # 对应 --path: 稀疏检出子目录或单文件路径（留空全量）
    single_file: false                # 对应 --single-file: 是否单文件直接下载模式
    proxy: "ghproxy"                  # 对应 --proxy: none / ghproxy / mirror / custom
    proxy_url: ""                     # 对应 --proxy-url: 自定义代理前缀
    auth_token: ""                    # 对应 --auth-token: 私有仓库访问 Token
    http_proxy: ""                    # 对应 --http-proxy: HTTP/SOCKS 代理
    whitelist_paths: ""               # 对应 --whitelist-paths: 白名单保留路径
    blacklist: ""                     # 对应 --blacklist: 黑名单剔除关键词
    target_path: "main"               # 存储子目录（默认 sources/{id}）

  - id: "extra_helper"                # [示例] 第二个源：引入外部独立的单文件公共助手库
    source_type: "url"
    source_url: "https://raw.githubusercontent.com/example/tools/main/bili_helper.py"
    single_file: true
    proxy: "ghproxy"
    target_path: "extra_helper"

# ------------------------------------------------------------------------------
# 2. 原生 Shell 环境与依赖编排 (Setup) —— 拒绝死板配置，原生 Shell 极速执行
# ------------------------------------------------------------------------------
setup:
  # [可选] 依赖快速探测命令：退出码为 0 表示已满足，直接秒级跳过安装流程
  check: |
    python3 -c "import requests, cryptography" 2>/dev/null && node -e "require('axios'); require('crypto-js')" 2>/dev/null

  # [必填] 原生 Shell 安装脚本（支持自由指定国内源、镜像加速、系统级软件包等）
  install: |
    echo ">> [1/2] 正在安装 Python 运行时依赖..."
    pip install -q --no-cache-dir requests cryptography -i https://pypi.tuna.tsinghua.edu.cn/simple

    echo ">> [2/2] 正在安装 Node.js 运行时依赖..."
    npm install -g axios crypto-js --registry=https://registry.npmmirror.com

    echo ">> 依赖环境准备就绪！"

  # [可选] 后置初始化命令：安装/构建完成后自动执行的 Shell 脚本
  post_install: |
    echo ">> 正在初始化基础配置文件..."
    cp -n "{{app_dir}}/main/config.example.json" "{{app_dir}}/main/config.json" 2>/dev/null || true

  # [可选] 应用卸载时的清理命令（防止垃圾文件与孤儿包残留）
  uninstall: |
    pip uninstall -y requests cryptography 2>/dev/null || true

# ------------------------------------------------------------------------------
# 3. 环境变量声明契约 (Env Schema) —— 驱动前端自动化渲染交互式表单
# ------------------------------------------------------------------------------
env_schema:
  - key: "BILI_COOKIE"
    label: "账号凭证 (Cookie)"
    type: "secret"                    # string / secret / number / boolean / select
    required: true
    description: "登录 bilibili.com 后在控制台获取包含 SESSDATA 与 bili_jct 的 Cookie"
    placeholder: "SESSDATA=xxxx; bili_jct=yyyy;"

  - key: "COIN_NUM"
    label: "每日投币数量"
    type: "select"
    required: false
    default: "5"
    options:
      - label: "不投币 (0枚)"
        value: "0"
      - label: "保底 (1枚)"
        value: "1"
      - label: "拿满经验 (5枚)"
        value: "5"

  - key: "AUTO_SILVER_TO_COIN"
    label: "自动银瓜子兑硬币"
    type: "boolean"
    default: true
    description: "每日 0点 自动将多余的直播间银瓜子兑换为硬币"

# ------------------------------------------------------------------------------
# 4. 任务生成与映射规则 (Sync Rules) —— 规则独立热更新，无须上游仓库 Git Push
# ------------------------------------------------------------------------------
sync_rules:
  # 全局任务默认参数
  defaults:
    timeout: 15                       # 默认超时（分钟）
    retry_count: 2                    # 失败重试次数
    retry_interval: 10                # 失败重试间隔（秒）
    work_dir: "{app_dir}"             # 脚本运行时工作目录占位符
    language: "node"                  # 默认执行环境

  # 任务清单定义（将代码源中的物理脚本重命名、重配 Cron，不受上游注释限制）
  tasks:
    - id: "daily_task"
      name: "每日基础经验任务"
      source: "main"                  # [可选] 关联所属源 id（多源时指定，留空默认首个源）
      file: "bili_daily.js"           # 对应该源目录下的脚本文件
      default_cron: "0 0 9 * * *"     # 默认 Cron 表达式
      enabled: true

    - id: "manga_task"
      name: "大会员漫画权益领取"
      source: "main"
      file: "bili_manga.js"
      default_cron: "0 30 10 * * *"
      enabled: true

    - id: "silver_coin"
      name: "银瓜子兑换硬币"
      source: "extra_helper"          # [示例] 关联第二个代码源 (extra_helper)
      file: "bili_helper.py"
      language: "python"              # 针对具体任务覆盖语言环境
      default_cron: "0 5 0 * * *"
      enabled: false

    - id: "live_lottery"
      name: "天选时刻高频巡检抽奖"
      source: "main"
      file: "live_lottery.js"
      default_cron: "*/15 * * * *"
      enabled: false

# ------------------------------------------------------------------------------
# 5. 使用场景模板 (Scenarios) —— 赋能用户一键选配，杜绝盲目生成冗余任务
# ------------------------------------------------------------------------------
scenarios:
  - id: "minimal"
    name: "佛系保级模式"
    description: "仅执行每日登录与签到，耗时极短且完全防风控，适合只需保级的账号"
    default: true
    task_presets:
      daily_task:
        enabled: true
      manga_task:
        enabled: false
      silver_coin:
        enabled: false
      live_lottery:
        enabled: false

  - id: "standard"
    name: "标准日常满收益模式"
    description: "开启投币、大会员礼包与漫画权益领取，最大化每日经验与收益"
    task_presets:
      daily_task:
        enabled: true
      manga_task:
        enabled: true
      silver_coin:
        enabled: true
      live_lottery:
        enabled: false

  - id: "hardcore"
    name: "全天候极客模式"
    description: "开启全部功能，包括 15 分钟一次的直播间自动巡检抽奖"
    task_presets:
      daily_task:
        enabled: true
      manga_task:
        enabled: true
      silver_coin:
        enabled: true
      live_lottery:
        enabled: true
        cron: "*/20 * * * *"          # 场景模板还可覆盖特定 Cron
```

---

## 三、 核心架构机制

### 1. 原生 Shell 依赖驱动 (Native Shell Setup)
放弃限制重重的键值对包清单声明，直接采用原生 Shell 脚本：
- **极致自由**：自由指定镜像源（清华源、阿里源）、私有仓库认证、系统级包管理命令（`apk`、`apt`）；
- **秒级跳过**：提供 `check` 命令进行先验探活，已安装过的依赖可在 0.1 秒内跳过，极大减轻同步开销；
- **实时输出**：面板调用执行引擎直接将安装过程以 WebSocket 形式流式回显在用户终端中，排错清晰透明。

### 2. 双轨同步引擎（规则与代码解耦）
- **代码同步轨道 (Code Track)**：负责 Git Pull 拉取业务脚本；
- **规则同步轨道 (Rule Track)**：负责拉取最新的 `baihu-app.yaml`。
- **价值**：规则维护者可以在独立的订阅仓库或 Gist 中动态调整防黑号 Cron、添加新启动参数、修正任务开关。用户只需触发“更新规则”，数秒内即可热更新所有受控任务，**完全不需要等待原脚本作者 Git Push**！

### 3. 用户场景模板 (Scenario Templates)
- 安装应用时，用户不再面对“几十个不知所云的脚本”，而是直接在友好的 UI 上单选：“**佛系保级模式**” 或 “**标准满收益模式**”；
- 选定场景后，面板自动按照场景的 `task_presets` 批量开关并调整对应的任务配置。用户日后也可随时一键切换场景。
