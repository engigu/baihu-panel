# 命令行工具 (CLI)

白虎面板在环境内内置了同名的 `baihu` 命令行工具。如果您在终端内需要执行系统级别的操作，可以使用这些内置命令。

## 常用核心指令

| 命令 | 描述 |
| :--- | :--- |
| `baihu server` | 面板启动指令，运行服务端后台进程。 |
| `baihu reposync` | 供定时任务调用，将远程 Git 仓库的高级特性同步到本地目录中。 |
| `baihu resetpwd` | 交互式重置系统 admin 账号密码（密码丢失时可通过进入终端重置）。 |
| `baihu restore <file>` | 使用本地的 .zip 备份压缩包文件，一条命令直接全量恢复系统数据。 |

---

## 使用场景示例

### 1. 密码重置
您可以进入 Docker 容器或通过 ssh 连入宿主机控制台：
```bash
docker exec -it baihu baihu resetpwd
```
然后根据提示，输入新的管理员密码即可重置成功。

### 2. 手动启动
如果是通过手动部署二进制文件，可以使用 `baihu server` 启动：
```bash
nohup ./baihu server > /dev/null 2>&1 &
```

### 3. 数据恢复
上传备份后的 ZIP 文件至容器目录：
```bash
docker exec -it baihu baihu restore /app/data/backup-2026xxxx.zip
```
该操作会全量覆盖现有数据库和脚本文件，请谨慎操作。

---

## 其他帮助
终端内直接执行 `baihu` 即可在控制台直接打印内置支持详细说明和命令列表参数。
