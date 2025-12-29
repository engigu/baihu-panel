# 白虎面板 [![Hits](https://hits.sh/github.com/engigu/baihu-panel.svg?view=today-total)](https://hits.sh/github.com/engigu/baihu-panel/)

白虎面板是一个轻量级定时任务管理系统，基于`Go`+`Vue3`构建，`docker`或者`docker-compose`，内置`python3`、`nodejs`、`bash`环境，开箱即用。

演示站点(演示站点的服务器比较烂，见谅) [demo](https://baihu-demo-site.qwapi.eu.org/)

## 项目来由

多少和青龙面板有点关系，我自己也是青龙面板的使用者，但是现在的青龙面板性能我觉得有点难以接受。以我自己的使用（`机器1C2G`）为例，一个`python`的`requests`脚本每隔`30s`执行一次，有时候cpu执行的时候能跳变到`50%`以上。可以看看下面gif图片（如果不动，点击图片查看）

![qinglong.gif](https://f.pz.al/pzal/2025/12/24/2d245b0a77f26.gif)

我觉得一个内存和性能占用低的面板更合适自己，所以做了这个项目。

如果你和我一样需要一个性能和内存占用低的定时面板，这个项目你可以体验下。

同样的定时场景和代码，这个项目的情况如下（cpu执行定时跳变不超过`20%`）：

![baihu.gif](https://f.pz.al/pzal/2025/12/24/f0d171f9a686d.gif)

## 特色

- **轻量级：** docker/compose部署，无需复杂配置，开箱即用
- **任务调度：** 支持标准 Cron 表达式，常用时间规则快捷选择。日志不落文件，没有磁盘频繁io的问题
- **脚本管理：** 在线代码编辑器，支持文件上传、压缩包解压
- **在线终端：** WebSocket 实时终端，命令执行结果实时输出
- **环境变量：** 安全存储敏感配置，任务执行时自动注入
- **现代 UI：** 响应式设计，深色/浅色主题切换
- **移动端：** 适配移动小屏样式

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

## 效果图 

![baihu-display.gif](https://raw.githubusercontent.com/engigu/resources/refs/heads/images/baihu-display.gif)
<!-- TODO: 添加效果图 -->

## 快速部署 

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
  -e BH_DB_PATH=/app/data/ql.db \
  -e BH_DB_TABLE_PREFIX=baihu_ \
  --restart unless-stopped \
  ghcr.io/engigu/baihu:latest
```

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
      - BH_DB_PATH=/app/data/ql.db
      - BH_DB_TABLE_PREFIX=baihu_
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

</details>

> 环境变量优先级高于配置文件，两种方式可以混合使用。


### 访问面板

启动后访问：http://localhost:8052

**默认账号：** `admin` / `123456`

> 首次登录后请立即修改默认密码

## 目录结构 

```
./
├── baihu                 # 可执行文件
├── data/                 # 数据目录（自动创建）
│   ├── ql.db             # SQLite 数据库
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
| `BH_DB_TYPE` | database.type | 数据库类型 (sqlite/mysql) | sqlite |
| `BH_DB_HOST` | database.host | 数据库地址 | localhost |
| `BH_DB_PORT` | database.port | 数据库端口 | 3306 |
| `BH_DB_USER` | database.user | 数据库用户 | root |
| `BH_DB_PASSWORD` | database.password | 数据库密码 | - |
| `BH_DB_NAME` | database.dbname | 数据库名称 | ql_panel |
| `BH_DB_PATH` | database.path | SQLite 文件路径 | ./data/ql.db |
| `BH_DB_TABLE_PREFIX` | database.table_prefix | 表前缀 | baihu_ |
| `BH_SECRET` | security.secret | JWT 密钥 | 手动指定 |

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

#### 关于推送

面板现在没有推送功能，还在考虑方式，但是现在有很多现成的消息聚合面板（这里自荐下自己的另一个项目[Message-Push-Nest](https://github.com/engigu/Message-Push-Nest)），如果大家有合适的方式可以提交Issue。

## 贡献 

欢迎提交 Issue 和 Pull Request！

## 许可证 

[MIT License](LICENSE)
