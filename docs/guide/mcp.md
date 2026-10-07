# 模型上下文协议 (MCP) 服务

白虎面板原生内置了 **MCP (Model Context Protocol)** 协议服务端，使 AI 工具（如 Cursor、Claude Desktop、Cline、Windsurf、VSCode 等）能够以标准协议直接连接并接管白虎面板的运维与开发能力。

通过 MCP 服务，AI 可以直接：
- 🔍 **智能查询**：检索定时任务列表、执行历史流水、实时系统状态与运行时环境；
- ⚡ **调度与控制**：一键手动触发任务、终止超时任务、热更新 Cron 表达式与配置；
- 🔐 **环境变量安全管理**：查询、新增、修改环境变量与机密，并自动触发集群 Agent 广播热更新；
- 📜 **脚本与代码管理**：在安全沙箱内浏览脚本目录树、搜索脚本、读取源码及写入自动化脚本；
- 🩺 **异常自动诊断**：借助内置 Prompt 模板提取任务报错日志，自动进行错误归因并给出修复建议。

---

## 快速接入配置

白虎面板 MCP 服务原生支持 **Stdio（标准输入输出）** 与 **SSE（Server-Sent Events 远程 HTTP）** 两种连接协议。

### 方式一：本地客户端 Stdio 管道接入（推荐）

适用于在本机运行的 AI 客户端（如 Claude Desktop、Cursor、Cline 等），直接调用白虎面板 CLI 的 `baihu mcp` 子命令。面板在启动 Stdio 模式时已自动隔离系统日志，确保标准输出通道纯净传输 JSON-RPC 报文。

#### 1. Claude Desktop 配置

打开 Claude Desktop 的配置文件 `claude_desktop_config.json`：
- **Windows**: `%APPDATA%\Claude\claude_desktop_config.json`
- **macOS**: `~/Library/Application Support/Claude/claude_desktop_config.json`

添加以下配置：

```json
{
  "mcpServers": {
    "baihu": {
      "command": "F:\\workspace\\baihu-panel\\baihu.exe",
      "args": ["mcp"]
    }
  }
}
```
*(请将 `command` 替换为您本地 `baihu` 可执行文件的绝对路径。Linux / macOS 下直接填写 `/usr/local/bin/baihu`)*

#### 2. Cursor 配置

在 Cursor 中依次打开 **Settings** $\to$ **Features** $\to$ **MCP** $\to$ **Add New MCP Server**：
- **Name**: `baihu`
- **Type**: `command`
- **Command**: `baihu mcp` （或指定面板可执行文件绝对路径）

---

### 方式二：远程网络 SSE 接入 (Server-Sent Events)

适用于跨主机、Docker 容器部署或远程 Web AI Agent 接入。

白虎面板在 Web 端口上挂载了受保护的 SSE 端点：
- **SSE 连接端点**: `http://<面板IP>:<端口>/open2api/v1/mcp/sse`
- **消息发送端点**: `http://<面板IP>:<端口>/open2api/v1/mcp/messages`

#### 鉴权与配置

连接 SSE 端点需要携带在面板中配置的 OpenAPI Token，支持两种传参方式：
1. **URL 参数**: `?token=YOUR_OPENAPI_TOKEN`（推荐，完美兼容浏览器原生 EventSource 握手）；
2. **HTTP Header**: `Authorization: Bearer YOUR_OPENAPI_TOKEN`。

在 AI 客户端中配置远程 SSE Server 示例：

```json
{
  "mcpServers": {
    "baihu-remote": {
      "url": "http://127.0.0.1:5678/open2api/v1/mcp/sse?token=sk-xxxxxxxxxxxxxxxxxxxx"
    }
  }
}
```

---

## 内置工具集 (Tools)

白虎面板 MCP 服务向 AI 模型暴露了 24 个全生命周期的原子运维工具，工具层均 100% 委托复用核心业务服务，天然具备并发控制、数据校验与 Agent 联动。

### 1. 系统与状态监控

| 工具名称 | 参数 | 说明 |
| :--- | :--- | :--- |
| **`get_system_status`** | 无 | 获取面板系统运行状态概览（版本号、架构、任务总数、调度数、运行中任务数、环境变量数与系统运行时间）。 |

### 2. 定时任务编排与控制

