# 更新日志 (v1.1.29)

### 2026.09.08 - Windows 原生 ConPTY 伪终端重构、hosts 自动化域名映射、托盘 Modern UI 升级与仓库代理支持

🎉 **新增与优化**
* **Windows 原生 ConPTY 伪终端重构 (New)**：伪终端底层重构为 `github.com/ActiveState/termtest/conpty` 官方库，彻底解决了 Windows 托盘与无控制台模式下管道被占用导致终端卡在“已连接”/黑屏的问题；优化了 `pwsh.exe` 启动参数。
* **Windows 安装包 hosts 自动化映射 (New)**：Inno Setup 安装包 (`build/windows/installer.iss`) 全新集成 Pascal 自动处理脚本，在安装时自动写入 `127.0.0.1 baihu.local` 且保证完全幂等与唯一性，卸载时静默清理；Windows 托盘一键打开默认切换为 `http://baihu.local:38052`。
* **Windows 系统托盘 Modern 沉浸式 UI 升级 (New)**：使用系统 `uxtheme` 激活了 Windows 10 1903+ / Win11 原生沉浸式深色 Modern 菜单主题；优化了菜单层级与加粗样式，并支持在任务栏双击托盘图标直接打开控制台。
* **仓库同步 HTTP 代理支持 (New)**：仓库同步任务全新支持配置 HTTP 代理，方便在网络限制环境下流畅同步 Git 远程代码库（[#164](https://github.com/engigu/baihu-panel/pull/164)）。
* **首页控制台按今日筛选任务日志 (New)**：首页 Dashboard 日志列表新增“按今日”时间维度快速筛选日志功能（[#23](https://github.com/engigu/baihu-panel/issues/23)）。

**✨ 修复与改进**
* **Web 终端断开锁屏与重连修复 (Fix)**：修复了点击终端右上角刷新按钮重连时由于旧连接异步关闭事件导致提示“连接已断开”误覆盖的 Bug；并在连接中断时自动锁定终端键盘输入（`disableStdin = true`）防盲打。
* **站点配置默认值回退修复 (Fix)**：修复了站点设置项在值为零/空时无法正常回退到系统默认默认值的 Bug，并重构清理了冗余硬编码。

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

**方式一：使用 GUI 安装包安装（推荐）**

直接运行附件中的安装程序 `BaihuPanel-Setup-v1.1.29-windows-amd64.exe` 完成安装，程序将自动完成桌面快捷方式创建、开机自启配置与 `127.0.0.1 baihu.local` 域名绑定。安装完成后通过桌面图标或任务栏右下角托盘图标即可直接打开 `http://baihu.local:38052`。

---

**方式二：二进制单文件运行（免安装/便携版）**

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

解压下载好的 `.zip` 压缩包（如 `baihu-windows-amd64.zip`），进入解压目录并在 PowerShell 中运行：

```powershell
.\baihu.exe server
```

---

**访问面板：**
* 启动后访问：`http://baihu.local:38052` 或 `http://localhost:38052` (或单文件默认端口 `http://localhost:8052`)
* **默认账号**：用户名 `admin`，密码见面板首次启动时的控制台日志。
