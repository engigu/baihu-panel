# 系统配置手册

白虎面板支持通过环境变量和配置文件两种核心方式进行系统参数微调。

## 环境变量配置 (优先级最高)

环境变量在容器内自动注入，非常适合 CI/CD 和 Docker 混合编排场景。

### 核心配置项列表

| 环境变量 | 对应配置 | 说明 | 默认值 |
| :--- | :--- | :--- | :--- |
| `BH_SERVER_PORT` | server.port | 服务监听端口 | 8052 |
| `BH_SERVER_HOST` | server.host | 监听地址 | 0.0.0.0 |
| `BH_SERVER_URL_PREFIX` | server.url_prefix | URL 前缀，用于反向代理子路径部署 | - |
| `BH_DB_TYPE` | database.type | 数据库类型 (sqlite/mysql) | sqlite |
| `BH_DB_HOST` | database.host | 数据库实例地址 | localhost |
| `BH_DB_PORT` | database.port | 数据库端口 | 3306 |
| `BH_DB_USER` | database.user | 数据库用户名 | root |
| `BH_DB_PASSWORD` | database.password | 数据库密码 | - |
| `BH_DB_NAME` | database.dbname | 数据库库名 | baihu |
| `BH_DB_PATH` | database.path | SQLite 物理文件存储路径 | ./data/baihu.db |
| `BH_DB_TABLE_PREFIX` | database.table_prefix | 数据库表前缀 | baihu_ |

---

## 配置文件挂载 (config.ini)

如果您希望对系统参数有更细致的控制（而非通过外部注入），可以使用配置文件。

### 挂载点
```yaml
volumes:
  - ./configs:/app/configs
```

### 配置文件示例 (`configs/config.ini`)
```ini
[server]
port = 8052
host = 0.0.0.0
# 配置 URL 前缀用于反向代理，例如 /baihu/
url_prefix = /baihu

[database]
type = sqlite
path = /app/data/baihu.db
table_prefix = baihu_
```

---

## 调度设置说明

系统采用异步任务队列 + Worker Pool 架构，可在「系统设置 > 调度设置」页面进行配置：

- **Worker 数量** (默认 4)：同时在后端并发运行的任务进程数。
- **队列大小** (默认 100)：待处理任务队列的最大容量。
- **速率间隔** (默认 200 ms)：控制两个任务启动之间的最小等待时长。

> **提示**：修改以上调度参数后，系统会立即响应并动态调整，无需重启容器。
