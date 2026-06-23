# 快速部署

项目提供多种基础镜像，默认版本基于 Debian 12，集成了 Python 3.13 与 Node.js 23。

## 基础镜像选择

| 标签 (Tag) | 基础镜像 | 说明 |
| :--- | :--- | :--- |
| `latest` | Debian 12 | **默认推荐**：集成 Python 3.13 与 Node.js 23，开箱即用 |
| `latest-debian13` | Debian 13 | 尝鲜版本，基于 Debian Trixie |
| `latest-minimal` | Debian 13 | **最小化版**：不预置任何语言环境，仅内置 Mise，适合追求极致纯净的用户 |

> **提示**：目前默认使用 `latest` 标签。如需切换环境，只需将镜像名后的 `latest` 替换为 `latest-minimal`（极致纯净）或 `latest-debian13` 即可。

## 环境版本重构说明 (2026.02.13+)

> **警告**：架构升级破坏性变更
> 
> 本版本（2026.02.13+）对底层运行时环境进行了彻底重构，弃用了原有的静态 Python/Node 环境，转为使用 **Mise** 进行动态版本管理。
> 
> 1. **不再提供 Alpine 镜像**：由于 glibc 兼容性问题，Mise 无法在 Alpine 上完美运行，因此暂时取消 Alpine 镜像支持。
> 2. **环境数据不兼容**：如果您是从旧版本升级上来，原有的 Python/Node 环境数据将无法迁移。您需要清空挂载的 `envs/` 目录并让其由新容器自动初始化。

---

## 方式一：Docker 运行 (环境变量配置)

通过环境变量指定配置，简单灵活，适合一般部署。

### SQLite (默认)
```bash
docker run -d \
  --name baihu \
  -p 8052:8052 \
  -v $(pwd)/data:/app/data \
  -v $(pwd)/envs:/app/envs \
  -e TZ=Asia/Shanghai \
  -e BH_SERVER_PORT=8052 \
  -e BH_SERVER_HOST=0.0.0.0 \
  -e BH_DB_TYPE=sqlite \
  -e BH_DB_PATH=/app/data/baihu.db \
  -e BH_DB_TABLE_PREFIX=baihu_ \
  --restart unless-stopped \
  ghcr.io/engigu/baihu:latest
```

### MySQL
```bash
docker run -d \
  --name baihu \
  -p 8052:8052 \
  -v $(pwd)/data:/app/data \
  -v $(pwd)/envs:/app/envs \
  -e TZ=Asia/Shanghai \
  -e BH_SERVER_PORT=8052 \
  -e BH_SERVER_HOST=0.0.0.0 \
  -e BH_DB_TYPE=mysql \
  -e BH_DB_HOST=mysql-server \
  -e BH_DB_PORT=3306 \
  -e BH_DB_USER=root \
  -e BH_DB_PASSWORD=your_password \
  -e BH_DB_NAME=baihu \
  -e BH_DB_TABLE_PREFIX=baihu_ \
  --restart unless-stopped \
  ghcr.io/engigu/baihu:latest
```

---

## 方式二：Docker Compose 部署

推荐的生产环境部署方式。

### 核心部署模板
```yaml
services:
  baihu:
    image: ghcr.io/engigu/baihu:latest
    container_name: baihu
    ports:
      - "8052:8052"
    volumes:
      - ./data:/app/data
      - ./envs:/app/envs
    environment:
      - TZ=Asia/Shanghai
      - BH_SERVER_PORT=8052
      - BH_SERVER_HOST=0.0.0.0
      - BH_DB_TYPE=sqlite
      - BH_DB_PATH=/app/data/baihu.db
      - BH_DB_TABLE_PREFIX=baihu_
      # - BH_SERVER_URL_PREFIX=/baihu  # 可选：配置 URL 前缀用于反向代理
    logging:
      driver: json-file
      options:
        max-size: "10m"
        max-file: "3"
    restart: unless-stopped
```

---

## 方式三：配置文件挂载模式

通过挂载 `/app/configs/config.ini` 来管理详细配置。

### 配置文件挂载示例
```yaml
volumes:
  - ./data:/app/data
  - ./configs:/app/configs
  - ./envs:/app/envs
```
---

## Docker 启动流程

容器启动时 `docker-entrypoint.sh` 会自动执行以下关键步骤：

1. **环境自检**：检查 `/app/data`、`/app/configs`、`/app/envs` 挂载点并创建必要子目录。
2. **Mise 同步**：自动将镜像内置的 Mise 核心运行时激活文件同步到持久化挂载目录中，确保容器重启后环境依然可用。
3. **运行时激活**：动态注入环境变量，将 `mise shims` 路径加入系统 `PATH`。
4. **包管理预设**：自动为 Python 配置清华源 (PIP) 镜像，配置 Node.js 内存限制。
5. **主进程启动**：运行 `baihu server` 开启面板。

> **提示**：通过持久化挂载 `./envs` 目录，您安装的所有运行时版本和第三方依赖库均会永久保留。

---

## 自动更新 Docker 镜像

如果您希望baihu-panel能够自动拉取最新镜像并无感更新，推荐使用 **Watchtower**。Watchtower 会定期检查被监控容器的基础镜像，当发现有新版本推送时，它会自动拉取新镜像、使用与原容器完全相同的配置重启容器。

由于baihu-panel采用持久化挂载（数据和环境都在外部），因此自动更新不会造成任何数据丢失。

### 方式一：独立一行命令运行 Watchtower（推荐）

执行以下命令，Watchtower 将会自动在每天凌晨 3 点自动检查并更新名为 `baihu` 的容器：

```bash
docker run -d \
  --name watchtower \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -e TZ=Asia/Shanghai \
  -e WATCHTOWER_SCHEDULE="0 0 3 * * *" \
  -e WATCHTOWER_CLEANUP=true \
  --restart unless-stopped \
  containrrr/watchtower \
  baihu
```

> **参数说明**：
> - 结尾处的 `baihu` 为指定仅监控更新名为 `baihu` 的容器。若不加此参数，则会自动更新宿主机上所有的 Docker 容器。
> - `WATCHTOWER_CLEANUP=true`：更新成功后自动删除旧版本的废弃镜像，防止存储空间被占满。
> - `WATCHTOWER_SCHEDULE`：设置定时检查的 Cron 表达式（秒 分 时 日 月 周）。

### 方式二：集成到 Docker Compose 中

您可以直接将 Watchtower 作为附加服务加入现有的 `docker-compose.yml` 中：

```yaml
services:
  baihu:
    image: ghcr.io/engigu/baihu:latest
    container_name: baihu
    # ... 省略端口、挂载等其他配置 ...
    
  watchtower:
    image: containrrr/watchtower
    container_name: watchtower
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
    environment:
      - TZ=Asia/Shanghai
      - WATCHTOWER_CLEANUP=true
      - WATCHTOWER_SCHEDULE="0 0 3 * * *"
    command: baihu
    restart: unless-stopped
```

修改完成后，执行 `docker compose up -d` 生效即可。
