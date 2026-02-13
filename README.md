# 白虎面板 

[![Hits](https://hits.sh/github.com/engigu/baihu-panel.svg?view=today-total)](https://hits.sh/github.com/engigu/baihu-panel/)
![Latest Version](https://ghcr-badge.egpl.dev/engigu/baihu/latest_tag?color=%2344cc11&ignore=latest%2Cmain*&label=docker+version&trim=)
![Image Size](https://ghcr-badge.egpl.dev/engigu/baihu/size?color=%2344cc11&tag=latest&label=docker+image&trim=)
![Image pulls](https://img.shields.io/badge/dynamic/json?url=https://ghcr-badge.elias.eu.org/api/engigu/baihu-panel/baihu&query=downloadCount&style=flat&label=docker%20pulls&color=44cc11)

白虎面板是一个轻量级定时任务管理系统，基于`Go`+`Vue3`构建，`docker`或者`docker-compose`，内置`python3`、`nodejs`、`bash`环境，开箱即用。

演示站点(演示站点的服务器比较烂，见谅)  [演示站点](https://baihu-demo-site.qwapi.eu.org/)

## 更新日志 ☕

### 最近更新

**2026.02.11** - 增强安全性：首次启动使用随机密码并打印在日志中，登录接口增加防暴力破解，文件系统操作增加路径穿越锁定  
**2026.02.10** - 重构任务调度系统，完善并发控制，优化文件树交互体验，支持任务执行实时日志流  
**2026.02.06** - 整理 Docker 目录结构，增加 Alpine 及 Debian 13 (Trixie) 镜像支持  

[查看完整更新日志](./CHANGELOG.md)

## 项目来由

多少和青龙面板有点关系，我自己也是青龙面板的使用者，但是现在的青龙面板性能我觉得有点难以接受。以我自己的使用（`机器1C2G`）为例，一个`python`的`requests`脚本每隔`30s`执行一次，有时候cpu执行的时候能跳变到`50%`以上。可以看看下面gif图片（如果不动，点击图片查看）

![qinglong.gif](https://f.pz.al/pzal/2025/12/24/2d245b0a77f26.gif)

我觉得一个内存和性能占用低的面板更合适自己，所以做了这个项目。

如果你和我一样需要一个性能和内存占用低的定时面板，这个项目你可以体验下。

同样的定时场景和代码，这个项目的情况如下（cpu执行定时跳变不超过`20%`）：

![baihu.gif](https://f.pz.al/pzal/2025/12/24/f0d171f9a686d.gif)

如果项目有用，请帮忙点个star。

## 特色

- **轻量级：** docker/compose部署，无需复杂配置，开箱即用
- **任务调度：** 支持标准 Cron 表达式，常用时间规则快捷选择。日志不落文件，没有磁盘频繁io的问题
- **脚本管理：** 在线代码编辑器，支持文件上传、压缩包解压
- **在线终端：** WebSocket 实时终端，命令执行结果实时输出
- **环境变量：** 安全存储敏感配置，任务执行时自动注入
- **现代UI：** 响应式设计，深色/浅色主题切换
- **移动端：** 适配移动小屏样式
- **远程执行：** 支持远程agent执行任务，展示执行结果

## 功能特性 

<details>
<summary><b>点击展开查看详细功能</b></summary>

### 定时任务管理
- 支持标准 Cron 表达式调度
- 常用时间规则快捷选择
- 任务启用/禁用状态切换
- 手动触发执行
- 任务超时控制

### 脚本文件管理
- 在线代码编辑器
- 文件树形结构展示
- 支持创建、重命名、删除文件/文件夹
- 支持压缩包上传解压
- 支持多文件批量上传

### 在线终端
- WebSocket 实时终端
- 支持常用 Shell 命令
- 命令执行结果实时输出

### 执行日志
- 任务执行历史记录
- 执行状态追踪（成功/失败/超时）
- 执行耗时统计
- 日志内容压缩存储
- 日志自动清理

### 环境变量
- 安全存储敏感配置
- 变量值脱敏显示
- 任务执行时自动注入

### 系统设置
- 站点标题、标语、图标自定义
- 分页大小、Cookie 有效期配置
- 调度参数热重载
- 数据备份与恢复

</details>

## 支持语言脚本和依赖

<details>
<summary><b>点击展开查看已支持的语言及依赖管理详情</b></summary>

### 脚本运行环境
白虎面板原生支持以下脚本的定时执行：
- **Python3**, **Node.js**, **Bash** (内置环境)
- 通过 **Mise** 扩展：支持几乎所有主流编程语言的动态安装与切换。

### 依赖管理支持
系统内置了高度集成的跨语言依赖管理器，支持自动化安装和管理以下语言的依赖项，并确保在容器内全局可用：

| 语言 | 包管理器 | 功能说明 |
| :--- | :--- | :--- |
| **Python** | pip | 自动使用内置虚拟环境，支持清华源 |
| **Node.js** | npm | 全局安装模式，自动配置 npmmirror 镜像 |
| **Go** | go install | 通过 `go install` 安装二进制工具 |
| **Rust** | cargo | 通过 `cargo install` 安装 Rust 依赖 |
| **Ruby** | gem | 支持 `gem install` 本地安装 |
| **Bun** | bun | 支持 `bun add -g` 全局模式 |
| **PHP** | composer | 支持 `composer global require` |
| **Deno** | deno | 支持 `deno install -g` |
| **.NET** | dotnet | 支持 `dotnet tool install -g` |
| **Elixir/Erlang** | mix | 支持 `mix archive.install` |
| **Lua** | luarocks | 通过 `luarocks` 管理 Lua 包 |
| **Nim** | nimble | 支持 `nimble install` |
| **Dart/Flutter** | pub | 支持 `pub global activate` |
| **Perl** | cpanm | 简单的 `cpanm` 安装支持 |
| **Crystal** | shards | `shards` 项目级别或工具安装 |

### 使用方法
1. **安装环境**：进入「编程语言」页面，使用 `mise` 一键安装所需的语言及版本。
2. **依赖管理**：在已安装列表点击「依赖管理」，输入名称（可选版本）即可自动在对应环境内完成安装。
3. **隔离机制**：系统基于 `mise exec` 实现了完善的环境隔离，不同版本的依赖包互不冲突。

</details>

## 效果图 

![baihu-display.gif](https://raw.githubusercontent.com/engigu/resources/refs/heads/images/baihu-display.gif)
<!-- TODO: 添加效果图 -->

## 快速部署 

项目提供多种基础镜像，可根据具体环境选择：

| 标签 (Tag) | 基础镜像 | 说明 |
| :--- | :--- | :--- |
| `latest` | Debian 12 | 默认版本，平衡稳定与兼容性 |
| `latest-alpine` | Alpine | 极简版本，镜像体积最小 |
| `latest-debian13` | Debian 13 | 尝鲜版本，基于 Debian Trixie |

> **提示**：下方部署示例默认使用 `latest` 标签，如需换用 Alpine 或 Debian 13 版，只需将 `latest` 替换为 `latest-alpine` 或 `latest-debian13` 即可。


<details>
<summary><b>方式一：环境变量部署（推荐）</b></summary>

通过环境变量指定配置，简单灵活，适合容器编排场景。

**使用 SQLite（默认）：**

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

> **提示**：如需通过反向代理部署在子路径（如 `/baihu`），添加环境变量：
> ```bash
> -e BH_SERVER_URL_PREFIX=/baihu
> ```
> 配置后访问地址为 `http://your-domain.com/baihu/`，详见下方「URL 前缀配置」说明。

**Docker Compose（SQLite）：**

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

**使用 MySQL：**

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

> **提示**：如需配置 URL 前缀，添加 `-e BH_SERVER_URL_PREFIX=/baihu`

**Docker Compose（MySQL）：**

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
      # - BH_SERVER_URL_PREFIX=/baihu  # 可选：配置 URL 前缀
      - BH_DB_TYPE=mysql
      - BH_DB_HOST=mysql-server
      - BH_DB_PORT=3306
      - BH_DB_USER=root
      - BH_DB_PASSWORD=your_password
      - BH_DB_NAME=baihu
      - BH_DB_TABLE_PREFIX=baihu_
    logging:
      driver: json-file
      options:
        max-size: "10m"
        max-file: "3"
    restart: unless-stopped
```

</details>

<details>
<summary><b>方式二：配置文件部署</b></summary>

通过挂载 `config.ini` 配置文件来管理配置，适合需要持久化配置的场景。

**Docker 命令：**

```bash
docker run -d \
  --name baihu \
  -p 8052:8052 \
  -v $(pwd)/data:/app/data \
  -v $(pwd)/configs:/app/configs \
  -v $(pwd)/envs:/app/envs \
  -e TZ=Asia/Shanghai \
  --restart unless-stopped \
  ghcr.io/engigu/baihu:latest
```

**Docker Compose：**

```yaml
services:
  baihu:
    image: ghcr.io/engigu/baihu:latest
    container_name: baihu
    ports:
      - "8052:8052"
    volumes:
      - ./data:/app/data
      - ./configs:/app/configs
      - ./envs:/app/envs
    environment:
      - TZ=Asia/Shanghai
    logging:
      driver: json-file
      options:
        max-size: "10m"
        max-file: "3"
    restart: unless-stopped
```

首次使用需要复制 `configs/config.example.ini` 为 `configs/config.ini`，然后根据需要修改配置。

**配置文件示例（`configs/config.ini`）：**

```ini
[server]
port = 8052
host = 0.0.0.0
# 可选：配置 URL 前缀用于反向代理，例如 /baihu
url_prefix = 

[database]
type = sqlite
path = ./data/baihu.db
table_prefix = baihu_
```

</details>

<details>
<summary><b>方式三：一键部署（白虎 + 消息推送）（推荐）</b></summary>

白虎面板本身不内置推送功能，但可以与消息聚合服务配合使用。推荐使用 [Message-Push-Nest](https://github.com/engigu/Message-Push-Nest) 作为统一的消息推送中心。

**使用 SQLite**

创建 `docker-compose.yml` 文件：

```yaml
services:
  # 白虎面板
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
    logging:
      driver: json-file
      options:
        max-size: "10m"
        max-file: "3"
    restart: unless-stopped
    depends_on:
      - message-nest

  # 消息推送服务
  message-nest:
    image: ghcr.io/engigu/message-nest:latest
    # 或使用 Docker Hub 镜像
    # image: engigu/message-nest:latest
    container_name: message-nest
    ports:
      - "8053:8000"
    environment:
      - TZ=Asia/Shanghai
      - DB_TYPE=sqlite
    volumes:
      - ./message-nest-data:/app/data
    logging:
      driver: json-file
      options:
        max-size: "10m"
        max-file: "3"
    restart: unless-stopped
```

**使用 MySQL（适合生产环境，需要已有 MySQL 服务）**

创建 `docker-compose.yml` 文件：

```yaml
services:
  # 白虎面板
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
      - BH_DB_TYPE=mysql
      - BH_DB_HOST=192.168.1.100  # 修改为你的 MySQL 地址
      - BH_DB_PORT=3306
      - BH_DB_USER=root
      - BH_DB_PASSWORD=your_password  # 修改为你的 MySQL 密码
      - BH_DB_NAME=baihu
      - BH_DB_TABLE_PREFIX=baihu_
    logging:
      driver: json-file
      options:
        max-size: "10m"
        max-file: "3"
    restart: unless-stopped
    depends_on:
      - message-nest

  # 消息推送服务
  message-nest:
    image: ghcr.io/engigu/message-nest:latest
    # 或使用 Docker Hub 镜像
    # image: engigu/message-nest:latest
    container_name: message-nest
    ports:
      - "8053:8000"
    environment:
      - TZ=Asia/Shanghai
      - DB_TYPE=mysql
      - MYSQL_HOST=192.168.1.100  # 修改为你的 MySQL 地址
      - MYSQL_PORT=3306
      - MYSQL_USER=root
      - MYSQL_PASSWORD=your_password  # 修改为你的 MySQL 密码
      - MYSQL_DB=message_nest
      - MYSQL_TABLE_PREFIX=message_
    logging:
      driver: json-file
      options:
        max-size: "10m"
        max-file: "3"
    restart: unless-stopped
```

启动服务：

```bash
docker-compose up -d
```

**访问地址：**
- 白虎面板：http://localhost:8052
- 消息推送服务：http://localhost:8053

> 注意：使用 MySQL 方式时，请先在 MySQL 中创建 `baihu` 和 `message_nest` 两个数据库，并修改配置中的 MySQL 地址和密码。也可以使用同一个数据库。

**在任务中使用推送**

Message-Push-Nest 提供了便捷的推送代码生成功能：

1. 在 Message-Push-Nest 管理界面中配置推送渠道（如企业微信、钉钉、邮件等）
2. 在「消息模板」或「任务」页面，点击「复制推送代码」按钮
3. 选择对应的编程语言（Python、Bash、Node.js 等）
4. 将生成的代码粘贴到白虎面板的任务脚本中需要推送的位置

![1768143124572.png](https://f.pz.al/pzal/2026/01/11/1360cd334ff20.png)

> 提示：在 Docker Compose 部署的环境中，推送服务地址使用 `http://message-nest:8000`（容器内部通信）。如果是独立部署，请使用实际的服务地址。

</details>

> 环境变量优先级高于配置文件，两种方式可以混合使用。

<details>
<summary><b>方式四：Nginx 反向代理部署（HTTPS）</b></summary>

如果需要通过域名和 HTTPS 访问白虎面板，可以使用 Nginx 作为反向代理。

**Nginx 配置示例：**

```nginx
# 在 http 块中添加 WebSocket 升级配置
map $http_upgrade $connection_upgrade {
    default upgrade;
    '' close;
}

server {
    listen 443 ssl http2;
    server_name example.com;
    
    ssl_certificate     /etc/letsencrypt/live/example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/example.com/privkey.pem;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_prefer_server_ciphers off;
    
    access_log /var/log/nginx/example.access.log;
    error_log  /var/log/nginx/example.error.log warn;
    
    location / {
        proxy_pass http://172.17.0.1:8052;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto https;
        
        # WebSocket 支持（终端功能需要）
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection $connection_upgrade;
        
        proxy_buffering off;
        proxy_read_timeout 60s;
    }
}

# HTTP 自动跳转 HTTPS（可选）
server {
    listen 80;
    server_name example.com;
    return 301 https://$server_name$request_uri;
}
```

**配置说明：**

1. 将 `example.com` 替换为你的域名
2. 修改 SSL 证书路径为你的实际路径
3. `172.17.0.1:8052` 是 Docker 容器的宿主机地址和端口，根据实际情况修改
4. WebSocket 配置是必需的，否则在线终端功能无法使用


**重载 Nginx 配置：**

```bash
nginx -t && nginx -s reload
```

</details>


### 访问面板

启动后访问：http://localhost:8052

**默认账号：** 用户名 `admin`，密码见启动日志（首次启动会自动生成 12 位随机密码并打印在日志中）

> **注意**：出于安全性考虑，系统不再使用固定默认密码。请在容器启动日志中搜索 `管理员账号创建成功` 找到您的随机密码，并登录后及时修改。

### 数据目录

```
./
├── baihu                 # 可执行文件
├── data/                 # 数据目录（自动创建）
│   ├── baihu.db             # SQLite 数据库
│   └── scripts/          # 脚本文件存储
├── configs/
│   └── config.ini        # 配置文件（自动创建）
└── envs/                 # 运行环境目录（自动创建）
    ├── python/           # Python 虚拟环境
    └── node/             # Node.js npm 全局安装目录
```

### Docker 启动流程

容器启动时 `docker-entrypoint.sh` 会执行以下操作：

1. **创建必要目录**：`/app/data`、`/app/data/scripts`、`/app/configs`、`/app/envs`
2. **初始化 Python 虚拟环境**：如果 `/app/envs/python` 不存在，自动创建并配置清华 pip 镜像源
3. **配置 Node.js 环境**：设置 npm prefix 到 `/app/envs/node`，配置 npmmirror 镜像源
4. **激活环境**：将 `/app/envs/python/bin` 和 `/app/envs/node/bin` 加入 PATH
5. **启动应用**

>  通过挂载 `./envs:/app/envs` 可以持久化 Python 和 Node.js 环境，避免每次重启容器都重新安装依赖。

## 配置说明

<details>
<summary><b>点击展开查看配置详情</b></summary>

### 配置文件

配置文件路径：`configs/config.ini`

```ini
[server]
port = 8052
host = 0.0.0.0
url_prefix =

[database]
type = sqlite
host = localhost
port = 3306
user = root
password = 
dbname = ql_panel
table_prefix = baihu_
```

### 环境变量

所有配置项都支持通过环境变量覆盖，环境变量优先级高于配置文件：

| 环境变量 | 对应配置 | 说明 | 默认值 |
|----------|----------|------|--------|
| `BH_SERVER_PORT` | server.port | 服务端口 | 8052 |
| `BH_SERVER_HOST` | server.host | 监听地址 | 0.0.0.0 |
| `BH_SERVER_URL_PREFIX` | server.url_prefix | URL 前缀，用于反向代理子路径部署 | - |
| `BH_DB_TYPE` | database.type | 数据库类型 (sqlite/mysql) | sqlite |
| `BH_DB_HOST` | database.host | 数据库地址 | localhost |
| `BH_DB_PORT` | database.port | 数据库端口 | 3306 |
| `BH_DB_USER` | database.user | 数据库用户 | root |
| `BH_DB_PASSWORD` | database.password | 数据库密码 | - |
| `BH_DB_NAME` | database.dbname | 数据库名称 | ql_panel |
| `BH_DB_PATH` | database.path | SQLite 文件路径 | ./data/baihu.db |
| `BH_DB_TABLE_PREFIX` | database.table_prefix | 表前缀 | baihu_ |
| `BH_SECRET` | security.secret | JWT 密钥 | 手动指定 |

### URL 前缀配置

如果需要通过反向代理（如 Nginx）将白虎面板部署在子路径下，可以配置 URL 前缀。

**配置方式：**

```bash
# 方式一：配置文件
[server]
url_prefix = /baihu

# 方式二：环境变量
-e BH_SERVER_URL_PREFIX=/baihu
```

**配置效果：**

配置 `url_prefix = /baihu` 后，访问路径变为：

| 类型 | 路径示例 |
|------|---------|
| 前端页面 | `http://your-domain.com/baihu/` |
| 登录页面 | `http://your-domain.com/baihu/login` |
| 任务管理 | `http://your-domain.com/baihu/tasks` |
| API 接口 | `http://your-domain.com/baihu/api/v1/*` |
| WebSocket | `ws://your-domain.com/baihu/api/v1/terminal/ws` |

**Nginx 反向代理配置示例：**

```nginx
location /baihu/ {
    proxy_pass http://localhost:8052/baihu/;
    proxy_http_version 1.1;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection "upgrade";
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
}
```

**MySQL 示例：**

参考上方「方式一：环境变量部署」中的 MySQL 配置示例。

### 调度设置

系统采用 Worker Pool + 任务队列的架构来控制任务执行，可在「系统设置 > 调度设置」中配置：

| 设置项 | 说明 | 默认值 |
|--------|------|--------|
| Worker 数量 | 并发执行任务的 worker 数量 | 4 |
| 队列大小 | 任务队列缓冲区大小 | 100 |
| 速率间隔 | 任务启动间隔（毫秒） | 200 |

修改调度设置后立即生效，无需重启服务。

</details>

## 贡献 

欢迎提交 Issue 和 Pull Request！

<img src="https://f.pz.al/pzal/2026/01/07/83be93eb4e2a3.png" width="200" />

## 许可证 

[MIT License](LICENSE)
