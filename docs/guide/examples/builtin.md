# 内置库示例

baihu-panel提供了一个名为 `baihu` 的内建包（Built-in SDK），支持 Python 和 Node.js。通过该内置库，您可以在脚本中实现**消息推送**、**环境变量管理**以及**任务执行控制**等高级功能。

---

## 准备工作

在运行内置库脚本之前，请确保完成了以下步骤：

### 1. 安装内置包
在baihu-panel的「终端」页面中，或者通过创建临时任务执行以下命令，为面板管理的所有语言环境安装 `baihu` 包：

```bash
baihu builtininstall
```

### 2. 配置环境变量
根据您需要调用的功能，在定时任务的“环境变量”或“机密”中配置以下对应 Key：

#### 消息推送所需环境变量
- **`BHPKG_NOTIFY_TOKEN`**：进入「消息推送」->「脚本调用说明」页面即可找到。
- **`BHPKG_NOTIFY_CHANNEL`**：进入「消息推送」->「渠道列表」页面，查看对应渠道的 **ID**。
- **`BHPKG_NOTIFY_URL`** (可选)：默认为 `http://localhost:8052/api/v1/notify/send`。如果修改了主服务端口，需要同步修改。

#### 环境变量管理与定时任务控制所需环境变量
- **`BHPKG_OPENAPI_TOKEN`** (或 `OPENAPI_TOKEN`)：用于 OpenAPI 接口鉴权，进入「系统设置」->「OpenAPI」页面，生成并复制 Token。
- **`BHPKG_OPENAPI_URL`** (或 `OPENAPI_URL`，可选)：默认为本地面板 API 地址。若在非标准环境下运行，可手动指定（例如 `http://localhost:8052`）。

---

## 消息通知示例

只需要一行代码即可触发零配置推送。

::: code-group

```python [Python]
import baihu

def main():
    print("正在尝试发送 Python 内建通知...")
    try:
        # 调用内置 notify 函数
        # 内部会自动使用环境变量进行鉴权和投递
        response = baihu.notify(
            title="Python 任务提醒",
            text="这是一条来自 Python 示例脚本的通知消息。调用非常简单！"
        )
        print("发送请求已处理。")
        if response:
            print(f"服务器响应: {response}")
            
    except Exception as e:
        print(f"发送过程发生异常: {e}")

if __name__ == "__main__":
    main()
```

```javascript [Node.js]
const baihu = require('baihu');

console.log("正在尝试发送 Node.js 内建通知...");

try {
    // 简单的一行代码即可完成推送，内置包采用异步非阻塞发送
    baihu.notify(
        "Node.js 任务提醒", 
        "这是一条来自 Node.js 示例脚本的通知消息。无需配置 API 地址或 Token。"
    );
    console.log("发送请求已提交。");
    
} catch (e) {
    console.error(`通知失败: ${e.message}`);
}
```

:::

---

## 环境变量管理

内置库支持对面板的环境变量进行增删改查。

### 支持方法

* **Python**: 
  - `get_envs()`: 获取所有环境变量列表。
  - `get_env(name)`: 根据变量名称获取详情。
  - `add_env(name, value, remark)`: 添加新的环境变量。
  - `update_env(id, name, value, remark)`: 更新指定 ID 的环境变量值。
  - `delete_env(id)`: 根据 ID 删除环境变量。
* **Node.js**:
  - `getEnvs()`: 获取所有环境变量列表。
  - `getEnv(name)`: 根据变量名称获取详情。
  - `addEnv(name, value, remark)`: 添加新的环境变量。
  - `updateEnv(id, name, value, remark)`: 更新指定 ID 的环境变量值。
  - `deleteEnv(id)`: 根据 ID 删除环境变量。

### 代码示例

::: code-group

```python [Python]
import baihu

def main():
    print("====== 开始运行 Python 环境变量管理示例 ======")
    try:
        # 1. 获取全部环境变量
        envs = baihu.get_envs()
        print(f"当前共有 {len(envs)} 个环境变量")
        
        # 2. 新增一个临时环境变量
        new_env_name = "BHPKG_TEST_KEY"
        new_env_val = "HelloBaihu"
        print(f"正在创建环境变量: {new_env_name}...")
        created_env = baihu.add_env(
            name=new_env_name,
            value=new_env_val,
            remark="Python SDK 测试自动创建"
        )
        print(f"创建成功: ID={created_env.get('id')}, Name={created_env.get('name')}")

        # 3. 查询刚才创建的环境变量详情
        checked_env = baihu.get_env(new_env_name)
        if checked_env:
            print(f"成功查询到变量: {checked_env.get('name')} = {checked_env.get('value')}")

            # 4. 修改该环境变量的值
            updated_val = "HelloBaihu_Updated"
            print(f"正在修改环境变量的值为: {updated_val}...")
            updated_env = baihu.update_env(
                id=checked_env.get("id"),
                name=new_env_name,
                value=updated_val,
                remark="Python SDK 测试自动更新"
            )
            print(f"更新成功: Value={updated_env.get('value')}")

            # 5. 删除该临时环境变量
            print(f"正在删除临时环境变量: ID={checked_env.get('id')}...")
            baihu.delete_env(checked_env.get("id"))
            print("删除成功！")
            
    except Exception as e:
        print(f"环境变量操作失败: {e}")
        print("提示: 请确保在面板任务设置中正确注入了 OpenAPI Token。")

if __name__ == "__main__":
    main()
```

