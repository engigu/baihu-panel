# 更新日志 (v1.2.0)

### 2026.09.22 - 全新声明式应用与官方应用市场、2FA 备份恢复修复、任务状态筛选与跨平台路径引擎升级

🎉 **新增功能与架构重构**
* **全新声明式应用引擎与官方应用市场 (Major Feature)**：
  - **官方应用市场 (AppStore)**：新增“应用市场”专属页面，无缝直连聚合生态仓库 `baihu-appstore`，可一览社区精选应用、版本、作者及构建时间戳，支持一键可视化参数配置与极速秒级部署；
  - **白虎声明式应用规范 v1 (Declarative App Specification)**：制定标准 YAML 应用清单规范，集成代码源同步 (`sources`)、跨平台依赖探测与预编译 (`setup.check` / `setup.install`)、环境变量契约声明 (`env_schema`)、任务编排映射 (`tasks`) 及运行场景模式 (`scenarios`)；
  - **主应用实体与受控子任务解耦架构 (Master-Child Architecture)**：创新采用 “1 个主应用实体 (Type='app') + N 个受控子任务 (Type='task')” 解耦模型统筹落库于 `tasks` 表，主任务统一集中管理场景、环境变量与生命周期，子任务独立执行调度，卸载与重配支持全自动级联清理；
  - **单一数据源与配置实时双向同步 (Single Source of Truth)**：运行时多语言与分类标签全面收敛至 `template: { tag, languages }` 强类型契约，无缝挂接 `mise` 运行时，用户在面板定制的参数实时写回应用 YAML 清单，保持数据严密一致；
  - **场景模式一键选配与调度预设继承**：支持多种运行场景模式（如标准模式、佛系保级、极简自检等），随场景受控启停子任务并覆盖定制 Cron；主应用任务支持自动继承 `schedule_opts` 定时规则预设；
  - **交互式环境变量表单与密文安全**：基于契约自动生成可视配置表单（文本/密文/下拉单选/开关），支持 AES 加密入库与 Secret 密文自动恢复，支持“覆盖已有环境变量”策略开关；
  - **应用热度统计支持**：集成无感知应用浏览量与安装计数异步上报，助力社区生态应用体验沉淀。
* **定时任务启用状态筛选 (Feature)**：
  - 定时任务列表新增按“全部 / 已启用 / 已禁用”状态一键筛选过滤，在大批量任务场景下查找和维护更加快捷从容。

✨ **修复与改进**
* **2FA 两步验证备份恢复失效修复 (Fix)**：
  - 允许 User 模型的 `OtpSecret` 与 `TokenVersion` 在备份时导出序列化，彻底修复从备份恢复后因缺失 OTP 密钥导致 2FA 验证失效无法登录的严重缺陷（[#169](https://github.com/engigu/baihu-panel/issues/169)）。
* **跨平台路径归一化与命令解析强化 (Fix & Refactor)**：
  - 引入 `ResolveCommand` 规范化处理命令与脚本路径占位符（`$SCRIPTS_DIR$` 等），彻底消除 Windows 平台下盘符反斜杠转义与路径拼接破坏问题；
  - 清理底层应用引擎废弃的旧实体存库与日志捕获死代码，修复多处循环指针安全隐患，提升运行稳定性。
* **中屏排版与视觉细节优化 (Style)**：
  - 优化定时任务卡片与列表在中等屏幕（平板及窄窗口视口）下的响应式网格布局；微调侧边栏导航项的高度与间距，信息展示更清晰紧凑。
* **安全依赖升级 (Security)**：
  - 升级并修复 `@ai-sdk/provider-utils` 依赖安全漏洞（[#75](https://github.com/engigu/baihu-panel/issues/75)）；
  - 锁定文档项目中 `undici` 版本至 `^7.28.0`，修复安全告警（[#76](https://github.com/engigu/baihu-panel/issues/76), [#77](https://github.com/engigu/baihu-panel/issues/77), [#78](https://github.com/engigu/baihu-panel/issues/78)）。

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

  直接运行附件中的安装程序 `BaihuPanel-Setup-v1.2.0-windows-amd64.exe` 完成安装，程序将自动完成桌面快捷方式创建、开机自启配置与 `127.0.0.1 baihu.local` 域名绑定。安装完成后通过桌面图标或任务栏右下角托盘图标即可直接打开 `http://baihu.local:38052`。

* **方式二：二进制单文件运行（免安装/便携版）**

  解压下载好的 `.zip` 压缩包（如 `baihu-windows-amd64.zip`），进入解压目录并在 PowerShell 中运行：

  ```powershell
  .\baihu.exe server
  ```

---

**访问面板：**
* 启动后访问：`http://baihu.local:38052` 或 `http://localhost:38052` (或单文件默认端口 `http://localhost:8052`)
* **默认账号**：用户名 `admin`，密码见面板首次启动时的控制台日志。
