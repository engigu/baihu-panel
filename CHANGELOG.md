# 更新日志 (v1.1.24)

### 2026.07.31 - 标签管理、编辑器多语言高亮、登录两步验证与响应式布局优化

🎉 **新增与优化**
* **全新标签管理功能 (New)**：新增了独立的“标签管理”后台服务及前端控制页，支持对系统内任务标签（`task_tag`）与环境变量标签（`env_tag`）进行创建、重命名、删除、分页、过滤等操作，并引入了标签名称的全局排重机制。
* **Monaco 多语言语法高亮支持 (New)**：编辑器组件扩展支持了 Python, JavaScript/TypeScript, Go, Shell/Bash 等多种主流语言的开箱即用语法高亮，优化了编辑器编码体验。
* **登录 2FA/OTP 二步验证 (New)**：系统全新集成了两步验证（OTP/2FA），支持管理员账号在系统设置页绑定和启用/停用 OTP，并配套重构了登录界面、OTP 验证码拦截以及安全防爆破拦截。同时在后台重构并平坦化了 OTP 设置页的移动端自适应排版。
* **极简图标与响应式布局升级**：重新设计了标签类型标识，舍弃了传统的徽章卡片背景 and 文字，直接以蓝色终端图标（任务）和橙色变量图标（变量）做纯视觉区分；针对移动端/小屏模式重构优化了控制工具栏、按钮的排列换行方式；并统筹统一了“环境变量”及“定时任务”页面移动端卡片底栏网格动作按钮的间距和贴边排版。
* **互联管理控制栏优化**：修复了“互联管理”页面在小屏宽度下由于没有换行导致“同步”页签按钮被半截截断溢出的 UI Bug；设计了两行自适应移动端布局，实现了搜索输入框、刷新按钮与各操作页签的上下视觉平衡，且在切换至无搜索框的“同步”标签时，自动收缩为空行以避免排版漏洞。

**✨ 修复与改进**
* **修复登录接口多重 JSON 响应 Bug**：修复了登录校验失败时后端可能同时吐出多个 JSON 结构体导致的 JSON 序列化损坏及前端无响应 Bug，完善了前端错误捕获提醒。
* **其他细节修复与依赖升级**：修复了推送日志清理按钮偶尔无响应的问题（#23）；升级了多处有安全漏洞的 npm 依赖项以消除 Dependabot 警报。

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


