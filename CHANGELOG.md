# 更新日志 (v1.1.28)

### 2026.09.04 - IDE 底部 Terminal 控制台、运行环境持久化、任务配置批量更新与 Git 过滤优化

🎉 **新增与优化**
* **VSCode 沉浸式 Terminal 控制台 (New)**：将代码编辑器的运行终端由中心模态弹窗改造为 VSCode 风格的底部抽屉控制台，实现代码编辑与终端输出同屏协同。提供【重新运行】、【清空控制台】、【最大化/还原】及【关闭】控制按钮，并优化了初始命令输出的格式与回车体验。
* **运行环境持久化与直接执行 (New)**：运行配置（Mise 语言及版本）将自动持久化到本地存储。首次配置后，后续点击【运行】即可直接启动底部 Terminal 执行，无需频繁确认弹窗；同时在编辑器头部与 Terminal 工具栏保留了快捷配置按钮。
* **任务与仓库批量配置修改 (New)**：任务列表与仓库列表新增“批量修改配置”功能，采用弹窗内下拉 + 搜索多选形式，支持批量选择配置运行环境和高级选项，未选字段保持原配置不覆盖。

**✨ 修复与改进**
* **仓库同步 Git 跟踪文件过滤优化 (Fix)**：优化 `reposync` 同步筛选逻辑，只自动过滤和清理受 Git 管理跟踪的文件，非 Git 跟踪及本地生成文件不受影响。
* **依赖安全漏洞升级 (Security)**：升级了 `kin-openapi` (v0.149.0) 及多个前端 NPM 依赖，修复了 CVE 漏洞与 Dependabot 安全警告。

> 💡 **提示**：出于安全及环境隔离考虑，推荐使用 Docker/Compose 部署方式。[镜像地址](https://github.com/engigu/baihu-panel/pkgs/container/baihu)

### 🐳 方式一：Docker 部署 (推荐)
[部署文档](https://github.com/engigu/baihu-panel?tab=readme-ov-file#%E5%BF%AB%E9%80%9F%E9%83%A8%E7%BD%B2)

---

### 🚀 方式二：单文件部署 (Linux / Windows)
从当前 Release 的附件中下载对应架构和平台的部署压缩包（Linux 为 `.tar.gz`，Windows 为 `.zip`）。

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

解压下载好的 `.zip` 压缩包，进入解压目录并打开 PowerShell，运行：

```powershell
.\baihu.exe server
```

---

**访问面板：**
* 启动后访问：`http://localhost:8052`
* **默认账号**：用户名 `admin`，密码见面板首次启动时的控制台日志。
