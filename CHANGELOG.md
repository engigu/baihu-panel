> 出于安全及环境隔离考虑，推荐使用 Docker/Compose 部署方式。[镜像地址](https://github.com/engigu/baihu-panel/pkgs/container/baihu)

## ⚠️ 重大升级与迁移说明 (v3 数据结构)
本次更新包含底层数据结构的重大突破，将所有数据的 ID 类型从数字编号平滑迁移为20位的字符式全局唯一标识符（`xid`）。系统在启动时会**自动进行数据的清洗、映射、拷贝与外键修补**，以确保旧数据被妥善对接。
- **备份位置**：执行迁移前，即使有程序自动转换逻辑（data\migration_v3_backup_backup_xxx.zip），为了数据安全，仍然建议您**提前手动进行备份**。系统的默认 SQLite 数据库文件通常位于配置的 `data/` 目录中。
- **如果遇到失败**：由于不同用户原本的数据和环境复杂度存在差异，如果遇到未预期的迁移失败或数据显示丢失，**请使用原本备份的数据库** 并 **降级至 `v1.0.10` 及以下旧版本** 进行恢复与使用。
- **全新启用**：对部分希望拥抱新数据结构的用户而言，也可以根据情况选择在新版本中直接重新建立配置。
- **致谢与展望**：这次数据结构级的大幅重构，主要是为了后续项目功能扩展（含分布式管理、多租户隔离、数据同步等）的底层根基准备，再不改以后改不动了。给大家带来的使用不便敬请见谅，感谢支持！

## 快速部署

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


