# 更新日志 (v1.5.1)

### 2026.10.09 - Windows托盘无黑窗静默运行、容器内存口径对齐Docker Stats、OpenAPI MCP服务健壮性加固

🎉 **新增功能与体验升级**
* **Windows 系统托盘无黑窗静默运行与更新检测优化 (Tray & Windows & Perf)**：
  - **全链路无黑窗静默执行**：封装 `silentCmd` 底层命令执行工具，统一注入 `HideWindow: true` 与 `CREATE_NO_WINDOW (0x08000000)` 标志位；彻底消除 Windows 系统托盘启动、后台定时检查更新 (`baihu.exe version`)、服务重启与停止 (`taskkill`)、自动升级批处理脚本触发时弹出的黑控制台/黑屏闪烁问题；
  - **版本号缓存优化**：引入 `sync.Once` 本地版本号单例缓存机制，避免后台 2 小时定时器或右键菜单重复唤起外部进程获取版本。
* **容器内存优化与 Docker Stats 官方相减算法对齐 (MemOpt & Docker & Perf)**：
  - **Docker Stats 真实占用口径对齐**：引入 Docker CLI 官方标准的相减计算规则 `docker_stats_used = total_usage - inactive_file`，彻底解决控制台日志所报 Page Cache 与宿主机 `docker stats` 显示严重对不上的口径错位问题；
  - **cgroup v1/v2 双架构真实账本解析**：结构化提取 `memory.current` / `memory.usage_in_bytes`、`memory.max` / `memory.limit_in_bytes` 与 `memory.stat` 完整账本；
  - **自适应控存巡检日志重构**：容器自适应巡检日志透明并列输出 Docker 真实物理占用、内存上限使用率及 Page Cache 缓存回收效果；
  - **安全文件类型断言**：增加 `IsRegular` 普通文件类型断言，遍历缓存清理时安全跳过特殊设备、管道及 Unix 套接字文件，杜绝异常 IO；
  - **服务启动时序调优**：收拢 Mise 环境扫描读盘缓存的首次回收至 Web 服务对外监听之前；Docker 启动脚本新增 `dropcache` 主动卸载镜像启动初期的读盘缓存；
  - **架构深度收敛**：移除 `utils/runtime` 空壳层，底层内存管理与优化模块彻底收敛统一至 `internal/memopt`。
* **OpenAPI 与 MCP 协议服务健壮性加固 (MCP & OpenAPI & Fix)**：
  - **依赖注入空指针修复**：修复在 OpenAPI 模式下调用 MCP SSE 路由时，因未完整挂载 `FileService`、`AppService`、`NotifyService` 引发的 nil 指针 Panic 缺陷（[#187](https://github.com/engigu/baihu-panel/issues/187)）；
  - **自动兜底补全与全局恢复**：引入基于 `cmp.Or` 与惰性初始化的 `EnsureDefaults()` 依赖补全兜底机制；挂载 `server.WithRecovery()` 全局恢复中间件，保障 MCP 服务的极致高可用与容错。
* **DevOps 现代紧凑风格任务耗时与运行时长优化 (Task & Settings & Style)**：
  - **任务耗时 DevOps 紧凑化**：全链路将定时任务执行耗时重构为现代化紧凑格式（如 `1m 23s`、`450ms`），直观清晰；
  - **服务运行时长前端统一格式化**：关于页面后端仅返回服务运行秒数数值，由前端统一实现紧凑/完整时长自适应排版及悬停 Tooltip 提示，并移除冗余版本标签。
* **声明式应用 YAML 清单预览、复制与导出 (AppStore & Tasks & UI)**：
  - **应用市场清单查看**：应用市场详情弹窗支持直接预览应用标准化 `app.yaml` 清单，提供一键复制与下载能力；
  - **配置导出统一复用**：定时任务列表“导出应用配置”弹窗全面复用应用市场专业 YAML 代码高亮预览组件。
* **系统调度与连接池自适应缩容 (Goroutine & DB & Perf)**：
  - **后台清理任务统一收拢**：将系统定期垃圾回收、日志归档与空闲资源清理任务收拢至统一的 `SysCron` 调度架构；
  - **数据库连接池空闲缩容**：根据系统空闲周期动态缩容数据库空闲连接，降低低峰期系统句柄与常驻内存开销。
* **加速源与镜像矩阵统一 (Mirrors & Network)**：
  - 前后端统一封装加速源与镜像矩阵，首选 `gh-proxy` 并标准化浏览器 User-Agent 请求头。

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

  直接运行附件中的安装程序 `BaihuPanel-Setup-v1.5.1-windows-amd64.exe` 完成安装，程序将自动完成桌面快捷方式创建、开机自启配置与 `127.0.0.1 baihu.local` 域名绑定。安装完成后通过桌面图标或任务栏右下角托盘图标即可直接打开 `http://baihu.local:38052`。

* **方式二：二进制单文件运行（免安装/便携版）**

  解压下载好的 `.zip` 压缩包（如 `baihu-windows-amd64.zip`），进入解压目录并在 PowerShell 中运行：

  ```powershell
  .\baihu.exe server
  ```

---

**访问面板：**
* 启动后访问：`http://baihu.local:38052` 或 `http://localhost:38052` (或单文件默认端口 `http://localhost:8052`)
* **默认账号**：用户名 `admin`，密码见面板首次启动时的控制台日志。
