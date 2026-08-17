const http = require('http');
const https = require('https');
const { URL } = require('url');

/**
 * 发送通知的辅助函数 (仅使用 Node.js 标准库)
 */
function notify(title, text, channelId) {
    const token = process.env.BHPKG_NOTIFY_TOKEN;
    const channel = process.env.BHPKG_NOTIFY_CHANNEL;

    if (!token || !channel) {
        const missing = [];
        if (!token) missing.push("BHPKG_NOTIFY_TOKEN");
        if (!channel) missing.push("BHPKG_NOTIFY_CHANNEL");
        throw new Error(`没有正确配置或缺少 ${missing.join(" 和 ")} 环境变量以使用 notify 函数。请在白虎面板的任务设置中配置这些 Key。`);
    }

    const notifyUrl = process.env.BHPKG_NOTIFY_URL || 'http://localhost:8052/api/v1/notify/send';
    const cid = channelId || channel;

    if (!notifyUrl || !token || !cid) return;

    const parsedUrl = new URL(notifyUrl);
    const protocol = parsedUrl.protocol === 'https:' ? https : http;
    
    const data = JSON.stringify({
        channel_id: cid,
        title: title || '系统通知',
        text: text
    });

    const options = {
        hostname: parsedUrl.hostname,
        port: parsedUrl.port,
        path: parsedUrl.pathname + (parsedUrl.search || ''),
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
            'notify-token': token,
            'Content-Length': Buffer.byteLength(data)
        }
    };

    const req = protocol.request(options);
    req.on('error', (e) => {});
    req.write(data);
    req.end();
}

module.exports = { notify };