| 工具名称 | 参数 | 说明 |
| :--- | :--- | :--- |
| **`list_tasks`** | `name`, `type`, `tag`, `enabled`, `page`, `page_size` | 分页检索任务列表，支持按任务名、类型（task/repo/app）、标签及启停状态筛选。 |
| **`get_task`** | `id` *(必填)* | 获取指定任务的完整配置元数据（含 Cron 表达式、超时时间、工作目录与环境变量映射）。 |
| **`create_task`** | `name`, `command`, `schedule`, `work_dir`, `timeout`, `remark`, `tags` | 创建新的自动化任务。包含严格的 6 位秒级 Cron 表达式校验，并自动挂载至计划任务调度器。 |
| **`update_task`** | `id` *(必填)*, `name`, `command`, `schedule`, `enabled` 等 | 修改任务配置。若启停状态或 Cron 变更，将自动进行热更新调度或广播至 Agent。 |
| **`delete_task`** | `id` *(必填)* | 删除指定任务，自动完成调度器注销、运行实例终止与通知关联级联清理。 |
| **`execute_task`**| `id` *(必填)* | 立即手动触发执行指定任务，返回本次执行的 `log_id`，便于后续追踪日志。 |
| **`stop_task`**   | `log_id` 或 `task_id` | 强制终止正在运行中的任务副本或特定执行实例。 |

### 3. 环境变量与安全机密

| 工具名称 | 参数 | 说明 |
| :--- | :--- | :--- |
| **`list_env_vars`** | `name`, `type` (normal/secret), `tags`, `page`, `page_size` | 检索环境变量列表，自动脱敏机密变量。 |
| **`set_env_var`**   | `name` *(必填)*, `value` *(必填)*, `remark`, `type`, `tags`, `enabled` | 智能写入环境变量。若存在同名变量则自动更新，不存在则自动新建；机密类型自动启用 AES 加密存储，并自动广播通知集群 Agent 热刷新。 |
| **`delete_env_var`**| `id` *(必填)*, `force` (布尔值) | 删除指定环境变量。支持依赖冲突防护：若变量被定时任务关联引用，默认拦截并提示受影响任务列表；传入 `force: true` 可强制解除关联并删除。 |

### 4. 执行历史与日志排查

| 工具名称 | 参数 | 说明 |
| :--- | :--- | :--- |
| **`list_task_logs`**| `task_id`, `status` (success/failed/running), `page`, `page_size` | 分页检索任务的历史执行记录流水。 |
| **`get_log_detail`**| `log_id` *(必填)* | 获取指定日志的完整输出内容。自动解压 Base64 压缩日志，并格式化输出标准控制台流与错误流。 |
| **`clean_task_logs`**| `task_id`, `days` | 清理历史任务执行日志，释放存储空间。支持指定任务或按保留天数（如 7 天前）清理。 |

### 5. 脚本文件管理（受安全沙箱保护）

| 工具名称 | 参数 | 说明 |
| :--- | :--- | :--- |
| **`get_file_tree`** | `path` (可选相对路径) | 浏览指定脚本目录下的单层文件与文件夹列表。 |
| **`read_script`**   | `path` *(必填)* | 读取脚本文件的文本源码，自动检测并拦截二进制文件。 |
| **`save_script`**   | `path` *(必填)*, `content` *(必填)* | 创建或覆盖保存脚本文件，若父级目录不存在将自动递归创建。 |
| **`search_scripts`**| `keyword` *(必填)*, `limit` | 根据关键字在整个脚本工作区中模糊搜索匹配的文件。 |
| **`delete_script`** | `path` *(必填)* | 安全删除脚本目录下的指定文件或文件夹。 |

### 6. 声明式应用与应用商店 (AppStore & Declarative Apps)

