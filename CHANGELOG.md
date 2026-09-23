# 更新日志 (v1.3.0)

### 2026.09.23 - 机密端到端安全解密查看 (ECDH+AES-GCM)、日志表冗余任务名称、应用全量变量注入与文档体验升级

🎉 **新增功能与架构改进**
* **现代椭圆曲线 ECDH + AES-GCM 机密端到端安全解密查看 (Security & Feature)**：
  - **端到端加密传输**：前端动态生成 ECDH (P-256) 临时密钥对，后端利用 ECDH 密钥交换算法派生会话密钥，并通过 AES-GCM 加密回传机密明文，网络通信链路全流程绝无明文凭证，杜绝抓包截获与中间人窃密风险（[#171](https://github.com/engigu/baihu-panel/pull/171)）；
  - **二级凭据安全鉴权**：解密查看前强制触发登录密码二次核验，弹窗表单进行域隔离，彻底杜绝浏览器密码管理器自动填充污染背景其他表单；
  - **可视交互与定时遮罩保护**：支持一键快速复制明文与便捷显隐切换，离开视口或超时后自动恢复星号遮罩保护。
* **日志表冗余任务名称与已删除任务状态追溯 (Feature & Perf)**：
  - **数据库字段冗余设计**：在 `baihu_task_logs` 数据表中持久化冗余 `task_name` 字段，彻底解决定时任务删除或更新后历史日志呈现空名称、不可读 ID 的痛点（[#170](https://github.com/engigu/baihu-panel/pull/170)）；
  - **生命周期清晰辨识**：前端日志列表智能识别已被物理删除的任务，并在任务名前醒目标注 `(已删除)` 状态徽章；
  - **检索性能显著提升**：支持在执行历史中直接按原任务名称精准/模糊查询历史归档日志，不再强依赖连表实时解析。
* **声明式应用全量环境变量注入规范强化 (Architecture & Fix)**：
  - **统一调度策略规范**：受控子任务（Child Tasks）严格遵循白虎面板调度规范，在 `UnifiedConfig` 中显式设置 `common.task_all_envs = true` 与 `task_concurrency = 0`，使应用编排出的全部子任务默认无缝获得全量环境变量与解密机密注入；
  - **杜绝 ID 冗余绑定**：移除向子任务 `Envs` 字段写入具体 ID 字符串的多余逻辑，保持 `tasks` 与 `DataRelation` 表的高性能与轻量解耦。

✨ **文档增强与体验优化**
* **在线动态加载与渲染应用商店文档 (Docs)**：
  - 文档中心全新支持免跨域实时拉取与动态解析渲染 GitHub 远端应用商店 `README.md` 与生态规范文档；
  - 远程 Markdown 渲染引擎全面支持 Monokai 代码高亮配色、目录锚点平滑滚动跳转以及代码块右上角一键复制反馈。
* **移动端响应式排版优化 (Style)**：
  - 应用市场顶部控制栏响应式重构，优化在移动端小屏/窄屏视口下的搜索框、刷新按钮与分类标签排版，彻底解决小屏排版溢出与换行挤压问题。

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

  直接运行附件中的安装程序 `BaihuPanel-Setup-v1.3.0-windows-amd64.exe` 完成安装，程序将自动完成桌面快捷方式创建、开机自启配置与 `127.0.0.1 baihu.local` 域名绑定。安装完成后通过桌面图标或任务栏右下角托盘图标即可直接打开 `http://baihu.local:38052`。

* **方式二：二进制单文件运行（免安装/便携版）**

  解压下载好的 `.zip` 压缩包（如 `baihu-windows-amd64.zip`），进入解压目录并在 PowerShell 中运行：

  ```powershell
  .\baihu.exe server
  ```

---

**访问面板：**
* 启动后访问：`http://baihu.local:38052` 或 `http://localhost:38052` (或单文件默认端口 `http://localhost:8052`)
* **默认账号**：用户名 `admin`，密码见面板首次启动时的控制台日志。
