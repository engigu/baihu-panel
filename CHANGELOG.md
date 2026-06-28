# 更新日志 (v1.1.17)

### 2026.06.28 - 全新节点互联功能发布！

🎉 **重磅新功能：节点互联体系**
* **互联协议与通道**：新增开放连接（OpenConnect）协议支持，轻松打通并管理多个白虎面板实例，解决分布式定时任务环境隔离痛点。
* **互联控制台**：全新上线的同步管理面板，可实时总览所有连接节点的资源开销、连接状态及各项负载指标。
* **核心数据同步**：支持了跨节点间的环境变量全量同步，并完美保留变量的原始结构与关联 ID，为任务跨环境漂移和无缝分发提供底层保障。
* **路由与代理增强**：底层代理路由机制迎来全面升级，完美支持了基于穿透隧道的前端互联访问代理。

**✨ 其它优化与修复**
* **终端体验优化**：针对移动端深度优化终端（xterm）交互，禁用移动端点击自动弹出软键盘，支持选中内容后 Ctrl+C 快捷复制，并大幅提升了小屏幕下的滑动流畅性。
* **调度器安全**：为 Worker 数量和队列大小添加了边界验证及安全限制，从底层防止高并发场景下出现 OOM 问题。
* **UI 体验优化**：修复了大屏模式下环境变量表格删除按钮丢失的问题；MasterView 按钮在小屏幕下支持自适应填满；增加了演示模式下互联角色的操作限制。

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


