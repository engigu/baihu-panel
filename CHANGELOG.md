# 更新日志 (v1.7.0)

### 2026.11.14 - 终端交互式控制台全新上线 (baihu tui)、登录页演示环境一键填入与提示卡片、命令规范收敛与代码健康治理

🎉 **新增功能与体验升级**
* **终端交互式控制台全新上线 (CLI & TUI & Feature)**：
  - **Bubble Tea 交互控制台集成**：新增 `baihu tui` 命令行子系统，基于 Charmbracelet Bubble Tea / Lipgloss 构建全屏响应式终端交互控制台，支持键盘快捷导航、菜单分类联动与动态视口渲染；
  - **全功能控制台分类导航**：系统内置四大运维分类：服务管理（启停重启、状态检查、日志追踪）、环境与依赖管理（Mise 扫描与配置、内建助手库安装）、系统维护与安全（密码重置、备份恢复、PageCache 回收、数据迁移）及系统诊断（版本查询、MCP 状态）；
  - **实时执行与输出捕获**：集成子进程命令派发管道与输出日志实时流式展示，支持长输出自由滚动浏览与自动退出确认；
  - **高危操作防误触确认**：对重置密码、恢复备份等不可逆操作引入安全确认对话框与参数动态交互输入，保障生产运维安全性。
* **登录页面演示环境提示与快速填入 (Web & Demo & Feature)**：
  - **演示模式视觉提示卡片**：登录页新增演示环境专属提示卡片，醒目标注演示环境特点与注意事项；
  - **一键填入体验优化**：支持一键将默认管理员凭据（`admin / admin123`）快速填入登录表单，大幅降低初次体验与测试成本。
* **CLI 命令层级收敛与高频排序优化 (CLI & Commands & Refactor)**：
  - **命令使用频次重构**：按日常运维高频场景全面重排 `internal/constant/commands.go` 中的命令注册顺序，核心服务指令与控制台工具优先展示；将单次查询类的 `version` 指令规范置底；
  - **类型定义标准收敛**：全局统一收敛并只保留标准 `CommandSpec` 结构体，消除冗余类型别名（`CommandInfo`），提升类型一致性。
* **代码健康度治理与冗余代码安全归档 (Refactor & Code Health)**：
  - **冗余代码安全注释**：全面排查并安全注释全项目未引用的函数与结构（覆盖 `agent/config.go`、`cmd/migrate`、`executor_service.go`、`buffer.go`、`crypto.go` 等），消除未引用的包导入与死代码隐患；
  - **TUI 启动横幅细节优化**：修复终端控制台启动日志版本号格式化缺陷，消除多余的前缀字符。
* **文档体系完善与 API 交互修复 (Docs & Fix)**：
  - **依赖架构设计说明**：于官方文档中系统性补充白虎面板采用 Mise 管理运行时与包依赖的设计初衷，深度解析与传统容器内置固定包管理的架构差异与隔离优势；
  - **API 文档路由拦截修复**：解决 VitePress 路由拦截导致的 API 文档分类侧边栏点击失效问题，恢复侧边栏默认折叠状态提升浏览体验。

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

  直接运行附件中的安装程序 `BaihuPanel-Setup-v1.7.0-windows-amd64.exe` 完成安装，程序将自动完成桌面快捷方式创建、开机自启配置与 `127.0.0.1 baihu.local` 域名绑定。安装完成后通过桌面图标或任务栏右下角托盘图标即可直接打开 `http://baihu.local:38052`。

* **方式二：二进制单文件运行（免安装/便携版）**

  解压下载好的 `.zip` 压缩包（如 `baihu-windows-amd64.zip`），进入解压目录并在 PowerShell 中运行：

  ```powershell
  .\baihu.exe server
  ```

---

**访问面板：**
* 启动后访问：`http://baihu.local:38052` 或 `http://localhost:38052` (或单文件默认端口 `http://localhost:8052`)
* **默认账号**：用户名 `admin`，密码见面板首次启动时的控制台日志。
