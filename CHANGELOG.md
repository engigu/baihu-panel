# 更新日志 (v1.1.16)

### 2026.06.22 - 体验优化与 Bug 修复
* **执行历史自适应**：修复了大屏或竖屏下页面高度被限制在 520px 导致底部留白的问题，动态计算视口高度，让列表和日志输出区域自伸缩（#134）。
* **通知日志截断优化**：修复了执行日志在全局发送前被 1000 字符强行截断导致自定义 LogLimit 失效的 Bug（#133）。
* **通知发送性能优化**：将 `stripAnsi` 清除控制字符逻辑移到发送循环外只执行一次，避免高开销正则在多渠道分发时重复运算（#133）。
* **机密使用说明**：在机密管理页面新增了温馨提示及使用说明，并支持点击关闭和 localStorage 持久化记忆状态（#135）。
* **系统命令与兼容性**：新增了版本输出命令（`version`），修复了终端 UTF-8 字节截断时的字符乱码问题。
* **数据恢复兼容**：在数据恢复时，自动将老的 task 环境和标签迁移至关系表（#135）。
* **仓库同步增强**：支持了自定义仓库同步文件夹名称（#132）。
* **Docker 支持**：在 CI 构建中，增加 arm64 架构支持，并在非 Release 构建中只为分支打 Tag。

---

> 出于安全及环境隔离考虑，推荐使用 Docker/Compose 部署方式。[镜像地址](https://github.com/engigu/baihu-panel/pkgs/container/baihu)



### 🐳 方式一：Docker 部署（推荐）
[部署文档](https://github.com/engigu/baihu-panel?tab=readme-ov-file#%E5%BF%AB%E9%80%9F%E9%83%A8%E7%BD%B2)

### 🚀 方式二：单文件部署
从当前 Release 的附件中下载对应架构的部署压缩包（如 `baihu-linux-amd64.tar.gz`），然后使用以下命令提取并运行：

**⚠️ 重要前置依赖：手动安装 `mise`**
单文件直接运行依赖宿主机系统环境，请务必先安装 [mise](https://mise.jdx.dev/getting-started.html) 供任务调度及环境管理使用：
```bash
curl https://mise.run | sh
export PATH="~/.local/share/mise/bin:~/.local/share/mise/shims:$PATH"
```

**运行面板：**
```bash
tar -xzvf baihu-linux-amd64.tar.gz
chmod +x baihu-linux-amd64
./baihu-linux-amd64 server
```

---

**访问面板：**
启动后访问：http://localhost:8052

**登录信息：**
默认账号：用户名 `admin`，密码见面板首次启动时的控制台日志。


