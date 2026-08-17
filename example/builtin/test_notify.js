const baihu = require('baihu');

/**
 * 内建通知测试示例 (Node.js)
 * 
 * 使用说明：
 * 1. 确保已在该 Node.js 环境下安装过内建包。
 * 2. 系统会自动注入 BHPKG_NOTIFY_TOKEN 和 BHPKG_NOTIFY_CHANNEL。
 */

console.log("正在尝试发送 Node.js 内建通知...");

try {
    // 简单的一行代码即可完成推送
    baihu.notify(
        "Node.js 任务提醒", 
        "这是一条来自 Node.js 示例脚本的通知消息。无需配置 API 地址或 Token。"
    );
    
    console.log("发送请求已提交。");
    console.log("提示：内建包采用异步非阻塞发送，不会干扰主逻辑执行。");
    
} catch (e) {
    console.error(`通知失败: ${e.message}`);
    console.error("请检查环境变量是否注入，或是否已运行过 'baihu builtininstall'。");
}
