# 更新日志 (v1.6.3)

### 2026.11.06 - 语言依赖未指定版本自动探测回填、备份与归档 Zip-Slip 目录遍历安全漏洞修复、动态机密 Upsert 与构建优化

🎉 **新增功能与体验升级**
* **语言依赖未指定版本安装时自动探测并回显版本号 (Deps & Fix)**：
  - **自动版本识别与回填**：在依赖管理页面安装 Python/Node.js 等环境依赖包时，若未手动指定版本，后端在安装成功后自动通过轻量命令（Python 执行 `pip show`，Node.js 执行 `npm list --json`）快速抓取实际安装的具体版本号并回填入库，彻底解决列表中版本显示为 `-` 的问题（[#237](https://github.com/engigu/baihu-panel/issues/237)）；
  - **依赖记录 Upsert 去重修复**：重构依赖记录入库与更新逻辑，按 `(name, language, lang_version)` 识别同环境已有记录执行更新，避免同环境多版本脏数据与重装主键冲突。
* **备份恢复与归档解包安全漏洞防御 (Security & Fix)**：
  - **Zip-Slip 路径逃逸阻断**：提取 `SafeExtractZip` 与 `SafeExtractTar` 通用安全解包方法，严格校验归档内相对路径，杜绝通过恶意压缩包构造 `../` 目录穿越覆盖宿主机关键系统文件（Zip-Slip 漏洞）（[#235](https://github.com/engigu/baihu-panel/issues/235)）；
  - **符号链接劫持防御**：解压时阻断指向目标目录外部的非法软链接与硬链接，防范解压后通过符号链接实施越权文件读写；全面覆盖系统备份恢复、应用中心归档解压与任务同步归档解压等核心场景。
* **动态机密环境变量安全 Upsert 支持 (Env & Feature)**：
  - **动态机密安全更新**：新增机密环境变量根据变量名称及用户作用域的安全 Upsert 能力，强化加密存储与写入校验（[#234](https://github.com/engigu/baihu-panel/pull/234)）。
* **构建优化与 Windows 托盘版本注入修复 (Build & Windows)**：
  - **构建参数统一收拢**：统一编译脚本构建参数，启用 `-trimpath` 剥离二进制中的本地文件绝对路径；
  - **托盘版本修复**：修复 Windows 托盘程序编译时版本号变量注入路径，使托盘程序关于信息与右键菜单正确展示当前面板版本。

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

  直接运行附件中的安装程序 `BaihuPanel-Setup-v1.6.3-windows-amd64.exe` 完成安装，程序将自动完成桌面快捷方式创建、开机自启配置与 `127.0.0.1 baihu.local` 域名绑定。安装完成后通过桌面图标或任务栏右下角托盘图标即可直接打开 `http://baihu.local:38052`。

* **方式二：二进制单文件运行（免安装/便携版）**

  解压下载好的 `.zip` 压缩包（如 `baihu-windows-amd64.zip`），进入解压目录并在 PowerShell 中运行：

  ```powershell
  .\baihu.exe server
  ```

---

**访问面板：**
* 启动后访问：`http://baihu.local:38052` 或 `http://localhost:38052` (或单文件默认端口 `http://localhost:8052`)
* **默认账号**：用户名 `admin`，密码见面板首次启动时的控制台日志。
