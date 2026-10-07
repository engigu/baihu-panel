# 更新日志 (v1.5.0)

### 2026.10.07 - 原生内置 Model Context Protocol (MCP) 服务、Server酱推送通道、向导式通知渠道网格、文件树懒加载与全局检索、Agent 实时同步与 Windows 二进制支持

🎉 **新增功能与体验升级**
* **原生内置 Model Context Protocol (MCP) 协议服务支持 (Major Feature & AI)**：
  - **全套原子运维工具集**：原生集成 24 个全生命周期原子运维工具，包含定时任务管理（增删改查、手动执行、超时强制终止）、声明式应用与官方应用商店全流程管控（检索、安装、切换运行场景、级联卸载）、环境变量安全脱敏读写、脚本沙箱文件管理、执行历史日志与空间清理、Git 仓库绑定与即时拉取、多渠道消息通知连通性自检与自定义告警推送（[#182](https://github.com/engigu/baihu-panel/pull/182)）；
  - **7 组上下文资源与智能诊断 Prompt 模板**：提供系统运行状态、通知渠道、已装应用、Git仓库、任务模型、执行日志及脚本源码的 URI 资源引入；内置任务失败智能归因排查 (`diagnose_task_failure`) 与自然语言生成定时任务 (`create_scheduled_task`) Prompt 模板；
  - **双传输协议支持**：支持远程网络传输（SSE/HTTP）与 CLI Stdio 子进程本地管道传输，自动隔离控制台输出保障标准 JSON-RPC 纯净通信；
  - **严格并发与协程控制**：内置 `ToolConcurrencyLimiter(4)` 中间件限制并发调用，排队超时自动熔断；限制活跃 SSE 连接数（$\le 3$），单客户端仅占用 2 协程，零任务常驻协程；
  - **可视化设置与指南**：站点设置中心新增“AI 客户端连接 (MCP)”指引面板，支持一键复制 SSE 链接与客户端配置 JSON，并提供官方文档。
* **ServerChan（Server酱）消息推送通道支持 (Feature & Notify)**：
  - 消息通知中心原生集成 ServerChan（Server酱·Turbo版）通知渠道，支持 SendKey 配置与连通性即时自检（[#183](https://github.com/engigu/baihu-panel/pull/183)）。
* **通知渠道添加流程交互重构 (UI & Refactor)**：
  - 将通知渠道选择升级为现代向导式卡片网格，醒目展示品牌图标、定位用途与特性支持；精简配置项并优化表单分组，大幅降低配置门槛。
* **脚本文件管理全流程重构 (Files & Editor & OpenAPI)**：
  - **文件树按需懒加载 (Lazy Load)**：重构文件树加载机制为按需懒加载，大幅提升海量脚本与深层嵌套目录下的浏览性能与首屏渲染速度；
  - **全盘模糊检索**：支持脚本目录树全局文件名与相对路径模糊搜索，前端设置中心新增搜索结果数量上限调节；
  - **OpenAPI 深度集成**：脚本文件树遍历、源码读取、沙箱写入与安全删除能力全量开放接入 OpenAPI 接口。
* **集群 Agent 深度演进与 Windows 客户端支持 (Agent & Windows)**：
  - **实时增量同步体系**：新增 Agent 脚本多目录精准实时变动监听与增量同步机制，支持防抖事件合并与监听休眠机制，避免无效 IO；
  - **物理内存深度优化**：优化 Agent 进程驻留内存管理，高频修改后主动触发物理内存回收与系统句柄释放；
  - **Windows Agent 正式发布**：CI/CD 产物流水线正式集成 Windows 二进制 Agent 打包；重构下载弹窗为全平台分类标签，提供一键复制的完整初始化命令集。

✨ **问题修复与细节打磨**
* **日志详情终端自适应排版修复 (Web & Fix)**：
  - 修复任务执行日志详情弹窗/侧边栏展开瞬间列宽未稳定导致的 5 字窄折行与右侧留白问题；新增 `fitAndRefresh` 列宽动态监测与多阶段尺寸校准，列宽变化时自动按真实全宽重新平铺渲染；
  - 优化日志 SSE 瞬时并发落库时的防抖查库与系统提示回车换行格式。
* **界面细节与小屏响应式排版优化 (Style & UI)**：
  - 优化“关于”页面在移动端小屏设备下的卡片网格排版与文本对齐；
  - 统一站点设置中心小屏设备下的按钮间距与表单交互风格。
* **安全依赖升级与 Dependabot 漏洞清零 (Security)**：
  - 升级 `vue` 至 `3.5.43`，修复 `@vue/server-renderer` XSS 漏洞；
  - 锁定升级 `source-map-js`（1.2.2）、`brace-expansion`（5.0.12）、`fast-uri`（3.1.8）、`dompurify`（3.4.16）；
  - 修正 `nanoid` 为 CommonJS 兼容版本 `3.3.20`，彻底消除 7 项安全漏洞告警。

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

  直接运行附件中的安装程序 `BaihuPanel-Setup-v1.5.0-windows-amd64.exe` 完成安装，程序将自动完成桌面快捷方式创建、开机自启配置与 `127.0.0.1 baihu.local` 域名绑定。安装完成后通过桌面图标或任务栏右下角托盘图标即可直接打开 `http://baihu.local:38052`。

* **方式二：二进制单文件运行（免安装/便携版）**

  解压下载好的 `.zip` 压缩包（如 `baihu-windows-amd64.zip`），进入解压目录并在 PowerShell 中运行：

  ```powershell
  .\baihu.exe server
  ```

---

**访问面板：**
* 启动后访问：`http://baihu.local:38052` 或 `http://localhost:38052` (或单文件默认端口 `http://localhost:8052`)
* **默认账号**：用户名 `admin`，密码见面板首次启动时的控制台日志。
