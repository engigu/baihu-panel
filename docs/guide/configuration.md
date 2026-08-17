# 系统配置手册

baihu-panel支持通过环境变量和配置文件两种核心方式进行系统参数微调。

## 环境变量配置 (优先级最高)

环境变量在容器内自动注入，非常适合 CI/CD 和 Docker 混合编排场景。

### 核心配置项列表

| 环境变量 | 对应配置 | 说明 | 默认值 |
| :--- | :--- | :--- | :--- |
| `BH_SERVER_PORT` | server.port | 服务监听端口 | 8052 |
| `BH_SERVER_HOST` | server.host | 监听地址 | 0.0.0.0 |
| `BH_SERVER_URL_PREFIX` | server.url_prefix | URL 前缀，用于反向代理子路径部署 | - |
| `BH_COOKIE_NAME` | server.cookie_name | 全局会话 Cookie 名称 | BHToken |
| `BH_DB_TYPE` | database.type | 数据库类型 (sqlite/mysql) | sqlite |
| `BH_DB_HOST` | database.host | 数据库实例地址 | localhost |
| `BH_DB_PORT` | database.port | 数据库端口 | 3306 |
| `BH_DB_USER` | database.user | 数据库用户名 | root |
| `BH_DB_PASSWORD` | database.password | 数据库密码 | - |
| `BH_DB_NAME` | database.dbname | 数据库库名 | baihu |
| `BH_DB_PATH` | database.path | SQLite 物理文件存储路径 | ./data/baihu.db |
| `BH_DB_DSN` | database.dsn | 数据库 DSN (仅 mysql/postgres, 优先级高。**需对应设置 type**) | - |
| `BH_DB_TABLE_PREFIX` | database.table_prefix | 数据库表前缀 | baihu_ |
| `BH_DB_SSL_MODE` | database.ssl_mode | SSL 模式: postgres 支持 disable/require/verify-ca/verify-full; mysql 支持 true/skip-verify | - |
| `BAIHU_SECRET_KEY` | - | 系统加密秘钥，用于机密变量功能（**注：仅支持环境变量设置，不支持配置文件**） | - |

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
# 全局会话 Cookie 名称
cookie_name = BHToken

[database]
type = sqlite
path = /app/data/baihu.db
# 数据库连接示例 (Unix Socket / DSN): 
# 注意：使用 dsn 时，type 必须设为 mysql 或 postgres
# dsn = user:password@unix(/var/run/mysqld/mysqld.sock)/dbname?charset=utf8mb4&parseTime=True&loc=Local
# dsn = postgres://user:password@localhost:5432/dbname?sslmode=disable
table_prefix = baihu_
```

---

## 调度设置说明

系统采用异步任务队列 + Worker Pool 架构，可在「系统设置 > 调度设置」页面进行配置：

- **Worker 数量** (默认 4)：同时在后端并发运行的任务进程数。
- **队列大小** (默认 100)：待处理任务队列的最大容量。
- **速率间隔** (默认 200 ms)：控制两个任务启动之间的最小等待时长。

---

## 机密管理 (Secret Management)

baihu-panel提供了一套基于 **AES-GCM** 工业级标准的安全机密管理系统，其设计理念参考了 GitHub Actions Secrets。

### 核心特性

1. **强加密存储**：所有标记为“机密”的变量在数据库中均以加密密文形式存储。
2. **秘钥安全**：通过环境变量 `BAIHU_SECRET_KEY` 注入加密秘钥。系统读取秘钥后会立即将其从进程环境变量中销毁（Unset），确保秘钥仅驻留在内存中。
3. **日志自动脱敏**：系统会自动扫描任务执行生成的实时日志流。一旦发现机密明文，将自动替换为 `********`，防止敏感信息通过日志泄露。
4. **严格权限隔离**：
   - 机密内容**仅在计划任务由调度器定时执行时**才会注入到环境。
   - 通过**终端命令**、**测试运行**或**调试运行**调起的临时进程无法获取机密内容，保障核心资产安全。

### 配置建议

- 建议在 Docker/Compose 启动项中设置 `BAIHU_SECRET_KEY` 为一个复杂的随机字符串。
- 不要将该秘钥写入 `config.ini` 或提交到版本控制系统。

