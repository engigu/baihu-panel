# 更新日志 (v1.4.2)

### 2026.10.01 - 系统设置全模块视觉与交互重构、声明式应用卸载级联清理与标签自动回收、Windows托盘更新弹窗优化

🎉 **新增功能与体验升级**
* **系统设置全模块视觉与交互体验深度重构 (Feature & UI & Refactor)**：
  - **安全设置解耦与 2FA 体验升级**：彻底解耦原有嵌套套娃卡片结构，将管理员凭据独立为身份卡并支持密码明密文快捷切换；2FA 两步验证新增安全盾安全等级评估、主流身份验证器客户端配置指引与极简状态卡；
  - **站点设置模块化布局与体验打磨**：站点设置重构为 2x2 模块化对称网格布局（站点外观、OpenAPI 开放接口、日志生命周期与自动清理、配置保存与生效）；优化日志清理策略为横向行式条目，数字输入框去除原生调节箭头并优化单位贴合排版；
  - **前端定制主题画廊网格与防白屏救生命令**：废弃冗长表格，升级为现代主题画廊卡片网格，展示主题名称、作者、版本及一键预览切换；新增终端防白屏应急恢复命令一键复制功能，支持现代拖拽上传；
  - **调度设置参数矩阵与硬件规格一键套用**：重组为核心并发参数与机制指南双列卡片，新增基于服务器硬件规格（1C1G、2C2G、4C4G 等）的推荐配置矩阵与一键套用功能；
  - **备份恢复双卡片并排与拖拽上传**：重构为系统快照与数据恢复双卡片并排，集成备份状态指示器与交互式拖拽区，简化操作链路；
  - **关于页面品牌质感提升**：采用单/双卡片一体化视觉，引入 Inter 字体现代排印的高质感数值卡，恢复完整开源免责声明并优化排版；
  - **代码规范与样式告警清理**：清理容器与子组件未使用的图标与变量，补充标准 `appearance` 兼容性属性，消除所有 TypeScript 与 CSS 兼容性告警。
* **声明式应用卸载级联清理与标签自动回收 (Feature & App)**：
  - **环境变量安全级联清理**：卸载应用时支持 `CleanEnvs` 选项，精准分析并安全清理未被其他任务复用的关联环境变量，避免变量遗留与数据冗余；
  - **孤儿标签自动回收**：新增孤儿标签自动清理机制，卸载应用后自动识别并删除无外部引用的任务标签与环境变量标签；
  - **调度器精准同步注销**：卸载应用时同步从底层调度器 (`CronManager`) 中移除主任务及所有受控子任务的 Cron 计划与执行上下文。

✨ **问题修复与交互体验优化**
* **Checkbox 默认勾选修复与弹窗交互体验增强 (UI & Fix)**：
  - 修复底层 Checkbox 组件默认勾选无效缺陷，应用卸载确认弹窗默认勾选“级联清理关联环境变量”；
  - 任务与应用删除弹窗接入统一 Loading 状态与防重复点击保护，增强操作交互反馈；
  - 修复 DialogContent 背景残留导致的 WAI-ARIA `aria-hidden` 控制台警告。
* **Windows 托盘更新弹窗优化 (Fix & Tray)**：
  - 优化 Windows 系统托盘程序的新版本更新提示弹窗，支持内容区域平滑滚动，保证长更新日志能够完整清晰展示。

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

  直接运行附件中的安装程序 `BaihuPanel-Setup-v1.4.2-windows-amd64.exe` 完成安装，程序将自动完成桌面快捷方式创建、开机自启配置与 `127.0.0.1 baihu.local` 域名绑定。安装完成后通过桌面图标或任务栏右下角托盘图标即可直接打开 `http://baihu.local:38052`。

* **方式二：二进制单文件运行（免安装/便携版）**

  解压下载好的 `.zip` 压缩包（如 `baihu-windows-amd64.zip`），进入解压目录并在 PowerShell 中运行：

  ```powershell
  .\baihu.exe server
  ```

---

**访问面板：**
* 启动后访问：`http://baihu.local:38052` 或 `http://localhost:38052` (或单文件默认端口 `http://localhost:8052`)
* **默认账号**：用户名 `admin`，密码见面板首次启动时的控制台日志。
