# 前端定制 (WebUI)

baihu-panel支持完全接管和替换默认系统面板界面。你可以开发自己专属的前端主题，甚至添加自定义的前端交互功能，并打包为独立的 WebUI 资源包上传至系统应用。

> [!IMPORTANT]
> **安全与一致性维护声明**：
> 更换前端包后，系统无法自动保障自定义前端的安全性，亦无法确保其与后续更新的后端 API 接口始终保持一致。**自定义前端包的更新、向后兼容维护与漏洞修复需完全由该前端资源提供者（或开发者）负责**。

---

## 快速使用

### 1. 网页端上传与切换
1. 进入系统后，点击导航栏的 **系统设置**。
2. 切换到 **前端定制** 面板。
3. 点击右上角的 **上传前端资源包**，选择你打包好的 `.zip`、`.tar.gz` 或 `.tgz` 格式的前端资源包。
4. 上传成功后，列表会显示该包的信息（名称、版本、作者、状态等）。
5. 点击操作栏中的 **启用** 按钮，系统将自动重载并切换至你的自定义前端包。

> [!WARNING]
> 自定义前端包若存在 Bug 或打包不完整可能导致界面白屏。如果不慎应用了错误或不兼容的包，请使用下方命令行工具恢复。

### 2. 命令行 (CLI) 运维
当界面因异常白屏无法访问时，可以进入baihu-panel容器/服务器终端，使用 `baihu webui` 命令一键管理或恢复：

- **一键恢复默认内置界面**：
  ```bash
  baihu webui reset
  ```
- **查看已安装的资源包列表**：
  ```bash
  baihu webui list
  ```
- **手动切换/启用前端包**：
  ```bash
  baihu webui set <包名>
  ```
- **删除指定的前端包**：
  ```bash
  baihu webui delete <包名>
  ```

---

## 开发自定义前端

baihu-panel的前端架构是完全解耦的。你可以选择以下两种方式之一来定制专属前端：

### 方式一：基于现有代码二次开发（推荐）
如果你只是想修改部分样式、布局，或者在原有功能基础上增加新特性，最简单的方式是 **Fork `baihu-panel` 项目**。
1. Fork 本项目并克隆代码到本地。
2. 直接在项目的 `web/` 目录下，对现有的 Vue3 源码进行修改与定制。
3. 修改完成后，利用项目自带的 Makefile 打包命令（见下方说明）一键将你的修改编译为独立的 WebUI 资源包。

### 方式二：从零开始全新开发
如果你想用自己熟悉的技术栈（如 React, Angular，或者是纯静态的 HTML/JS）完全重写整个面板，这也是完全支持的！你只需要按照下方的核心规范进行开发和打包即可。
### 1. 核心校验规则
baihu-panel后端提取并启用前端资源时，会执行以下强校验：
1. **压缩包根目录下必须包含 `index.html`**：作为单页应用 (SPA) 的静态入口文件。
2. **压缩包根目录下必须包含 `uimanifest.json`**：声明该前端包的元数据信息。

### 2. 配置文件 `uimanifest.json` 规范
在前端打包产物的根目录下（与 `index.html` 同级），必须创建一个 `uimanifest.json` 文件。格式示例如下：

```json
{
  "name": "custom-neon-theme",
  "version": "1.0.2",
  "author": "YourName",
  "description": "baihu-panel霓虹暗黑风定制前端主题",
  "min_panel_version": "1.0.0"
}
```

*注：`name` 字段不能设置为 `"default"`（default 被保留作为内置前端的系统标识）。*

### 3. API 请求地址与开发环境代理
在独立开发自定义前端时，需要配置请求与后端的通信地址及代理：

- **后端默认服务地址与端口**：
  baihu-panel后端服务默认运行在端口 `8052` 上，本地调试 API 的基础 URL 通常为：
  `http://127.0.0.1:8052/api/v1`
  
- **本地开发环境代理配置（以 Vite 为例）**：
  为了避免跨域问题（CORS），推荐在前端开发服务器中设置代理。在 `vite.config.ts` 中配置示例如下：
  ```typescript
  export default defineConfig({
    server: {
      proxy: {
        '/api/v1': {
          target: 'http://127.0.0.1:8052', // 本地运行的baihu-panel后端地址
          changeOrigin: true
        }
      }
    }
  })
  ```

- **生产环境线上适配（相对路径）**：
  前端包部署生效后，与后端处于同端口同域名下。后端会在返回的 `index.html` 的 `<head>` 中自动注入以下配置变量：
  ```html
  <script>
    window.__BASE_URL__ = "";          // 部署子路径前缀 (根据实际反代配置)
    window.__API_VERSION__ = "/api/v1"; // API 接口版本前缀
  </script>
  ```
  建议在封装 Axios 或 Fetch 时，直接通过浏览器环境变量拼接相对路径作为 API 地址：
  ```typescript
  const baseURL = `${window.location.origin}${window.__BASE_URL__ || ''}${window.__API_VERSION__ || '/api/v1'}`;
  ```

- **接口定义与类型参考**：
  默认系统中已经定义好了所有的后端 API 接口签名、传参格式以及 TS 类型声明。你在二次开发或自定义前端时，可以直接参考项目源码中的前端接口定义文件： `web/src/api/index.ts`。

---

## 打包前端资源包(现成)

你可以利用baihu-panel项目自带的 `Makefile` 脚本，在现在的前端页面进行修改，将开发好的前端项目快速编译打包成标准的 `.tar.gz` 前端资源包, 自己使用或者分享使用。

### 使用 Makefile 打包

在项目根目录下，运行以下指令（参数必须填写完整）：

```bash
make pack-webui NAME=neon-theme VERSION=1.0.2 AUTHOR=MyName DESC="霓虹定制主题包"
```

该指令会自动执行以下步骤：
1. 进入 `web/` 目录并安装依赖；
2. 编译构建前端静态资源（默认输出到 `web/dist`）；
3. 在 `web/dist` 中自动按参数生成校验所需的 `uimanifest.json`；
4. 将该目录下所有文件使用 `tar` 命令进行 gzip 压缩打包；
5. 输出归档文件在项目根目录的 `bin/webui-neon-theme-1.0.2.tar.gz`，此包即可直接在面板中上传安装。
