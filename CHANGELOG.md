# 更新日志 (v1.4.0)

### 2026.09.29 - RESTful 状态码与业务码双轨对齐、Bark 端到端加密推送、任务深层目录展开、安全防护与体验全面升级

🎉 **新增功能与架构改进**
* **HTTP 状态码与业务码双轨对齐规范 (Architecture & API)**：
  - **双轨标准化对齐**：全面重构系统响应信封，彻底消除冗余嵌套，统一收敛至 7 大核心标准 HTTP 状态码（200 OK、201 Created、204 No Content、400 Bad Request、401 Unauthorized、403 Forbidden、500 Internal Server Error），使得 HTTP 层面的传输状态与 JSON 信封内的业务业务码严格一致（[#174](https://github.com/engigu/baihu-panel/pull/174)）；
  - **清晰规范的错误回传**：统一错误响应结构，完善错误捕获并透出真实底层原因，极大增强 OpenAPI 及前端数据消费的稳健性。
* **Bark 推送端到端加密与动态表单能力增强 (Feature)**：
  - **现代高强度加密通道**：Bark 渠道全新支持基于 AES 算法的端到端数据传输加密，兼容 ECB、CBC 与现代 GCM 认证加密模式，支持自动生成高强度随机 IV，杜绝消息明文被任何网络中间节点监听窃视（[#177](https://github.com/engigu/baihu-panel/pull/177)）；
  - **通知表单动态组件扩充**：通知渠道配置表单支持 `Switch` 开关和 `Select` 下拉选择控件，为渠道参数定制提供更优雅的用户交互体验。
* **演示模式安全防护增强 (Security)**：
  - **备份恢复严防恶意覆写**：在演示模式下（`BH_DEMO_MODE=true`）后端与前端全面拦截并禁用“数据恢复”功能，前端增加醒目标签与风险提示，杜绝公共演示站点被恶意篡改或覆盖系统快照数据。
* **历史明文机密全量脱敏保护与安全加固 (Security)**：
  - **日志防泄露兜底**：系统将历史未加密明文机密统一自动纳入脱敏安全词典，在任务执行输出、调度历史与推送通知中自动打码遮罩，杜绝凭据无意泄露（[#175](https://github.com/engigu/baihu-panel/pull/175)）；
  - **管理员防重复初始化**：当系统已存在 `role=admin` 账号时自动跳过管理员初始化逻辑，避免修改默认用户名后被恶意重置（[#172](https://github.com/engigu/baihu-panel/pull/172)）；
  - **禁用环境变量严密隔离**：任务调度执行引擎在注入环境变量时，对处于“禁用”状态的变量彻底跳过注入，确保禁用即生效。

✨ **交互体验优化与问题修复**
* **任务编辑目录层级深层自动展开 (UI & Fix)**：
  - 任务编辑抽屉中的目录树选择器（`DirTreeSelect.vue`）支持依据当前绑定路径自动展开所有上级父目录，彻底解决深层目录无法自动展开、回显被遮挡的痛点（[#178](https://github.com/engigu/baihu-panel/pull/178)）；
  - 移除了武断的绝对路径末尾匹配逻辑，保障各种层级下的工作目录均能精准高亮定位。
* **任务编辑通知时机选择修复 (Fix)**：
  - 修复任务编辑弹窗中勾选通知时机（成功、失败、结束）时状态回显异常的问题（[#176](https://github.com/engigu/baihu-panel/pull/176)）。
* **多层级脚本目录长名称显示优化 (Style)**：
  - 解决左侧脚本文件目录在深度嵌套场景下长名称被过度截断挤压的问题，保证目录结构清晰易辨。
* **Windows 系统托盘自动静默更新与工作集优化 (Tray & Perf)**：
  - Windows 系统托盘程序支持全量安装包自动静默下载与平滑覆盖升级，并大幅精简后台工作集内存开销。
* **时间排版与核心指标排版体验 (Style)**：
  - 全局优化时间显示友好度，控制面板核心指标数字采用 Inter 字体并配合 `tabular-nums` 表格式特性，彻底消除数值变动时的抖动。

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

  直接运行附件中的安装程序 `BaihuPanel-Setup-v1.4.0-windows-amd64.exe` 完成安装，程序将自动完成桌面快捷方式创建、开机自启配置与 `127.0.0.1 baihu.local` 域名绑定。安装完成后通过桌面图标或任务栏右下角托盘图标即可直接打开 `http://baihu.local:38052`。

* **方式二：二进制单文件运行（免安装/便携版）**

  解压下载好的 `.zip` 压缩包（如 `baihu-windows-amd64.zip`），进入解压目录并在 PowerShell 中运行：

  ```powershell
  .\baihu.exe server
  ```

---

**访问面板：**
* 启动后访问：`http://baihu.local:38052` 或 `http://localhost:38052` (或单文件默认端口 `http://localhost:8052`)
* **默认账号**：用户名 `admin`，密码见面板首次启动时的控制台日志。
