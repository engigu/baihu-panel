# 更新日志 (v1.5.9)

### 2026.10.30 - Docker 容器异步初始化内建助手库、Node.js 内建通知健壮性加固与返回值支持、变量机密编辑框最后编辑时间展示

🎉 **新增功能与体验升级**
* **Docker 容器异步初始化内建助手库 (Docker & Core & Perf)**：
  - **非阻塞异步派发初始化**：在容器公共启动入口脚本中抽象并统一 `init_builtin_packages`，将 `baihu builtininstall` 派发至后台异步执行，主服务进程无需同步等待内置包编译安装即可极速启动对外提供服务；
  - **临时状态标记与锁文件防护**：通过 `/tmp/.baihu_builtin_installed` 标记与防并发锁文件记录已就绪版本，避免容器重启或多实例重复触发安装。
* **Node.js 内建助手库 `baihu.notify` 健壮性加固与返回值支持 (Builtin & Notify & Fix)**：
  - **支持 Promise / Await 获取发送回执**：重构 Node.js 内建 `notify` 模块，全面支持返回 Promise，用户可通过 `const res = await baihu.notify(...)` 结构化获取投递状态、HTTP 状态码及错误详情（[#203](https://github.com/engigu/baihu-panel/pull/203)）；
  - **异常拦截与错误透出**：彻底修复旧版本静默吞掉异常的缺陷，针对缺少配置环境变量（`BHPKG_NOTIFY_TOKEN` / `BHPKG_NOTIFY_CHANNEL`）、非法接口 URL 及网络请求超时错误进行前置校验与标准控制台报错输出；
  - **文档与示例同步更新**：同步更新官方指南文档与 Node.js 示例脚本中的推荐调用姿势。
* **环境变量与安全机密最后编辑时间展示 (Environments & Web & Feature)**：
  - **对齐紧凑 DevOps 排版规范**：在环境变量与安全机密编辑弹窗（`EditEnvDialog.vue`）左下角新增“最后编辑于”时间提示，鼠标悬浮展示完整时间戳，与任务编辑弹窗、代码仓库弹窗等既有设计规范保持完全一致（[#204](https://github.com/engigu/baihu-panel/pull/204)）。
* **应用任务行菜单文案优化 (Apps & Tasks & Web)**：
  - **语义精准化**：优化声明式应用受控任务列表项的更多操作菜单文案为“更新详情”，消除歧义并与实际跳转行为保持一致。

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

  直接运行附件中的安装程序 `BaihuPanel-Setup-v1.5.9-windows-amd64.exe` 完成安装，程序将自动完成桌面快捷方式创建、开机自启配置与 `127.0.0.1 baihu.local` 域名绑定。安装完成后通过桌面图标或任务栏右下角托盘图标即可直接打开 `http://baihu.local:38052`。

* **方式二：二进制单文件运行（免安装/便携版）**

  解压下载好的 `.zip` 压缩包（如 `baihu-windows-amd64.zip`），进入解压目录并在 PowerShell 中运行：

  ```powershell
  .\baihu.exe server
  ```

---

**访问面板：**
* 启动后访问：`http://baihu.local:38052` 或 `http://localhost:38052` (或单文件默认端口 `http://localhost:8052`)
* **默认账号**：用户名 `admin`，密码见面板首次启动时的控制台日志。
