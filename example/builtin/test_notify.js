const baihu = require('baihu');

/**
 * 内建通知测试示例 (Node.js)
 * 
 * 使用说明：
 * 1. 确保已在该 Node.js 环境下安装过内建包。
 * 2. 系统会自动注入 BHPKG_NOTIFY_TOKEN 和 BHPKG_NOTIFY_CHANNEL。
 */

async function main() {
    console.log("正在尝试发送 Node.js 内建通知...");

    // 支持通过 await 获取发送回执；未加 await 亦可兼容执行，失败时会自动向控制台输出错误日志
    const res = await baihu.notify(
        "Node.js 任务提醒", 
        "这是一条来自 Node.js 示例脚本的通知消息。无需配置 API 地址或 Token。"
    );

    if (res && res.success) {
        console.log("通知发送成功。");
    } else {
        console.error(`通知发送失败: ${res ? res.error : '未知错误'}`);
        console.error("请检查环境变量是否注入，或是否已运行过 'baihu builtininstall'。");
    }
}

main();
