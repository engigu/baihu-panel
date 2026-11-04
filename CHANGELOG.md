# 更新日志 (v1.6.2)

### 2026.11.04 - 单点登录第三方身份与本地账号绑定管理、标签管理跨分类重名误判修复与文档构建依赖加固

🎉 **新增功能与体验升级**
* **单点登录第三方身份与本地账号绑定管理 (Auth & SSO & Feature)**：
  - **多身份源独立映射模型**：新增 `sys_user_sso_bindings` 独立绑定表（使用 20 位 xid），解耦本地账号与第三方认证身份，支持单个本地系统账号关联多个第三方身份源（OIDC、GitHub、Gitee 等）（[#232](https://github.com/engigu/baihu-panel/issues/232)）；
  - **存量平滑迁移与自愈**：启动时自动扫描迁移历史 `users` 表中的 `oidc_subject` 字段，存量老用户无需重新绑定即可无感平滑登录；
  - **历史多余账号合并与数据过户**：自动识别清理早期 SSO 自动建号机制遗留的多余账号，将关联业务数据、创建者/修改者审计字段与第三方凭据平滑过户至本地主账号（[#232](https://github.com/engigu/baihu-panel/issues/232)）；
  - **一键授权绑定与解绑**：登录状态下可在设置页面通过已启用的 SSO 提供方一键跳转授权，完成与当前登录本地账号的绑定映射；支持随时快捷解绑并提供二次确认安全弹窗；
  - **未绑定安全拦截**：移除弱安全性的静默自动建号逻辑，未预先绑定的第三方身份访问时重定向回登录页并进行友好 Toast 报错提示；
  - **轻量化 UI 与全屏自适应**：安全设置中 SSO 模块重构为轻量级 Tab 分离结构（「已绑定账号列表」与「通道接入与参数配置」），支持外层统一折叠、移动端卡片流及桌面端 100% 宽度对称铺满排版。
* **标签管理跨分类重名校验隔离修复 (Tags & Fix)**：
  - **分类限定查重**：修复在标签管理页面手工新建或编辑标签时，因全表全局查重导致的跨分类误判拦截问题，将查重范围严格限定在同分类（`task_tag` vs `env_tag`）内，允许任务标签与环境变量标签使用相同名称（[#233](https://github.com/engigu/baihu-panel/issues/233)）。
* **文档构建依赖优化 (Docs & Fix)**：
  - **依赖覆盖配置清理**：移除过期的 `@ai-sdk/provider-utils` 依赖覆盖项，恢复 VitePress 文档流水线正常编译构建。

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

  直接运行附件中的安装程序 `BaihuPanel-Setup-v1.6.2-windows-amd64.exe` 完成安装，程序将自动完成桌面快捷方式创建、开机自启配置与 `127.0.0.1 baihu.local` 域名绑定。安装完成后通过桌面图标或任务栏右下角托盘图标即可直接打开 `http://baihu.local:38052`。

* **方式二：二进制单文件运行（免安装/便携版）**

  解压下载好的 `.zip` 压缩包（如 `baihu-windows-amd64.zip`），进入解压目录并在 PowerShell 中运行：

  ```powershell
  .\baihu.exe server
  ```

---

**访问面板：**
* 启动后访问：`http://baihu.local:38052` 或 `http://localhost:38052` (或单文件默认端口 `http://localhost:8052`)
* **默认账号**：用户名 `admin`，密码见面板首次启动时的控制台日志。
