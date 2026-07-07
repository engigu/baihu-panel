# 更新日志 (v1.1.19)

### 2026.07.07 - 性能优化与体验升级

🎉 **新增与优化**
* **日志系统升级**：实时日志查看由 WebSocket 迁移至 SSE（Server-Sent Events），配合 JSON 格式封装，提升了日志传输的稳定性和解析效率。
* **前端性能优化**：优化了 API 轮询频率，并支持根据页面可见性（Page Visibility）自动控制轮询，有效降低资源占用。
* **UI 体验提升**：引入了全新的 `StatusBadge` 状态徽章组件，并已应用于日志详情卡片，使状态展示更加直观美观；优化了主布局在宽屏（大屏）下的菜单光标交互体验。
* **任务管理**：任务列表现已支持设置默认视图（Default View），提升个性化操作体验。
* **文档与生态**：新增了节点互联功能文档；上线了基于 GHCR 的包下载量趋势图表页面。

**✨ 修复与改进**
* **任务调度**：修复了任务取消时可能出现的进程树泄漏问题，补充了取消事件的触发逻辑。
* **通知与邮件**：修复了纯文本邮件发送时未正确使用 `text/plain` 的问题；修复了通知前缀更新 Bug (#144)。
* **系统构建**：修复了 Release 版本更新相关问题及部分构建错误。



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