```javascript [Node.js]
const baihu = require('baihu');

async function main() {
    console.log("====== 开始运行 Node.js 环境变量管理示例 ======");
    try {
        // 1. 获取全部环境变量
        const envs = await baihu.getEnvs();
        console.log(`当前共有 ${envs.length} 个环境变量`);

        // 2. 新增一个临时环境变量
        const newEnvName = "BHPKG_TEST_KEY_JS";
        const newEnvVal = "HelloBaihuJS";
        console.log(`正在创建环境变量: ${newEnvName}...`);
        const createdEnv = await baihu.addEnv(
            newEnvName,
            newEnvVal,
            "Node.js SDK 测试自动创建"
        );
        console.log(`创建成功: ID={createdEnv.id}, Name={createdEnv.name}`);

        // 3. 查询该环境变量
        const checkedEnv = await baihu.getEnv(newEnvName);
        if (checkedEnv) {
            console.log(`成功查询到变量: ${checkedEnv.name} = ${checkedEnv.value}`);

            // 4. 修改该环境变量的值
            const updatedVal = "HelloBaihuJS_Updated";
            console.log(`正在修改环境变量的值为: ${updatedVal}...`);
            const updatedEnv = await baihu.updateEnv(
                checkedEnv.id,
                newEnvName,
                updatedVal,
                "Node.js SDK 测试自动更新"
            );
            console.log(`更新成功: Value=${updatedEnv.value}`);

            // 5. 删除该临时环境变量
            console.log(`正在删除临时环境变量: ID={checkedEnv.id}...`);
            await baihu.deleteEnv(checkedEnv.id);
            console.log("删除成功！");
        }

    } catch (e) {
        console.error(`环境变量操作失败: ${e.message}`);
        console.log("提示: 请确保在面板任务设置中正确注入了 OpenAPI Token。");
    }
}

main();
```

:::

---

## 定时任务管理与控制

内置库支持查询面板的任务列表、最近的执行结果以及手动触发特定任务的运行。

### 支持方法

* **Python**: 
  - `get_tasks()`: 获取所有定时任务列表。
  - `execute_task(id)`: 立即触发指定 ID 任务的运行。
  - `get_last_results()`: 获取最近任务的执行记录。
* **Node.js**:
  - `getTasks()`: 获取所有定时任务列表。
  - `executeTask(id)`: 立即触发指定 ID 任务的运行。
  - `getLastResults()`: 获取最近任务的执行记录。

### 代码示例

::: code-group

```python [Python]
import baihu

def main():
    print("====== 开始运行 Python 任务管理与执行控制示例 ======")
    try:
        # 1. 获取所有任务列表
        tasks = baihu.get_tasks()
        print(f"成功获取到 {len(tasks)} 个定时任务:")
        for task in tasks[:5]:  # 仅打印前5个
            print(f"  - [{task.get('id')}] {task.get('name')} (表达式: {task.get('schedule')}, 备注: {task.get('remark')})")

        # 2. 尝试触发第一个任务的运行
        if tasks:
            target_task = tasks[0]
            print(f"\n尝试手动触发任务运行: [{target_task.get('id')}] {target_task.get('name')}...")
            baihu.execute_task(target_task.get("id"))
            print("执行指令发送成功。")

        # 3. 获取最近的执行结果列表
        results = baihu.get_last_results()
        print(f"\n最近共有 {len(results)} 条任务执行记录。")
            
    except Exception as e:
        print(f"任务操作失败: {e}")
        print("提示: 请确保在面板任务设置中正确注入了 OpenAPI Token。")

if __name__ == "__main__":
    main()
```

```javascript [Node.js]
const baihu = require('baihu');

async function main() {
    console.log("====== 开始运行 Node.js 任务管理与执行控制示例 ======");
    try {
        // 1. 获取所有任务列表
        const tasks = await baihu.getTasks();
        console.log(`成功获取到 ${tasks.length} 个定时任务:`);
        tasks.slice(0, 5).forEach(task => { // 仅展示前5项
            console.log(`  - [${task.id}] ${task.name} (表达式: ${task.schedule || ''}, 备注: ${task.remark || ''})`);
        });

        // 2. 尝试触发第一个任务的运行
        if (tasks.length > 0) {
            const targetTask = tasks[0];
            console.log(`\n尝试手动触发任务运行: [${targetTask.id}] ${targetTask.name}...`);
            await baihu.executeTask(targetTask.id);
            console.log("执行指令发送成功。");
        }

        // 3. 获取最近的执行结果列表
        const results = await baihu.getLastResults();
        console.log(`\n最近共有 ${results.length} 条任务执行记录。`);

    } catch (e) {
        console.error(`任务操作失败: ${e.message}`);
        console.log("提示: 请确保在面板任务设置中正确注入了 OpenAPI Token。");
    }
}

main();
```

:::
