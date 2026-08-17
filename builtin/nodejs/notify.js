const http = require('http');
const https = require('https');
const { URL } = require('url');

/**
 * 发送通知的辅助函数 (仅使用 Node.js 标准库)
 *
 * @param {string} title - 通知标题
 * @param {string} content - 通知正文
 * @param {Object|string} [options] - 选项对象 { format, channel_id } 或格式/渠道简写
 *
 * @example
 * // 基本用法
 * baihu.notify('标题', '通知正文内容')
 *
 * // 指定 Markdown 格式
 * baihu.notify('标题', '**加粗**', { format: 'markdown' })
 *
 * // 指定渠道
 * baihu.notify('标题', '内容', { channel_id: 'ch-xxx' })
 *
 * // 简写格式
 * baihu.notify('标题', '**加粗**', 'markdown')
 *
 * // 兼容旧版调用: notify(title, text, 'ch-xxx')
 * baihu.notify('标题', '内容', 'ch-xxx')
 */
function notify(title, content, options) {
    let format = '';
    let channel_id = '';

    // 1. 如果 options 是字符串: 支持简写格式 (如 'markdown') 或兼容旧版渠道ID (如 'ch-xxx')
    if (typeof options === 'string') {
        if (['text', 'markdown', 'html'].indexOf(options) !== -1) {
            format = options;
        } else {
            channel_id = options;
        }
    } 
    // 2. 如果 options 是对象: 提取 format 与 channel_id
    else if (typeof options === 'object' && options !== null) {
        format = options.format || '';
        channel_id = options.channel_id || options.channelId || '';
    }

    const token = process.env.BHPKG_NOTIFY_TOKEN;
    const defaultChannel = process.env.BHPKG_NOTIFY_CHANNEL;
    const cid = channel_id || defaultChannel;

    if (!token || !cid) return;

    const notifyUrl = process.env.BHPKG_NOTIFY_URL || 'http://localhost:8052/api/v1/notify/send';
    if (!notifyUrl) return;

    const parsedUrl = new URL(notifyUrl);
    const protocol = parsedUrl.protocol === 'https:' ? https : http;
    
    const data = JSON.stringify({
        channel_id: cid,
        title: title || '系统通知',
        content: content,
        format: format
    });

    const optionsHttp = {
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

    const req = protocol.request(optionsHttp);
    req.on('error', (e) => {});
    req.write(data);
    req.end();
}

module.exports = { notify };
