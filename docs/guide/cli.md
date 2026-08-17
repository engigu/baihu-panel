# 命令行工具 (CLI) 指南

白虎面板在环境内内置了同名的 `baihu` 命令行工具。无论是通过终端控制台、Docker 容器内交互，还是编写自动化运维脚本，您都可以通过 `baihu` 指令直接调用底层能力进行高效运维。

---

## 核心指令速查表

| 命令 | 描述 | 适用场景 |
| :--- | :--- | :--- |
| [`baihu server`](#1-baihu-server-服务启动) | 启动面板后台服务主进程 | 单文件/系统服务部署、后台守护运行 |
| [`baihu task`](#2-baihu-task-任务运维控制) | 任务查询、手动触发、启停控制及日志跟踪 | 命令行快速运维、CI/CD 触发、故障排查 |
| [`baihu reposync`](#3-baihu-reposync-仓库与脚本同步) | 远程 Git 仓库及文件流同步、规则过滤与任务提取 | 脚本自动化同步、仓库定时拉取 |
| [`baihu resetpwd`](#4-baihu-resetpwd-管理员密码重置) | 交互式或命令行快速重置管理员密码 | 管理员密码遗忘应急找回 |
| [`baihu restore`](#5-baihu-restore-数据备份恢复) | 从本地 `.zip` 备份包全量恢复数据库与配置 | 系统迁移、灾难恢复 |
| [`baihu webui`](#6-baihu-webui-前端主题管理) | 管理、切换、重置第三方 WebUI 前端包 | 自定义主题包管理、界面回退 |
| [`baihu depinstall`](#7-baihu-depinstall-依赖一键自动补全) | 智能分析执行日志并自动安装缺失的依赖包 | 脚本依赖报错快速排查与自动补齐 |
| [`baihu builtininstall`](#8-baihu-builtininstall-内建助手库安装) | 为所有 Python / Node.js 运行时安装面板原生 SDK | 新增多版本解释器后一键注入 SDK |
| [`baihu completion`](#9-baihu-completion-shell-自动补全) | 生成 PowerShell / Bash / Zsh 的 Tab 自动补全脚本 | 提升终端交互与命令敲击体验 |
| [`baihu version`](#10-baihu-version-版本查询) | 查看当前二进制版本号 (同 `-v`, `-V`) | 环境排查、版本确认 |

---

## 详细命令场景与 Demo 示例

### 1. `baihu server` 服务启动

启动白虎面板核心后台服务（内置 Web 界面、任务调度引擎与 WebSocket 服务）。

#### 常见使用场景
- **直接前台运行**：测试或调试面板运行状态；
- **后台守护运行**：结合 `nohup`、`systemd` 或 Windows 服务在后台长期运行。

#### 示例
```bash
# 1. 前台直接运行（输出实时运行日志）
./baihu server

# 2. Linux 下后台静默运行
nohup ./baihu server > /dev/null 2>&1 &

# 3. Windows 下后台运行 (PowerShell)
Start-Process -FilePath ".\baihu.exe" -ArgumentList "server" -WindowStyle Hidden
```

---

### 2. `baihu task` 任务运维控制

专为纯终端操作与自动化工作流打造的轻量级任务管理工具，无需打开浏览器即可快速管理任务与查看执行日志。

> [!TIP]
> 目标识别非常智能：支持传入**任务 ID**、**任务名称（模糊/精准搜索）**，或直接传入字面量 **`repo`** 快捷操作主力仓库任务。

#### 子命令与参数详解

| 子命令 | 参数 | 默认值 | 描述 |
| :--- | :--- | :--- | :--- |
| `list` | `-name` | `""` | 按任务名称或备注进行模糊筛选 |
| | `-type` | `""` | 按任务类型筛选（如 `task`, `repo`） |
| | `-page` | `1` | 查询页码 |
| | `-size` | `20` | 每页展示条数 |
| `run` | `<目标>` | *(必填)* | 立即触发指定的任务或仓库 |
| `enable` | `<目标>` | *(必填)* | 启用指定任务（重新加入调度队列） |
| `disable`| `<目标>` | *(必填)* | 禁用指定任务（从调度队列摘除） |
| `status` | `<目标>` | *(必填)* | 查看任务最近一次执行的状态与输出 |
| | `[日志ID]`| *(选填)* | 指定查看某一次历史执行的完整日志 |
| `history`| `<目标>` | *(必填)* | 查看近期执行历史流水 |
| | `-limit` | `10` | 展示的历史流水记录条数 |

#### 场景与 Demo 示例

##### (1) 查看任务列表
```bash
# 默认展示第 1 页（20 条）
baihu task list

# 按关键词筛选包含 "签到" 的任务
baihu task list -name "签到"

# 仅筛选仓库同步类任务 (repo)，每页展示 50 条
baihu task list -type repo -size 50

# 查看第 2 页，每页 10 条
baihu task list -page 2 -size 10
```

##### (2) 手动触发执行任务
```bash
# 按任务 ID 触发
baihu task run a1b2c3d4e5

# 按任务名称模糊匹配触发
baihu task run "每日签到"

# 一键触发仓库同步任务
baihu task run repo
```

##### (3) 启用 / 禁用任务
```bash
# 启用某个任务
baihu task enable a1b2c3d4e5

# 禁用仓库同步任务
baihu task disable repo
```

##### (4) 查看任务执行日志与退出状态
```bash
# 查看指定任务最近一次执行的详细信息（包含状态、耗时、退出码及解压后的标准输出）
baihu task status a1b2c3d4e5

# 查看仓库任务最新一次拉取输出
baihu task status repo

# 查看某次特定执行日志（传入特定日志ID）
baihu task status a1b2c3d4e5 clk8901234567890
```

##### (5) 查询任务历史流水
```bash
# 查看任务最近 10 次执行记录
baihu task history a1b2c3d4e5

# 查看最近 30 次执行流水
baihu task history repo -limit 30
```

---

### 3. `baihu reposync` 仓库与脚本同步

用于将远程 Git 仓库或文件直链高速同步到本地目录，支持青龙注释解析、过滤白名单、黑名单关键字剔除等。

#### 参数列表

| 参数名 | 默认值 | 描述 |
| :--- | :--- | :--- |
| `--source-type` | `git` | 同步源类型：`git`（Git 仓库）或 `url`（文件直链下载） |
| `--source-url` | *(必填)* | 远程仓库地址（如 `https://github.com/xxx/repo.git`）或下载 URL |
| `--target-path` | `""` *(可选)* | 本地存储路径。**可不指定**（留空时默认自动保存在 `scripts/` 下对应的仓库目录中）；若显式指定支持环境变量占位符（如 `$SCRIPTS_DIR$/my_folder`） |
| `--branch` | `""` | 指定 Git 分支名（留空则自动探测远程默认分支） |
| `--path` | `""` | 稀疏检出（Sparse checkout）指定的子目录，或单文件模式下的文件名 |
| `--single-file` | `false` | 是否开启单文件提取模式（仅提取指定的一个文件） |
| `--proxy` | `none` | GitHub 加速代理类型：`none`、`ghproxy`、`mirror`、`custom` |
| `--proxy-url` | `""` | 自定义代理前缀（仅在 `--proxy=custom` 时生效） |
| `--auth-token` | `""` | 私有仓库访问 Token（HTTP Basic Auth 或 Personal Access Token） |
| `--http-proxy` | `""` | 通用 HTTP/SOCKS5 网络代理（如 `http://127.0.0.1:7890`） |
| `--whitelist-paths` | `""` | 白名单保留路径（支持竖线 `\|` 或逗号分隔） |
| `--blacklist` | `""` | 黑名单关键字过滤（竖线 `\|` 分隔），匹配到的脚本将被删除 |
| `--dependence` | `""` | 依赖文件保护关键字（竖线 `\|` 分隔），强制保留不被清理 |
| `--extensions` | `""` | 脚本扩展名白名单（竖线 `\|` 分隔，如 `.py\|.js\|.sh`） |
| `--commenttotask` | `false` | 是否自动解析脚本头部青龙注释（`new Env('xxx')` 与 cron 规则）并注册任务 |
| `--pre-command` | `""` | 为发现的新任务配置默认前置执行命令 |
| `--post-command` | `""` | 为发现的新任务配置默认后置执行命令 |
| `--repo-name` | `""` | 自定义保存的本地仓库文件夹名称（不指定时自动根据仓库 URL 识别） |
| `--task-timeout` | `30` | 同步命令执行超时时间（分钟） |

#### 场景与 Demo 示例

##### (1) 最简同步（不指定 `--target-path`，自动存入默认 scripts 目录）
> 💡 `--target-path` 参数**完全可选**。当不指定时，系统会自动在 `scripts/` 目录下按仓库标识创建对应文件夹：
```bash
# 最简拉取：自动在 scripts 目录下同步并使用 ghproxy 代理加速
baihu reposync \
  --source-url https://github.com/myuser/myscripts.git \
  --proxy ghproxy

# 自定义目标存储目录示例（显式指定 $SCRIPTS_DIR$）
baihu reposync \
  --source-url https://github.com/myuser/myscripts.git \
  --target-path $SCRIPTS_DIR$/custom_folder \
  --proxy ghproxy
```

##### (2) 稀疏检出 (Sparse Checkout) 仅同步特定子目录
当仓库体积巨大（包含大量无关文件）时，仅拉取所需的脚本目录：
```bash
baihu reposync \
  --source-url https://github.com/myuser/huge-repo.git \
  --path "scripts/daily" \
  --proxy mirror
```

##### (3) 单文件下载模式
仅下载远程仓库中的某单个脚本文件：
```bash
baihu reposync \
  --source-url https://github.com/myuser/myscripts.git \
  --single-file true \
  --path "checkin.py"
```

##### (4) 过滤黑名单并自动解析青龙注释注册定时任务
```bash
baihu reposync \
  --source-url https://github.com/myuser/myscripts.git \
  --blacklist "test|backup|utils" \
  --dependence "package.json|requirements.txt|sendNotify.js" \
  --extensions ".py|.js" \
  --commenttotask true
```

##### (5) 指定分支与白名单保护路径（--branch, --whitelist-paths, --repo-name）
```bash
# 指定拉取 dev 分支，自定义保存文件夹名称为 my_dev_scripts，并保护 config 目录不被清理
baihu reposync \
  --source-url https://github.com/myuser/myscripts.git \
  --branch "dev" \
  --repo-name "my_dev_scripts" \
  --whitelist-paths "config|assets"
```

##### (6) 自动为发现的任务注入前置与后置命令（--pre-command, --post-command）
```bash
# 同步脚本并在自动创建任务时，为所有任务统一配置运行前拉取依赖、运行后清理临时文件的钩子
baihu reposync \
  --source-url https://github.com/myuser/myscripts.git \
  --commenttotask true \
  --pre-command "npm install --production" \
  --post-command "rm -rf /tmp/cache_*"
```

##### (7) 自定义代理前缀与本地 HTTP 代理（--proxy custom, --proxy-url, --http-proxy）
```bash
# 场景 A: 使用自定义的 GitHub 反代加速节点
baihu reposync \
  --source-url https://github.com/myuser/myscripts.git \
  --proxy "custom" \
  --proxy-url "https://my-custom-proxy.domain.com/"

# 场景 B: 走本地 Clash / v2ray 等 HTTP 代理通道进行拉取
baihu reposync \
  --source-url https://github.com/myuser/myscripts.git \
  --http-proxy "http://127.0.0.1:7890"
```

##### (8) 私有仓库鉴权与单文件直链下载（--auth-token, --source-type url, --task-timeout）
```bash
# 场景 A: 携带 GitHub Token 拉取私有仓库，设置超时 15 分钟
baihu reposync \
  --source-url https://github.com/myorg/private-scripts.git \
  --auth-token "ghp_xxxxxxxxxxxxxxxxxxxx" \
  --task-timeout 15

# 场景 B: 直接从 URL 直链下载单文件脚本
baihu reposync \
  --source-type "url" \
  --source-url "https://raw.githubusercontent.com/myuser/scripts/main/test.py" \
  --path "test.py"
```

---

### 4. `baihu resetpwd` 管理员密码重置

忘记登录密码时的应急处理工具。可直接在宿主机或容器内直接重置管理员密码。

#### 场景与 Demo 示例

##### (1) 交互式重置（进入 Docker 容器或本地终端）
```bash
# 在 Docker 容器内执行
docker exec -it baihu baihu resetpwd

# 本地直接运行
./baihu resetpwd
```
*提示输入新密码时，直接按回车可自动生成 12 位高强度随机密码并显示在屏幕上。*

##### (2) 指定用户重置
```bash
baihu resetpwd admin
```

---

### 5. `baihu restore` 数据备份恢复

将导出的 `.zip` 数据备份包全量覆盖恢复到当前面板中（包含数据库配置、定时任务、变量机密、通知渠道等）。

#### 参数说明
```bash
baihu restore <备份文件路径.zip>
```

#### 场景与 Demo 示例
```bash
# 恢复本地指定备份
./baihu restore /data/backup/baihu-backup-20260817.zip

# 在 Docker 容器内恢复备份
docker exec -it baihu baihu restore /app/data/backup_20260817.zip
```
> [!WARNING]
> 恢复操作会覆盖现有的所有数据库表结构与配置数据，请在操作前确认备份文件的有效性。

---

### 6. `baihu webui` 前端主题管理

白虎面板支持插件化替换前端 Web 界面。通过 `baihu webui` 子命令可以在终端轻松管理和切换不同的前端包。

#### 支持子命令
- `list`：列出当前已安装的所有 WebUI 资源包及其激活状态；
- `set <name>`：切换并激活指定的 WebUI 前端；
- `reset`：一键回退到系统内置的官方默认 WebUI；
- `delete <name>`：从系统中删除指定的前端包。

#### 场景与 Demo 示例

```bash
# 1. 查看当前安装的前端包列表
baihu webui list

# 输出示例:
# ====================================================================================================
# 名称                 | 版本         | 作者            | 状态       | 描述
# ----------------------------------------------------------------------------------------------------
# default              | 1.0.0        | Official        | 使用中     | 系统内置官方默认前端
# dark-cyberpunk       | 1.2.0        | Developer       | -          | 赛博朋克深色高对比度主题
# ====================================================================================================

# 2. 切换激活第三方前端主题
baihu webui set dark-cyberpunk

# 3. 出现显示异常时，一键回退至官方默认前端
baihu webui reset

# 4. 删除不需要的旧前端包
baihu webui delete dark-cyberpunk
```

---

### 7. `baihu depinstall` 依赖一键自动补全

当脚本执行报错（如 Python `ModuleNotFoundError: No module named 'requests'` 或 Node.js `Cannot find module 'axios'`）时，该命令能够根据**日志 ID** 自动分析错误输出，精准识别缺失的依赖包名并自动安装到对应运行环境中。

#### 参数说明
```bash
baihu depinstall <日志ID>
```

#### 场景与 Demo 示例
```bash
# 1. 假设任务执行失败，从历史或 status 中获知日志 ID 为 c0ab123456
baihu depinstall c0ab123456

# 终端输出流程示例:
# >> 分析结果: 从运行日志中检测到以下缺失依赖包：
#    [python]: requests, bs4
# >> 是否确认自动安装上述依赖包？(y/N): y
# >> 正在安装 [requests] -> 执行指令: mise exec python@3.11 -- pip install requests
# >> 【成功】依赖包 [requests] 安装成功！
```

---

### 8. `baihu builtininstall` 内建助手库安装

白虎面板内置了开箱即用的原生消息推送 SDK（支持 Python 的 `from baihu import notify` 与 Node.js 的 `const { notify } = require('baihu-notify')`）。

当您在面板中通过 Mise 新增安装了新的 Python 或 Node.js 版本后，运行此命令可**自动为所有已安装的解释器版本全局注册注入 SDK**，无需手动重复 `pip install` 或 `npm install`。

#### 场景与 Demo 示例
```bash
# 为所有本地已安装的 Node.js 和 Python 版本批量注入白虎助手 SDK
baihu builtininstall

# 输出示例:
# >> [Builtin] 开始为 mise 环境安装内建包...
# >> [Builtin] 正在为 node@20.10.0 安装内建包...
# >> [Builtin] 为 node@20.10.0 安装成功
# >> [Builtin] 正在为 python@3.11.8 安装内建包...
# >> [Builtin] 为 python@3.11.8 安装成功
# >> [Builtin] 内建包安装流程完成
```

---

### 9. `baihu completion` Shell 自动补全

白虎面板 CLI 针对 **PowerShell**、**Bash** 与 **Zsh** 提供了原生的 Tab 键命令、子命令及参数自动补全支持。

#### 场景与配置示例

##### (1) PowerShell (Windows / pwsh)
```powershell
# 临时在当前窗口启用:
baihu completion powershell | Out-String | Invoke-Expression

# 永久启用 (写入系统配置文件 $PROFILE):
if (!(Test-Path $PROFILE)) { New-Item -Type File -Path $PROFILE -Force }
baihu completion powershell | Out-File -Append -Encoding utf8 $PROFILE
```

##### (2) Bash (Linux / macOS)
```bash
# 临时在当前会话生效:
source <(baihu completion bash)

# 永久生效 (写入 ~/.bashrc):
baihu completion bash > ~/.baihu_completion.bash
echo 'source ~/.baihu_completion.bash' >> ~/.bashrc
```

##### (3) Zsh (Linux / macOS)
```zsh
# 永久生效 (写入 ~/.zshrc):
baihu completion zsh > ~/.baihu_completion.zsh
echo 'source ~/.baihu_completion.zsh' >> ~/.zshrc
```

---

### 10. `baihu version` 版本查询

打印当前编译的白虎面板版本号及发布构建信息。

#### 示例
```bash
baihu version
# 或简写
baihu -v
```

---

## 常用运维组合技巧

### 技巧 1：快速重置密码并重启服务
```bash
baihu resetpwd admin
# 如果使用 systemd 托管
systemctl restart baihu
```

### 技巧 2：在 CI/CD 流水线中自动更新并触发任务
```bash
# 1. 触发代码库拉取
baihu task run repo
# 2. 检查最近一次拉取结果
baihu task status repo
# 3. 立即触发业务脚本
baihu task run "数据清洗任务"
```
