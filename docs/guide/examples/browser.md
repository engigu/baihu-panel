# 浏览器示例

`example/playwright` 提供了一组远程浏览器脚本示例，演示如何连接 **Browserless**。

> [!NOTE]
> 本示例以 Browserless 为主要演示对象。以此类推，您也可以使用其他的浏览器镜像（如原生的 `headless-shell` 或其他的浏览器集群服务）进行部署，只要它们支持 CDP 协议。

---

## 部署方式对比

baihu-panel强烈建议采用 **单独部署浏览器服务（如 Browserless）** 的方案，而不是在白虎镜像内部安装浏览器。

### 本地部署 (在白虎容器内安装) 的缺点
- **资源争抢**：浏览器是极度的“内存/CPU 杀手”，在同一容器内运行多任务极易导致白虎主进程因 OOM (内存溢出) 而崩溃。
- **镜像臃肿**：安装 Chromium 后，原本精简的 Docker 镜像体积会暴增数倍（增加 500MB+），导致拉取和更新缓慢。
- **依赖环境复杂**：在精简镜像中安装浏览器常会遇到各种缺失 `.so` 库文件的底层错误，排查极其困难。
- **不利于扩展**：无法实现多节点负载均衡，一个容器内的资源始终是有限的。

### 远程部署 (Browserless) 的优势
- **性能隔离**：浏览器的负载波动不会影响baihu-panel的稳定性。
- **开箱即用**：专业的浏览器镜像是针对性优化的，包含所有底层依赖和沙箱安全配置。
- **可视化调试**：大多数服务（如 Browserless）支持通过 VNC 同步查看浏览器画面，方便排查脚本逻辑。
- **弹性伸缩**：支持多会话、多实例模式，可以应对高并发爬虫需求。

---

## 准备工作

在baihu-panel中运行浏览器自动化脚本前，需要先配置好对应的 **语言环境** 与 **第三方依赖包**。

### 1. Node.js 环境 (JavaScript)
如果您使用 `playwright.js` 脚本：
- **依赖安装**：前往「语言依赖」->「Node.js」，安装 `puppeteer-core`。
- **说明**：该脚本使用 `puppeteer-core` 通过 CDP 协议连接远程浏览器，无需安装完整的 puppeteer 及其内置浏览器。

### 2. Python 环境
如果您使用 `playwright.py` 脚本：
- **版本推荐**：建议在「语言环境」中安装并使用 **Python 3.11**。
    > [!TIP]
    > 建议避开更高版本的 Python（如 3.12+），因为目前部分 Playwright 依赖在极新版本的 Python 环境下可能会遭遇编译或安装失败。
- **依赖安装**：前往「语言依赖」->「Python」，安装 `playwright`。

---

配置完成后，即可创建定时任务并关联对应的脚本文件。

## Browserless 连接要点

如果你当前是通过 Browserless 连接远程浏览器：

- 不要执行 `playwright install`
- 不需要额外下载 Chromium / Firefox / WebKit
- 直接使用 `connect_over_cdp` 连接远程浏览器即可

## 适用场景

- 验证白虎到 Browserless 的网络连通性
- 验证远程浏览器地址和 Token 是否配置正确
- 快速测试浏览器自动化脚本是否能正常运行
- 快速确认 Node.js / Python 语言环境是否已经配置完成
- 作为后续网页自动化脚本的基础模板

## 推荐部署方式

建议配合 `browserless/chromium` 一起使用，再由白虎中的脚本连接远程浏览器服务。

参考 `docker-compose.yml`：

```yaml
version: "3.8"

services:
  browser:
    image: ghcr.io/browserless/chromium:latest
    container_name: browser
    restart: unless-stopped
    environment:
      MAX_CONCURRENT_SESSIONS: 5
      MAX_QUEUE_LENGTH: 20
      CONNECTION_TIMEOUT: 300000
      DEFAULT_LAUNCH_ARGS: '["--no-sandbox","--disable-setuid-sandbox","--disable-dev-shm-usage"]'
      TOKEN: your-secret-token
      ENABLE_DEBUGGER: "false"
    shm_size: "1gb"
    mem_limit: 2g
    cpus: 2
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:3000/healthz"]
      interval: 30s
      timeout: 10s
      retries: 3

  baihu:
    image: ghcr.io/engigu/baihu:latest
    container_name: baihu
    restart: unless-stopped
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
    depends_on:
      - browser
```

## 使用步骤

1. 启动 `browser` 和 `baihu` 服务。
2. 在白虎的“语言依赖”中安装对应包。
3. 按实际环境修改脚本中的 Browserless 地址和 Token。
4. 在白虎中创建任务并运行对应脚本。

Python 示例的核心写法如下：

```python
from playwright.sync_api import sync_playwright

with sync_playwright() as p:
    browser = p.chromium.connect_over_cdp(
        "http://browser:3000?token=your-secret-token"
    )
    page = browser.new_page()
    page.goto("https://www.baidu.com")
    page.screenshot(path="baidu.png")
    browser.close()
```

## 运行前检查

- Browserless 服务已经正常启动
- 白虎可以访问 Browserless 地址
- `TOKEN` 与 Browserless 配置保持一致
- 脚本中的地址、端口和协议填写正确

## 常见问题

### 1. 报错提示找不到模块

通常是还没有在“语言依赖”中安装对应包：

- Node.js 示例需要 `puppeteer-core`
- Python 示例需要 `playwright`

### 2. 为什么没有执行 `playwright install`

这是预期行为。

如果你使用的是 Browserless 这类远程浏览器服务，Playwright 只是作为客户端发起连接，不需要在白虎容器里再下载本地浏览器，所以通常不要执行 `playwright install`。

### 3. 连接不上 Browserless

请优先检查：

- Browserless 服务是否正常启动
- Token 是否正确
- 白虎与 Browserless 是否在同一网络中
- 地址是否写成了当前运行环境可访问的地址

### 4. 页面打开超时

可以尝试：

- 换一个更稳定的目标站点
- 调大超时时间
- 增加 Browserless 容器的 `shm_size`
- 检查容器 CPU / 内存是否不足

### 5. 没有看到截图文件

请确认：

- 脚本已经执行成功
- 截图保存路径是否正确
- 任务工作目录是否符合预期

## 说明

这组示例主要用于快速验证白虎与远程浏览器服务之间的连通性，以及对应语言环境是否已经配置完成。
