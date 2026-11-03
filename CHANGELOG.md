# 更新日志 (v1.6.0)

### 2026.11.03 - 任务超时支持0不限制、限制并发任务运行中快捷停止、执行历史日志防重渲染、演示模式切面重构与容器控存调优

🎉 **新增功能与体验升级**
* **任务超时控制与 0（不限制）支持 (Tasks & Backend & Fix)**：
  - **支持不限制超时运行**：移除数据模型中 `Timeout` 字段的默认值标签，解决 GORM 在创建/更新时将零值判定为未赋值从而自动回退为 30 分钟的缺陷，底层调度引擎原生支持 `timeout <= 0` 长期或守护型脚本无超时限制运行；
  - **前端配置与解析兼容**：任务高级配置中超时输入框最小值放宽为 `0` 并补充不限制说明，修正批量修改与仓库导入解析对 `0` 值的短路判断，确保 `0` 正常保存与导出。
* **限制并发任务列表动态切换停止按钮 (Tasks & Web & Feature)**：
  - **运行中即时停止**：针对开启单实例保护（限制并发）的普通任务及仓库同步任务，在执行中将桌面端操作位由“执行/同步”动态切换为醒目的“停止”按钮，移动端卡片操作同步切换为“停止”，附带独立的防重点击与加载中反馈；
  - **通用停止接口兼容 TaskID**：后端执行器扩展 `StopTaskExecution` 支持以 `task_id` 定位并终止当前正在运行的执行实例，无需强制进入日志弹窗即可在列表一键终止。
* **执行历史日志重复输出与竞态修复 (Logs & Web & Fix)**：
  - **多任务切换防重与生命周期重构**：重构 `LogContent.vue` 的请求版本追踪机制，解决历史日志在快速切换或重复点击时内容重复追加的问题（[#205](https://github.com/engigu/baihu-panel/issues/205)）；
  - **幂等加载与取消保护**：优化日志加载请求的幂等判断与状态清理，确保日志内容准确纯净。
* **演示模式切面中间件与守卫重构 (Security & Demo & Refactor)**：
  - **切面中间件统一拦截**：提取专用的演示模式切面中间件与前端路由守卫，统一拦截写操作并规范占位样式。
* **Docker 容器内存回收优化与启动精简 (Docker & MemOpt & Perf)**：
  - **内建助手库缓存主动回收**：内建助手库安装完成后主动回收活跃文件缓存（Active Page Cache），消除安装过程中的临时内存驻留；
  - **启动入口解耦精简**：优化容器入口流程，进一步精简启动耗时。

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

  直接运行附件中的安装程序 `BaihuPanel-Setup-v1.6.0-windows-amd64.exe` 完成安装，程序将自动完成桌面快捷方式创建、开机自启配置与 `127.0.0.1 baihu.local` 域名绑定。安装完成后通过桌面图标或任务栏右下角托盘图标即可直接打开 `http://baihu.local:38052`。

* **方式二：二进制单文件运行（免安装/便携版）**

  解压下载好的 `.zip` 压缩包（如 `baihu-windows-amd64.zip`），进入解压目录并在 PowerShell 中运行：

  ```powershell
  .\baihu.exe server
  ```

---

**访问面板：**
* 启动后访问：`http://baihu.local:38052` 或 `http://localhost:38052` (或单文件默认端口 `http://localhost:8052`)
* **默认账号**：用户名 `admin`，密码见面板首次启动时的控制台日志。
