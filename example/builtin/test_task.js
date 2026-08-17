const baihu = require('baihu');

/**
 * 内建工具任务管理测试示例 (Node.js)
 * 
 * 使用说明：
 * 1. 确保已在该 Node.js 环境下安装过内建包。
 * 2. 运行时系统需要注入有效的 OpenAPI 凭证 (BHPKG_OPENAPI_TOKEN 或 OPENAPI_TOKEN)。
 */

async function main() {
    console.log("====== 开始运行 Node.js 任务管理与执行控制示例 ======");

    try {
        // 获取所有任务列表
        const tasks = await baihu.getTasks();
        console.log(`成功获取到 ${tasks.length} 个定时任务:`);
        tasks.slice(0, 5).forEach(task => { // 仅展示前5项
            console.log(`  - [${task.id}] ${task.name} (表达式: ${task.schedule || ''}, 备注: ${task.remark || ''})`);
        });

        // 尝试触发第一个任务的运行
        if (tasks.length > 0) {
            const targetTask = tasks[0];
            console.log(`\n尝试手动触发任务运行: [${targetTask.id}] ${targetTask.name}...`);
            await baihu.executeTask(targetTask.id);
            console.log("执行指令发送成功。");
        }

        // 获取最近的执行结果列表
        const results = await baihu.getLastResults();
        console.log(`\n最近共有 ${results.length} 条任务执行记录。`);

    } catch (e) {
        console.error(`任务操作失败: ${e.message}`);
        console.log("提示: 请确保在面板任务设置中正确注入了 OpenAPI Token。");
    }
}

main();
