# 更新日志 (v1.1.20)

### 2026.07.13 - 日志 ZSTD 压缩、依赖补全与计划任务排序

🎉 **新增与优化**
* **日志 ZSTD 压缩升级**：日志流式压缩机制由 zlib 全面升级至 ZSTD，显著降低磁盘开销与带宽占用；前端集成 `fzstd` 无缝支持新格式解码，并实现了对旧版 zlib 的向后兼容；针对小于 128 字节的短日志自动绕过压缩，避免资源浪费。
* **依赖自动补全与交互终端 (#147)**：全新上线依赖分析与自动补全安装 CLI，并提供终端安装引导；在定时任务日志页支持通过“补全依赖”一键调出内嵌终端进行交互式依赖补全。
* **计划任务表头排序 (#148)**：大屏与中屏布局支持点击“名称”、“执行时间”（下次执行时间）和“状态”表头进行排序，后端支持 `sort_by` 与 `order` 传参并保证置顶任务最高优先级；移动端顶栏同步新增了“排序规则”下拉选择器。
* **过滤视图联动**：自定义过滤视图功能全面支持排序参数（`sort_by` / `order`）联动保存，应用视图时自动还原当时的排序配置。
* **全局 ESC 关闭弹窗**：底层通用 Dialog 组件集成了非侵入式全局 Escape 按键拦截机制，优先退出最顶层弹窗，避免输入框/Monaco等组件焦点被抢占时 ESC 失效。

**✨ 修复与改进**
* **样式与体验**：将大屏及中屏下的状态列宽度由 `w-8` 扩大至 `w-14`，彻底消除因加入排序图标导致的文字折行与表头挤压；在 DialogContent 上追加了聚焦样式清除，消除了窗口边缘的白色聚焦边框；为新建任务的日志清理配置默认设置为保留最近 30 条记录，防爆盘；日志详情弹窗增加了最大高度及滚动条优化。

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


