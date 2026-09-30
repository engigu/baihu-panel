# 更新日志 (v1.4.1)

### 2026.09.30 - 远程 Agent 工作目录解析修复、Windows Agent Shell 免依赖回退、Bark 动态参数覆盖与安全加固

🎉 **新增功能与体验升级**
* **Windows Agent Shell 免依赖降级回退支持 (Feature & Agent)**：
  - **自动回退内置 PowerShell**：针对 Windows 远程 Agent 节点开放 Shell 降级策略，当受控机器未安装 PowerShell 7 (`pwsh.exe`) 时自动回退使用系统自带的 `powershell.exe`，实现单二进制零依赖开箱即用（主面板服务端仍保持 `pwsh` 强校验）。
* **Bark 推送动态参数覆盖与内置 SDK 增强 (Feature)**：
  - **发送级个性化参数覆盖**：Bark 通知渠道支持在调用发送接口时通过 `options` 动态传入额外参数（如 `group`、`icon`、`url`、`sound`、`level`、`badge` 等），按需灵活覆盖渠道全局默认配置（[#179](https://github.com/engigu/baihu-panel/pull/179)）；
  - **多语言内置 SDK 同步升级**：同步更新内置 Python (`notify.py`) 与 Node.js (`notify.js`) 通知助手库及使用文档，无缝支持透传渠道扩展参数。
* **演示模式备份安全加固与文档升级 (Security & Docs)**：
  - **演示模式严禁备份导出**：在演示模式下（`BH_DEMO_MODE=true`）全面拦截并禁止创建与下载系统备份文件，防止公共演示环境数据或配置通过备份接口外泄；
  - **文档与 README 视觉排版优化**：重构 README 排版结构，采用 GitHub 原生告警块与轻量矢量徽章，新增贡献者头像墙与社区交流群入口。

✨ **问题修复与交互体验优化**
* **远程 Agent 任务工作目录解析修复与执行优化 (Fix & Agent)**：
  - **消除 `$SCRIPTS_DIR$` 占位符污染**：修复 Agent 远程任务在创建、更新及切换启用开关时工作目录被错误归一化为面板本地 `$SCRIPTS_DIR$` 占位符（导致远程节点启动进程报错 `no such file or directory`）的缺陷，并对数据库历史存量脏数据实现全链路自动清洗与无损还原；
  - **跨平台盘符路径精准识别**：新增跨平台绝对路径识别，修复 Linux 服务端向 Windows Agent 下发盘符路径（如 `C:\...`）时被误判为相对路径并错误拼接 `$SCRIPTS_DIR$/` 前缀的问题；
  - **禁用任务手动触发与实时目录透传**：修复禁用状态或未配置 Cron 表达式的 Agent 任务无法通过面板点击“立即运行”的问题，并在下发立即执行指令时实时透传最新工作目录。
* **全局时间显示统一与界面细节打磨 (UI & Fix)**：
  - **时间格式化与状态展示优化**：统一应用市场、任务编辑、运行日志、通知渠道、系统监控及备份列表的时间友好度显示；禁用状态的任务隐藏无意义的“下次运行时间”；登录日志查询 IP 归属地失败时静默处理不再弹窗打扰（[#180](https://github.com/engigu/baihu-panel/pull/180)）。

---

> 💡 **提示**：出于安全及环境隔离考虑，推荐使用 Docker/Compose 部署方式。[镜像地址](https://github.com/engigu/baihu-panel/pkgs/container/baihu)

### 🐳 方式一：Docker 部署 (推荐)
[部署文档](https://github.com/engigu/baihu-panel?tab=readme-ov-file#%E5%BF%AB%E9%80%9F%E9%83%A8%E7%BD%B2)

---

### 🚀 方式二：单文件/安装包部署 (Linux / Windows)
从当前 Release 的附件中下载对应架构和平台的部署压缩包（Linux 为 `.tar.gz`，Windows 为安装包 `.exe` 或 `.zip`）。

#### 🐧 Linux 平台

**1. 安装前置依赖 `mise`**

单文件直接运行依赖宿主机系统环境，请务必先安装 [mise](https://mise.jdx.dev/getting-started.html) 供任务调度及环境管理使用：

```bash
curl https://mise.run | sh
export PATH="~/.local/share/mise/bin:~/.local/share/mise/shims:$PATH"
```

**2. 运行面板**

```bash
tar -xzvf baihu-linux-amd64.tar.gz
chmod +x baihu-linux-amd64
./baihu-linux-amd64 server
```

#### 🪟 Windows 平台

**1. 安装前置依赖**

* **安装 `mise`**（用于统一依赖和运行时环境管理）：

  在 PowerShell 中运行以下命令使用 `winget` 安装：
  ```powershell
  winget install jdx.mise
  ```

* **安装 `pwsh`**（PowerShell 7.6+，用于执行后台任务）：

  白虎面板在 Windows 下运行任务和工具链强依赖 PowerShell 7+。请参考 [微软官方 PowerShell 安装文档](https://learn.microsoft.com/zh-cn/powershell/scripting/install/install-powershell-on-windows?view=powershell-7.6) 安装，或通过 `winget` 快捷安装：
  ```powershell
  winget install Microsoft.PowerShell
  ```

**2. 运行面板**

* **方式一：使用 GUI 安装包安装（推荐）**

  直接运行附件中的安装程序 `BaihuPanel-Setup-v1.4.1-windows-amd64.exe` 完成安装，程序将自动完成桌面快捷方式创建、开机自启配置与 `127.0.0.1 baihu.local` 域名绑定。安装完成后通过桌面图标或任务栏右下角托盘图标即可直接打开 `http://baihu.local:38052`。

* **方式二：二进制单文件运行（免安装/便携版）**

  解压下载好的 `.zip` 压缩包（如 `baihu-windows-amd64.zip`），进入解压目录并在 PowerShell 中运行：

  ```powershell
  .\baihu.exe server
  ```

---

**访问面板：**
* 启动后访问：`http://baihu.local:38052` 或 `http://localhost:38052` (或单文件默认端口 `http://localhost:8052`)
* **默认账号**：用户名 `admin`，密码见面板首次启动时的控制台日志。
