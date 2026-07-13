# 更新日志 (v1.1.20)

### 2026.07.13 - 计划任务排序与全局 ESC 退出优化

🎉 **新增与优化**
* **计划任务排序**：大屏与中屏布局支持点击“名称”、“执行时间”（下次执行时间）、“状态”表头进行列表排序；后端支持 `sort_by` 和 `order` 查询字段并保持置顶任务最高优先级；小屏/移动端顶栏新增了“排序规则”下拉选择器，体验与大屏逻辑无缝统一。
* **过滤视图联动**：自定义过滤视图功能全面支持排序联动保存，在应用视图时自动还原当时的排序配置。
* **全局 ESC 关闭弹窗**：在底层通用 Dialog 组件内集成了非侵入式全局 Escape 按键拦截机制，优先对最顶层弹窗执行关闭，解决了在输入框聚焦或 Monaco/终端组件内 ESC 失效的问题。

**✨ 修复与改进**
* **样式与体验**：将大屏及中屏下的状态列宽度由 `w-8` 扩大至 `w-14`，彻底消除因加入排序图标导致的文字折行与表头挤压；在 DialogContent 上追加了聚焦样式清除，消除了窗口边缘的白色聚焦边框。

---



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