| 工具名称 | 参数 | 说明 |
| :--- | :--- | :--- |
| **`list_store_apps`** | `keyword`, `category` | 检索白虎官方应用商店中的可用声明式自动化应用列表，支持按分类与关键字模糊搜索。 |
| **`get_store_app`** | `app_id` *(必填)* | 获取应用商店中指定应用的完整元数据与定义规范（含多语言环境、预置任务清单、配置表单与场景预设）。 |
| **`list_installed_apps`**| 无 | 列出当前面板中已安装的声明式应用清单、版本、作者及当前激活的场景。 |
| **`get_installed_app`** | `app_id` *(必填)* | 获取指定已安装应用的配置详情、当前运行场景及其拆解导出的全部受控子任务清单。 |
| **`install_app`** | `app_id` 或 `path_or_url`, `scenario_id`, `schedule`, `force_setup` 等 | 一键安装应用商店应用或通过远程/本地 Manifest 部署，自动完成依赖准备、源码克隆、任务生成与 Cron 调度注册。 |
| **`switch_app_scenario`**| `app_id` *(必填)*, `scenario_id` *(必填)* | 切换已安装应用的运行场景预设（批量调整子任务启停状态与定制 Cron）。 |
| **`uninstall_app`** | `app_id` *(必填)*, `clean_data`, `clean_envs` | 卸载声明式应用，级联安全清理受控子任务、关联环境变量、无引用孤儿标签及磁盘代码产物目录。 |

### 7. 代码仓库同步 (Repo Sync)

| 工具名称 | 参数 | 说明 |
| :--- | :--- | :--- |
| **`list_repos`** | `keyword` | 列出白虎面板已绑定的所有 Git 脚本代码仓库配置与最近同步状态。 |
| **`sync_repo`** | `repo_id` *(必填)* | 手动触发指定 Git 仓库的即时拉取与同步更新。 |

### 8. 消息通知与告警测试 (Notification Channels)

| 工具名称 | 参数 | 说明 |
| :--- | :--- | :--- |
| **`list_notify_channels`** | 无 | 列出白虎面板已配置的全部通知渠道（自动脱敏密钥 Token）及支持的渠道类型。 |
| **`test_notification`** | `channel_id` *(必填)* | 向指定通知渠道发送一条连通性测试消息，检验渠道配置是否正常。 |
| **`send_notification`** | `title` *(必填)*, `content` *(必填)*, `channel_id`, `format` | 借由面板向指定渠道或所有已启用的渠道推送告警或运维摘要消息。 |


---

## 上下文资源 (Resources)

AI 客户端可通过 URI 模式将面板实体作为背景上下文引入到对话提示中：

- **`baihu://status`**：白虎面板当前运行状态与统计摘要（JSON 格式）。
- **`notify://channels`**：白虎面板已配置的消息通知渠道清单与健康状态摘要。
- **`app://{id}`**：根据应用 ID 动态读取已安装声明式应用的完整配置与受控子任务清单（如 `app://bilibili-tool-pro`）。
- **`repo://{id}`**：根据代码仓库 ID 动态读取 Git 仓库配置与同步流水状态（如 `repo://d9u87x456`）。
- **`task://{id}`**：根据任务 ID 动态读取任务的完整配置模型（如 `task://d9u87x123`）。
- **`log://{id}`**：根据日志 ID 动态读取某次执行的控制台完整输出与错误堆栈（如 `log://c782109`）。
- **`file://{path}`**：根据工作目录相对路径动态读取脚本源码内容（如 `file://my_script.js`）。

---

## 智能提示词模板 (Prompts)

白虎面板预置了针对自动化任务运维场景深度调优的 Prompt 模板：

### 1. `diagnose_task_failure`（任务失败智能排查）
- **参数**: `log_id`
- **能力**: 自动抓取失败日志与对应任务配置，组装并引导 AI 专家排查核心报错行、环境依赖缺失、语法缺陷，并直接给出可运行的修复代码或命令。

### 2. `create_scheduled_task`（自动化定时任务生成助手）
- **参数**: `task_description`（例如：“每天凌晨3点同步代码并推送到企业微信通知”）
- **能力**: 自动将自然语言需求转化为规范的 6 位秒级标准 Cron 表达式、推荐执行命令与完整的脚本示例代码，并指引用户写入面板。

---

## 安全性保障

1. **工作区沙箱隔离**：文件读写工具严格限制在 `data/scripts` 工作目录下，所有相对路径经过权威规范化校验，严格禁止通过 `../` 进行目录穿越攻击。
2. **OpenAPI 密钥鉴权**：远程 SSE 模式强制校验 OpenAPI 令牌，采用恒定时间哈希比对防范时序侧信道攻击。
3. **敏感机密安全存储**：机密类型环境变量在落盘前均通过密钥加密，并在控制台回显与日志检索中进行脱敏掩码保护。
