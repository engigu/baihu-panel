# Linux 系统依赖处理

在使用baihu-panel时，您可能会在运行某些脚本时遇到缺少底层 Linux 系统级依赖（例如 `apt` 或 `apk` 包）的情况。这篇指南将详细讲解如何优雅、持久地解决这些依赖问题。

## 背景与痛点

baihu-panel通常以 Docker 容器的形式运行。Docker 的文件系统具有以下特性：
- **挂载目录（持久化）**：像 `data/` 这样的目录被映射到了宿主机，其中的数据（如脚本、日志、配置文件）在重启或升级镜像时会保留。
- **容器层（非持久化）**：容器自身的系统目录（如 `/usr/bin`, `/lib`, `/etc`）是临时层。如果您直接在终端里手动执行 `apt-get install xxx`，虽然当下可以立即使用，**但在容器被销毁重建或更新镜像后，这些刚安装的系统包就会全部丢失。**（注意：仅仅是普通的 `docker restart` 重启容器并不会丢失，只有重建容器时才会重置）

## 核心解决思路

为了解决依赖丢失的问题，baihu-panel提供了一种自动化的解决方案：**利用 `baihu_startup`（开机触发）类型的定时任务，在面板每次启动时自动执行一段依赖安装脚本。**

这样，无论您如何更新镜像或重启容器，系统依赖都能在面板核心服务就绪前自动被补充安装，并且对后续的普通脚本任务透明。

---

## 具体操作步骤

### 第一步：编写依赖安装脚本

首先，在您的脚本目录（通常为 `data/scripts` 下，或者您可以单独建一个 `data/scripts/deps` 目录）创建一个 Shell 脚本，例如 `install_my_deps.sh`。

由于baihu-panel的镜像目前均基于 Debian 系统，您可以直接在脚本中使用 `apt` 或 `apt-get` 命令来管理系统依赖。

**示例 1：安装 Puppeteer (无头浏览器) 的依赖动态库**
```bash
#!/bin/bash
# 遇到错误即停止执行
set -e

echo "正在检测并安装 Puppeteer 依赖..."

# 提前 update 索引是非常重要的一步
apt-get update
apt-get install -y libnss3 libnspr4 libatk1.0-0 libatk-bridge2.0-0 libcups2 libdrm2 libxkbcommon0 libxcomposite1 libxdamage1 libxfixes3 libxrandr2 libgbm1 libasound2

echo "Puppeteer 依赖安装完成！"
```

**示例 2：安装 Python/C++ 编译所需的基础工具链**
```bash
#!/bin/bash
set -e

apt-get update
# 安装 gcc, g++, make 以及 python3 相关的头文件
apt-get install -y build-essential python3-dev
```

**示例 3：带 Hash 检查的高阶依赖安装脚本（推荐）**
此脚本利用 `/tmp` 目录和脚本自身内容的哈希值，完美匹配 Docker 容器的生命周期，避免在普通重启时无意义地检测。
```bash
#!/bin/bash
set -e

# 生成当前脚本的哈希标识
SCRIPT_HASH=$(md5sum "$0" | awk '{print $1}')
FLAG_FILE="/tmp/deps_installed_${SCRIPT_HASH}"

# 如果标识文件存在，说明在此容器生命周期内已安装过，且脚本未被修改，直接退出
if [ -f "$FLAG_FILE" ]; then
    echo "系统依赖已就绪，跳过安装。"
    exit 0
fi

echo "开始安装系统环境依赖..."
apt-get update
# 假设您需要安装 ffmpeg 和 imagemagick
apt-get install -y ffmpeg imagemagick

# 安装成功后写入标识文件
touch "$FLAG_FILE"
echo "系统依赖安装完成！"
```

### 第二步：配置开机触发任务

脚本编写并保存到面板后，接下来只需将其配置为开机任务：

1. 进入面板的 **「定时任务」** 页面，点击 **「新建任务」**。
2. **任务名称**: 填写容易辨识的名称，例如 “安装系统底层依赖”。
3. **执行命令**: 输入执行该脚本的命令，例如 `bash deps/install_my_deps.sh` （假设您将脚本放在了 `deps` 文件夹下）。
4. **触发类型**: 在下拉菜单中选择 **`baihu_startup` (开机触发)**。
5. **保存** 任务。

现在，你可以尝试在终端中执行一下该任务验证脚本是否无误。一旦无误，未来每次容器重启，面板都会自动在后台静默执行这个任务，确保环境完备。

---

## 官方预设示例：PHP 编译依赖

为了方便用户参考，我们在项目源码中内置了一个更完善的依赖安装脚本示例。

通过 `mise` 安装某些 PHP 版本时，系统会尝试从源码编译，这就需要用到 `autoconf`, `bison`, `pkg-config` 等工具。

如果您在安装 PHP 时遇到 `autoconf not found` 或 `buildconf failed`，可以直接使用项目根目录下的预设示例脚本：
- **路径位置**: `example/deps/install_php_env_deps.sh`
- **使用方法**: 新建 `baihu_startup` 触发类型的任务，执行命令填写 `bash example/deps/install_php_env_deps.sh` 即可。

此示例脚本中还包含了“检测是否已安装再决定是否执行 apt install”的逻辑，您可以查阅其源码作为编写自己依赖脚本的最佳实践参考。

---

## 💡 注意事项与进阶建议

1. **幂等性 (Idempotency)**：开机脚本在每次重启时都会在后台异步执行。像 `apt-get install -y` 这种命令天然是幂等的（如果已安装就不会重新下载），虽然它不会阻塞面板的启动速度，但每次无意义地检查和刷新软件源仍会白白占用开机初期的系统资源。建议您在脚本中先用 `dpkg -l <包名>` 或 `command -v <命令>` 判断依赖是否存在，不存在时再执行安装。
   - **进阶技巧**：您也可以在依赖安装完成后，向 `/tmp` 目录下写入一个带有当前脚本内容 Hash 值的标识文件（例如 `touch /tmp/deps_installed_$(md5sum "$0" | awk '{print $1}')`）。在脚本开头判断该文件是否存在，若存在则直接退出。由于容器被销毁重建时 `/tmp` 目录和您安装的系统依赖会一并丢失，而在普通的重启中它们又会一并保留，这种方式完美契合了容器的临时层生命周期，能避免反复执行依赖检测逻辑，进一步加速开机任务。
2. **网络环境**： Docker 镜像已经**默认将 APT 源替换为了清华源 (TUNA)**，因此在国内网络下执行 `apt-get` 也能获得很快的下载速度，您通常不需要在脚本中再次手动替换源。
3. **避免冲突**：请仅安装您脚本运行强依赖的底层库，尽量不要通过 `apt` 安装 Node.js 或 Python 的运行环境，这些高级语言环境应交由面板的 **「编程语言」** (Mise) 模块统一管理。
