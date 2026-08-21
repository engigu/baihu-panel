# 更新日志 (v1.1.27)

### 2026.08.21 - 通知过滤与小屏适配、文件查看下载优化与 CLI 命令行文档增强

🎉 **新增与优化**
* **通知过滤与响应式优化 (New)**：新增全局与任务级的通知通道过滤选项，并对中小屏幕/移动端的页面样式及响应式适配进行了细节微调与体验优化。
* **文件查看与下载体验提升 (New)**：重构了编辑器中二进制文件与图片文件的处理逻辑，优化了其在前端的加载与直接下载操作，避免查看特殊非文本文件时引发的网页卡顿及浏览器强制下载弹窗问题。
* **CLI 命令行文档增强 (Docs)**：充实了 CLI 实用的参数用例说明，明确规范了脚本文件的元数据头部注释格式（Script Header Comments），方便自动提取和注册定时任务。

**✨ 修复与改进**
* **Web 编译与构建修复 (Fix)**：清除了 Web 前端多余未使用的 `ShieldAlert` 图标引用，彻底解决因 TS 类型检查严格而导致的构建打包失败报错。

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
