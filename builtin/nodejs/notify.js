const http = require('http');
const https = require('https');
const { URL } = require('url');

/**
 * 发送通知的辅助函数 (仅使用 Node.js 标准库)
 *
 * @param {string} title - 通知标题
 * @param {string} content - 通知正文
 * @param {string} [format] - 内容格式："text"(默认), "markdown", "html"
 * @param {string} [channelId] - 渠道ID，不传则使用环境变量 BHPKG_NOTIFY_CHANNEL
 *
 * @example
 * // 最简用法
 * baihu.notify('标题', '消息')
 *
 * // 指定格式
 * baihu.notify('标题', '**粗体**', 'markdown')
 *
 * // 指定渠道（跳过格式）
 * baihu.notify('标题', '消息', undefined, 'ch-xxx')
 *
 * // 同时指定
 * baihu.notify('标题', '<b>粗体</b>', 'html', 'ch-xxx')
 *
 * // 兼容旧版调用: notify(title, text, channelId)
 * baihu.notify('标题', '消息', 'ch-xxx')
 */
function notify(title, content, format, channelId) {
    // 兼容旧版调用: notify(title, text, channelId)
    // 如果只传了3个参数且第3个参数不是合法的format值，则将其视为channelId
    // 后期移除
    var VALID_FORMATS = ['text', 'markdown', 'html', ''];
    if (arguments.length === 3 && VALID_FORMATS.indexOf(format) === -1) {
        channelId = format;
        format = '';
    }

    format = format || '';
    channelId = channelId || '';
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
        content: content,
        format: format || ''
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
