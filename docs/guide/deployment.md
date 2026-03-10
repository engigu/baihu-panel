# 快速部署

项目提供多种基础镜像，默认版本基于 Debian 12，集成了 Python 3.13 与 Node.js 23。

## 基础镜像选择

| 标签 (Tag) | 基础镜像 | 说明 |
| :--- | :--- | :--- |
| `latest` | Debian 12 | 默认版本，集成 Python 3.13 与 Node.js 23 |
| `latest-debian13` | Debian 13 | 尝鲜版本|

> **提示**：下方部署示例默认使用 `latest` 标签，如需换用 Debian 13 版，只需将 `latest` 替换为 `latest-debian13` 即可。

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
