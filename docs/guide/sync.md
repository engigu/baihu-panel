# 仓库同步 (Repo)

仓库同步允许baihu-panel直接以 Git 仓库的形式管理和更新脚本库，极大地方便了脚本的大规模分发与自动化部署。

## 同步源管理

- **baihu 指令同步**：可直接在终端通过 `baihu reposync` 指令添加同步源，例如：
    ```bash
    baihu reposync --source-url https://github.com/example/repo.git --target-path /app/data/scripts/example_repo
    ```
    完整参数说明见下文「[baihu 命令行指令](#baihu-命令行指令)」章节。
- **青龙 (QL) 指令解析**：如果您曾经是青龙面板的用户，您可以直接粘贴类似的 `ql repo <url> <whitelist> <blacklist> <dependence> <branch>` 指令，系统将自动提取各项参数。
- **Git 源管理**：支持从 **GitHub**, **GitLab**, **Gitee** 等主流代码托管平台同步脚本。
- **Token 访问**：支持私有仓库的访问，详细配置方式见下文「[私有仓库访问认证](#私有仓库访问认证)」。

### 私有仓库访问认证

同步源支持通过 **Token** 访问私有仓库：在编辑仓库源时填写「身份认证」输入框（对应命令行参数 `--auth-token`），即可访问私有仓库。支持以下两种 Token 形式：

- **HTTPS 访问 Token**：在 GitHub / GitLab / Gitee 生成的 Personal Access Token。系统会自动将 Token 拼接到 Git 地址中（`https://<token>@github.com/owner/private-repo.git`），或通过 `Authorization: token <token>` 请求头完成鉴权。
- **API 下载 Token**：访问私有仓库 API 或 raw 文件直链下载时使用的鉴权凭据。

### 环境变量注入

同步任务运行时会继承面板主程序的全部环境变量，并原样透传给 `baihu reposync` 及底层 `git` 命令。以下变量会参与同步过程：

| 环境变量 | 作用 | 注入方式 |
| :--- | :--- | :--- |
| `CURR_REPO_DIR` | 当前仓库的实际物理目录，供同步前置/后置指令使用 | **自动注入** |
| `BH_SCRIPTS_DIR` | 脚本根目录的绝对路径（默认 `data/scripts`，可用环境变量覆盖）。仓库同步时，`--target-path` 中的 `$SCRIPTS_DIR$` 占位符被替换为该值 | **自动注入** |
| `http_proxy` / `https_proxy` | Git 网络代理，仅当在仓库源中配置了「HTTP 代理」时由系统自动设置 | **自动注入** |

> [!IMPORTANT]
> - 上表变量均为系统内置注入，无需任何配置。
> - **注入链路**：「变量机密」→ 任务执行进程（自动合并）→ `baihu reposync`（继承全部环境变量）→ `git` 命令。

## 文件过滤规则

同步完成后，系统会对仓库目录执行一次**脚本过滤**：依据依赖保护、扩展名、黑名单、白名单四类规则逐个检查文件，删除不符合要求的文件（并清理产生的空目录）。检查按以下优先级顺序进行：

```
dependence 命中      → 无条件保留（优先级最高，直接跳过后续检查）
extensions 后缀不符  → 删除
blacklist 命中       → 删除
whitelist 未命中     → 删除
```

### 白名单模式（--whitelist-paths）

白名单是**排除式筛选（强过滤）**，语义是"这个仓库里我只要这些"。一旦配置了白名单，**只有命中白名单关键字（或 `--dependence` 保护）的文件会被保留，其余文件一律删除**。

- **配置格式**：多个关键字用逗号 `,` 或竖线 `|` 分隔，例如 `daily|checkin`。
- **匹配方式**：每个关键字优先作为**正则表达式**整体匹配（自动忽略大小写），依次对**文件名、相对路径**进行匹配；若关键字不是合法正则，则回退为**大小写不敏感的包含匹配**。
- **典型场景**：仓库包含大量脚本，只想保留某个子目录或特定命名的脚本（此时需用 `--dependence` 同时保住 `notify.js` 等公共依赖）。

#### 白名单的执行流程

配置白名单后，一次同步中白名单文件会经历以下四个阶段：

| 阶段 | 动作 | 白名单路径的效果 |
| :--- | :--- | :--- |
| ① 同步前 | `preserve` 将命中白名单的本地文件**临时移出**目标目录（重命名至 `.baihu_sync_preserve_*` 临时目录） | 目录中暂时不再有这些路径 |
| ② 同步中 | `git fetch / checkout / reset`（或 url 下载）拉取仓库最新内容 | 若远端存在同名文件，会被重新检出 |
| ③ 同步后 | `restore` 将移出的本地文件**移回原位，覆盖**第②步检出的版本 | 白名单路径**以本地版本为准**，远端更新不生效 |
| ④ 过滤阶段 | `filterFiles` 依据 依赖 / 扩展名规则/ 白名单 / 黑名单 删除文件 | 白名单文件命中白名单 → **保留** |

## 自动新增任务

- **自动解析配置**：在同步代码至本地物理磁盘后，面板将深度扫描每个支持扩展名的文件（默认为 `.js`、`.py`、`.ts`、`.sh`，也可通过 `extensions` 参数自定义扩展名集合；不支持的扩展名不会被解析注册）。
- **配置探测**：
    - `name: "任务名称"`：解析脚本定义的展示名。
    - `cron: "0 0 * * *"`：自动提取脚本头部的 Cron 定时规则。

## 扫描与注册规则

同步代码到本地后，面板会深度扫描每个脚本文件，**自动解析脚本头部的注释信息并注册为定时任务**，无需逐个手动创建。

### 脚本中需要添加的注释

以下写法需出现在**脚本文件的头部（前 15 行内）**，即可被自动识别。其中任务名称（`new Env(...)` / `Env(...)`）**不限于注释，直接写在代码行同样生效**（如常见的 `var $ = new Env('任务名称')`）。

#### 1. 任务名称

| 写法 | 说明 |
| :--- | :--- |
| `name: "任务名称"` | 青龙同样推荐的写法，写到注释里 |
| `new Env('任务名称')` | 兼容青龙生态的写法；注释或代码行均可 |
| `Env('任务名称')` | 同上，不带 `new` 关键字；注释或代码行均可 |

未匹配到以上格式时，将依次回退使用脚本头部第一行非空注释、文件名作为任务名称。

#### 2. 定时规则 (Cron)

| 写法 | 示例 |
| :--- | :--- |
| `cron: "..."` | `// cron: "0 5 * * *"` |
| `cron "..."` | `// cron "0 5 * * *"` |
| `cron = '...'` | `// cron = '0 5 * * *'` |
| 关联文件名的 Cron | `// 0 0 * * * my_task.js` |

支持 5 位（分 时 日 月 周）或 6 位（秒 分 时 日 月 周）的标准 Cron 表达式。

### 完整示例

**写法一：`name:` + `cron:`（简单直观）**

```javascript
/*
 * name: "每日签到"
 * cron: "0 8 * * *"
 */

function main() {
  // 业务逻辑
}
```

同步完成后，面板会自动创建一个名为「每日签到」、定时规则为 `0 8 * * *` 的任务。

**写法二**

```javascript
/*
 * new Env('京东抽奖')
 * cron "0 5 * * *"
 */

function main() {
  // 业务逻辑
}
```

**写法三：`Env(...)`（不带 `new`）+ `cron = '...'`**

```javascript
/*
 * Env('微信读书')
 * cron = '0 0 * * *'
 */

function main() {
  // 业务逻辑
}
```

**写法四：关联文件名的 Cron**

```javascript
// 0 0 * * * daily_clean.js
```

同步后名为 `daily_clean.js` 的脚本会被注册为 `0 0 * * *` 的定时任务（任务名称回退为文件名）。

**写法五：代码行中的 `var $ = new Env(...)`（无需注释）**

```javascript
var $ = new Env('每日签到')
// cron: "0 8 * * *"

function main() {
  // 业务逻辑
}
```

任务名称不限于注释，写在代码行同样生效；Cron 规则仍写在头部注释中即可。

---

## baihu 命令行指令

仓库同步与任务注册相关操作也可以通过内置的 `baihu` 命令行工具完成，适合定时任务、容器内运维等纯终端场景。

### 仓库同步 (reposync)

`baihu reposync` 是同步命令的核心，面板内的仓库任务本质上就是在调度该命令。

| 参数 | 默认值 | 描述 |
| :--- | :--- | :--- |
| `--source-type` | `git` | 同步源类型，可选 `git`（Git 仓库）或 `url`（文件直链下载）。 |
| `--source-url` | | 同步源地址，Git 仓库地址或下载 URL。 |
| `--target-path` | | 目标保存路径，支持变量替换（如 `$SCRIPTS_DIR$`）。 |
| `--branch` | | Git 分支名，留空时自动检测远程默认分支。 |
| `--path` | | 稀疏检出（Sparse checkout）的指定路径，或单文件模式下的相对路径。 |
| `--single-file` | `false` | 是否开启单文件模式，仅从 Git 提取指定单个文件。 |
| `--proxy` | `none` | Github 加速代理类型，可选 `none`、`ghproxy`、`mirror`、`custom`。 |
| `--proxy-url` | | 自定义代理地址，仅在 `--proxy=custom` 时生效。 |
| `--auth-token` | | 私有仓库或 API 访问使用的鉴权 Token。 |
| `--http-proxy` | | HTTP/HTTPS 代理地址，例如 `http://127.0.0.1:7890`。 |
| `--whitelist-paths` | | 白名单关键字（逗号或竖线分隔），为**强过滤**模式：一旦填写，目录最终仅保留命中白名单（及 `--dependence`）的文件。同时命中路径会在 git 同步前被临时移出、同步后恢复，防止被强制 `reset` 覆盖（本地版本优先）。匹配优先按正则（忽略大小写）匹配文件名/相对路径，回退为包含匹配。详见「[文件过滤规则](#文件过滤规则)」。 |
| `--blacklist` | | 黑名单关键字（竖线 `\|` 分隔），包含该关键字的文件将会被过滤删除。 |
| `--dependence` | | 脚本依赖文件关键字（竖线 `\|` 分隔）。命中关键字的文件**无条件保留**（过滤优先级最高，即使同时命中黑名单、扩展名不符或不在白名单也不会被删除），用于保护 `notify.js` 等公共依赖文件；注意它只做保留保护，不会主动安装依赖。 |
| `--extensions` | | 允许的脚本扩展名（竖线 `\|` 分隔，如 `.js\|.py`）。 |
| `--commenttotask` | `false` | 是否启用青龙 (QL) 格式的脚本注释解析（`true`/`false`）。仅对标准 Git 源生效；青龙源始终解析。注意：真正新增任务的开关是任务配置的「自动添加任务」（`auto_add_cron`），且需配合 `--task-id` 使用。 |
| `--pre-command` | | 自动发现任务的默认前置命令。 |
| `--post-command` | | 自动发现任务的默认后置命令。 |
| `--repo-name` | | 自定义仓库目录名。 |
| `--task-timeout` | `30` | 同步任务的超时时间，单位为分钟。 |

使用示例：

```bash
# 1. 基础同步：拉取仓库到指定目录
baihu reposync --source-url https://github.com/example/repo.git \
  --target-path /app/data/scripts/example_repo

# 2. 同步并自动新增任务：开启脚本注释解析（new Env / cron）
baihu reposync --source-url https://github.com/example/repo.git \
  --target-path /app/data/scripts/example_repo \
  --commenttotask true

# 3. 带代理与过滤条件，且自动新增任务
baihu reposync --source-url https://github.com/example/repo.git \
  --target-path /app/data/scripts/example_repo \
  --proxy ghproxy \
  --extensions ".js|.py" \
  --blacklist "test|mock" \
  --commenttotask true
```

---

## 增量同步

- **Git 离线拉取**：支持增量更新，仅下载变更部分，降低带宽压力。
- **分支切换**：支持指定任意分支进行同步，方便用户在生产与测试环境间切换脚本源。
- **稀疏检出 (Sparse Checkout)**：如果仓库过于庞大，您可以配置仅同步特定的子文件夹以节省存储空间。
