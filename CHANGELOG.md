# 更新日志 (v1.6.1)

### 2026.11.04 - 旧备份两步验证死锁自愈修复、分页空数据白屏异常修复、依赖安全漏洞加固与运行时组件升级

🎉 **新增功能与体验升级**
* **旧备份两步验证破损自愈与数据清洗 (Auth & 2FA & Fix)**：
  - **登录自动自愈**：优化两步验证触发判断，仅在明确开启且密钥有效时要求输入动态验证码；针对旧版本（如 v1.1.25）备份导出缺失密钥造成的破损脏状态，自动自愈重置两步验证状态并允许密码登录，解决输入验证码提示“未开启两步验证”并卡死的死锁缺陷（[#231](https://github.com/engigu/baihu-panel/issues/231)）；
  - **备份恢复清洗**：恢复用户数据时若检测到密钥为空，自动清洗重置两步验证状态，杜绝历史备份还原污染数据库；
  - **启动自检迁移与 CLI 救砖**：新增系统启动自动纠偏迁移，修复存量脏数据；增强 `baihu resetpwd` 命令行工具，在重置密码时同步解除两步验证绑定。
* **分页空数据返回统一处理与页面白屏修复 (Web & Components & Fix)**：
  - **列表空数据健壮性加固**：针对后端空列表返回 `data: null` 导致的页面 `data.records` 解构报错白屏问题，在 `usePaginatedList` 中统一加入兜底与空列表兼容（[#206](https://github.com/engigu/baihu-panel/issues/206)）；
  - **全站关键页面重构接入**：对标签管理（`Tags.vue`）、环境变量（`Environments.vue`）、执行历史（`History.vue`）以及 4 个日志 Tab（登录日志、调度日志、通知日志、过滤日志）全面接入统一分页组件，彻底根治白屏缺陷并精简样板代码。
* **安全漏洞加固与依赖版本升级 (Deps & Security)**：
  - **后端 Go JOSE DoS 漏洞修复**：升级 `github.com/go-jose/go-jose/v4` 至 `v4.1.8`，修复解析恶意 RSA JWK 时的拒绝服务漏洞（[#118](https://github.com/engigu/baihu-panel/security/dependabot/118) / CVE-2026-116587）；
  - **前端依赖安全漏洞修复**：通过 package.json 依赖覆盖修复 `minimatch` DoS 漏洞（[#121](https://github.com/engigu/baihu-panel/security/dependabot/121) / CVE-2026-118764）与 `deepmerge` 原型污染漏洞（[#119](https://github.com/engigu/baihu-panel/security/dependabot/119) / CVE-2026-93753），并同步升级 `glob` 至安全版本；
  - **核心依赖升级**：同步升级 `silenceper/wechat/v2` (2.1.14)、`golang.org/x/net` (0.62.0)、`gofrs/flock` (0.13.1)、`klauspost/compress` (1.20.3)、`gorm.io/driver/postgres` (1.6.3)、`vue` (3.5.44) 等。

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

  直接运行附件中的安装程序 `BaihuPanel-Setup-v1.6.1-windows-amd64.exe` 完成安装，程序将自动完成桌面快捷方式创建、开机自启配置与 `127.0.0.1 baihu.local` 域名绑定。安装完成后通过桌面图标或任务栏右下角托盘图标即可直接打开 `http://baihu.local:38052`。

* **方式二：二进制单文件运行（免安装/便携版）**

  解压下载好的 `.zip` 压缩包（如 `baihu-windows-amd64.zip`），进入解压目录并在 PowerShell 中运行：

  ```powershell
  .\baihu.exe server
  ```

---

**访问面板：**
* 启动后访问：`http://baihu.local:38052` 或 `http://localhost:38052` (或单文件默认端口 `http://localhost:8052`)
* **默认账号**：用户名 `admin`，密码见面板首次启动时的控制台日志。
