# 消息中心

消息中心集成了一整套灵活且现代的消息分发引擎，支持多场景自动推送到外部 IM 工具。

## 消息通道

- **企业 IM**：支持集成 **企业微信** (WeCom)、**钉钉** (DingTalk)、**飞书** (Lark)。
- **个人推送到位**：支持 **Telegram** Bot、**Bark** (支持自建)、**VoceChat** (支持自建) 以及基于 **Wpush** 的推送服务。
- **公共渠道**：标准的 **SMTP 邮件** 及 **Webhook** 回调。

## 事件通知规则

- **多事件配置**：您可以灵活定义在哪些场景下触发通知，包括但不限于：
    - **任务失败**：定时任务在 Cron 触发后运行报错。
    - **任务超时**：任务由于运行过长被系统中止。
    - **登录安全**：检测到异地登录或多次密码错误。
    - **服务下线**：Agent 节点掉线提醒。

## 推送使用路径

baihu-panel提供了两种不同层面的通知推送方式，满足从“自动报警”到“程序内自定义推送”的全场景需求。

---

### 路径一：任务绑定通知（零代码自动化）

这是最常用的方式，用于在定时任务执行完成后，根据结果自动发送通知。

1. **入口**：在 **「定时任务」** 页面，点击任务右侧的 **「编辑」**。
2. **配置**：在弹窗底部的 **「通知配置」** 栏目中：
    - **选择渠道**：指定发送消息的 IM 渠道。
    - **触发时机**：勾选 `成功时`、`失败时` 或 `超时时`（建议至少勾选失败和超时）。
    - **附带日志**：开启后可在消息中直接预览报错日志，支持设置截取长度。
3. **生效**：保存后，该任务每次运行结束都会按设定的逻辑自动推信。

---

### 路径二：脚本手动调用 (内置助手库 - 推荐)

baihu-panel提供了一套**零配置**的内建助手库（Built-in SDK），支持 Python 和 Node.js。除了支持极简的消息通知投递外，它还支持管理面板的**环境变量**与**定时任务控制**。

#### 1. 如何获取配置 Key？
在使用助手库前，请确保您已经在任务设置的“环境变量”或“机密”中配置了以下对应 Key：
- **消息推送相关**：
  - `BHPKG_NOTIFY_TOKEN`：进入「消息推送」->「脚本调用说明」标签，可以直接复制此处的 Token。
  - `BHPKG_NOTIFY_CHANNEL`：进入「消息推送」->「渠道列表」标签，可以查看每个渠道对应的 **ID**。
  - `BHPKG_NOTIFY_URL` (可选)：内置通知 API 的地址。默认为 `http://localhost:8052/api/v1/notify/send`。
- **环境与任务管理相关**：
  - `BHPKG_OPENAPI_TOKEN` (或 `OPENAPI_TOKEN`)：OpenAPI 鉴权 Token，在「系统设置」->「OpenAPI」中生成。
  - `BHPKG_OPENAPI_URL` (可选)：默认为本地面板 API 地址。

#### 2. 环境初始化
在开始编写脚本前，您需要在终端执行以下命令，为面板管理的所有语言环境安装 `baihu` 包：

```bash
baihu builtininstall
```
*该操作会将助手库安装到 mise 管理的所有版本中，确保 import 成功。*

#### 3. 代码示例

##### Python (同步调用)
```python
import baihu

# 消息通知
baihu.notify("任务标题", "通知正文内容")

# 环境变量与任务管理（详细用法见内置库示例）
envs = baihu.get_envs()
tasks = baihu.get_tasks()
```

##### Node.js (异步调用)
```javascript
const baihu = require('baihu');

// 消息通知
baihu.notify("任务标题", "通知正文内容");

// 环境变量与任务管理（详细用法见内置库示例）
(async () => {
    const envs = await baihu.getEnvs();
    const tasks = await baihu.getTasks();
})();
```

> [!TIP]
> 关于环境变量增删改查以及任务触发控制的完整 API 列表与更详尽的代码，请参考 [内置库示例](./examples/builtin.md)。


---

### 路径三：其他语言/高级调用 (原始 API)
> [!IMPORTANT]
> 以下示例中的端口均默认为 `8052`。如果您更改了容器内部的服务端口（通过 `BH_SERVER_PORT` 环境配置），请务必在调用时将 `8052` 替换为您的实际端口。

如果您使用 Shell 或其他尚未提供助手库的语言，可以通过标准 HTTP POST 请求调用。

#### 1. 快速获取代码
进入 **「消息推送」** -> **「脚本调用说明」** 标签，页面会根据您的配置自动生成包含 **通知 Token** 和 **默认渠道 ID** 的完整代码。

#### 2. 代码参考示例

##### Shell (Curl)
> **注意**：如果更改了容器内部的服务端口，请将 `8052` 替换为实际端口，或直接使用环境变量 `BHPKG_NOTIFY_URL`。

```bash
curl -X POST "http://localhost:8052/api/v1/notify/send" \
  -H "notify-token: 您的_NOTIFY_TOKEN" \
  -d '{"channel_id":"渠道ID", "title":"标题", "text":"内容"}'
```

##### 基础 Python (requests)
```python
import requests

def send_notify(title, content):
    url = "http://localhost:8052/api/v1/notify/send"
    headers = { "notify-token": "您的_NOTIFY_TOKEN" }
    data = {"channel_id": "您的_渠道_ID", "title": title, "text": content}
    requests.post(url, headers=headers, json=data)
```

---

## 消息中心管理

除了配置发送路径，您还可以在消息中心进行以下操作：

- **发送记录 (审计)**：实时记录每一条通过baihu-panel发送至外部的消息，方便追溯。
- **回执查询**：在 **「消息日志」** 页面查看到每条推送的详细状态，如果发送失败，会提供原始的错误响应代码以供排查。
