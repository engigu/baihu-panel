# 更新日志 (v1.1.21)

### 2026.07.20 - Windows 平台深度适配、网页终端 Ctrl+C 中止与单文件构建

🎉 **新增与优化**
* **Windows 平台适配包重构**：新建并集成了后端 `internal/windows` 与前端 `web/src/windows` 专有包，统一收拢 Windows 的特异性环境检测、PSReadline 影响规避、PATH 优先级修复 (FixPathEnv) 等底层逻辑，大幅提升了在 Windows 平台直接运行时的环境稳定性与规范度。
* **网页终端 Ctrl+C 中断支持**：支持了 Windows 网页终端下通过快捷键 `Ctrl+C` 中止运行中程序，后端会拦截 `\x03` 信号并调用 `taskkill /F /T` 强行递归终结前台子进程树，并保持外层 Shell 会话完好，与 Linux/macOS 的体验全面看齐。
* **Monaco 编辑器高亮与检测**：前端编辑器新增了保存脚本时的风险警告拦截审查 (`scriptCheck.ts`)，针对 `timeout` 或 `pause` 等后台挂起指令给出安全平替建议；同时为 Monaco 编辑器补齐了 `.bat`、`.cmd`、`.ps1` 等 Windows 脚本语言的语法高亮支持。
* **编译与自动化发布**：在 `Makefile` 中添加了跨平台编译 Windows 二进制的 `release-windows` 目标；重构了 GitHub Actions 自动发布工作流 `.github/workflows/release.yml`，在发布时自动编译并打包 Windows 平台的单文件发布包 (`baihu-windows-amd64.zip`) 并自动上传 Release 附件。
* **xterm 终端换行 Bug 修复**：开启了终端组件的 `convertEol: true` 自动换行翻译配置，彻底解决了 Windows 管道重定向模式下，因 Shell 回显单 `\n` 导致首行输出排版乱折行的排版问题。
* **Windows 部署使用文档**：更新了部署说明文档，新增了“二进制单文件运行 (Linux / Windows)”专栏，细化了 `mise` 以及 `pwsh 7+` 工具链的安装指导。

**✨ 修复与改进**
* **脚本执行参数校验**：修复了在“测试运行” Windows 脚本时，即便不需要运行环境也会强行拼接 `python` / `node` 执行器前缀导致命令无法执行的缺陷。

---



---


> 出于安全及环境隔离考虑，推荐使用 Docker/Compose 部署方式。[镜像地址](https://github.com/engigu/baihu-panel/pkgs/container/baihu)



### 🐳 方式一：Docker 部署（推荐）
[部署文档](https://github.com/engigu/baihu-panel?tab=readme-ov-file#%E5%BF%AB%E9%80%9F%E9%83%A8%E7%BD%B2)

### 🚀 方式二：单文件部署 (Linux / Windows)
从当前 Release 的附件中下载对应架构和平台的部署压缩包（Linux 为 `.tar.gz`，Windows 为 `.zip`）。

#### 🐧 Linux 平台
**1. ⚠️ 重要前置依赖：手动安装 `mise`**
单文件直接运行依赖宿主机系统环境，请务必先安装 [mise](https://mise.jdx.dev/getting-started.html) 供任务调度及环境管理使用：
```bash
curl https://mise.run | sh
export PATH="~/.local/share/mise/bin:~/.local/share/mise/shims:$PATH"
```

**2. 运行面板：**
```bash
tar -xzvf baihu-linux-amd64.tar.gz
chmod +x baihu-linux-amd64
./baihu-linux-amd64 server
```

#### 🪟 Windows 平台
**1. ⚠️ 重要前置依赖**
* **安装 `mise`**（用于统一依赖和运行时环境管理）：
  在 PowerShell 中运行以下命令安装：
  ```powershell
  irm https://mise.run | iex
  ```
* **安装 `pwsh`**（PowerShell 7.6+，用于执行后台任务）：
  白虎面板在 Windows 下运行任务和工具链强依赖 PowerShell 7+。请参考 [微软官方 PowerShell 安装文档](https://learn.microsoft.com/zh-cn/powershell/scripting/install/install-powershell-on-windows?view=powershell-7.6) 安装，或通过 `winget` 快捷安装：
  ```powershell
  winget install Microsoft.PowerShell
  ```

**2. 运行面板：**
解压下载好的 `.zip` 压缩包，进入解压目录并打开 PowerShell，运行：
```powershell
.\baihu.exe server
```

---

**访问面板：**
启动后访问：http://localhost:8052

**登录信息：**
默认账号：用户名 `admin`，密码见面板首次启动时的控制台日志。


