# 更新日志 (v1.1.30)

### 2026.09.11 - 标签关联资源穿透与跳转、输入法体验修复、调度引擎稳定性调优与安全升级

🎉 **新增与优化**
* **标签关联资源穿透与一键跳转 (New)**：在“标签管理”页面点击关联资源计数即可弹窗查看该标签关联的所有任务与环境变量明细，并支持直接点击资源名称快捷跳转至对应管理页进行编辑。

✨ **修复与改进**
* **标签输入组件末尾字符丢失修复 (Fix)**：修复了前端 `TagInput.vue` 组件在按下回车确认添加标签时，输入法缓冲区末尾字符意外丢失的问题。
* **任务结束时序与实时日志竞态修复 (Fix)**：优化任务执行结束时的日志压缩与状态入库时序，修复超短任务及高频并发下实时日志流与状态落库存在竞态导致前端显示不同步的问题。
* **Cron 注册数据库查询性能调优 (Perf)**：优化仓库同步任务在 Cron 注册阶段的数据库查询开销，提升调度器整体性能并适度放宽内部通信超时时间。
* **依赖安全漏洞升级 (Security)**：升级 `golang.org/x/crypto` 至 `v0.55.0`，彻底修复已知安全隐患（[#165](https://github.com/engigu/baihu-panel/issues/165)）。

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

  直接运行附件中的安装程序 `BaihuPanel-Setup-v1.1.29-windows-amd64.exe` 完成安装，程序将自动完成桌面快捷方式创建、开机自启配置与 `127.0.0.1 baihu.local` 域名绑定。安装完成后通过桌面图标或任务栏右下角托盘图标即可直接打开 `http://baihu.local:38052`。

* **方式二：二进制单文件运行（免安装/便携版）**

  解压下载好的 `.zip` 压缩包（如 `baihu-windows-amd64.zip`），进入解压目录并在 PowerShell 中运行：

  ```powershell
  .\baihu.exe server
  ```

---

**访问面板：**
* 启动后访问：`http://baihu.local:38052` 或 `http://localhost:38052` (或单文件默认端口 `http://localhost:8052`)
* **默认账号**：用户名 `admin`，密码见面板首次启动时的控制台日志。
