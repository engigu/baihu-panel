const http = require('http');
const https = require('https');
const { URL } = require('url');

/** 格式字符串校验白名单 */
const VALID_FORMATS = new Set(['text', 'markdown', 'html']);

/**
 * 发送内建通知 (仅使用 Node.js 标准库)
 *
 * @param {string} title - 通知标题
 * @param {string} content - 通知正文
 * @param {Object|string} [options] - 选项对象或格式/渠道简写
 *
 * =========================================================================
 * 兼容的 5 种传参格式与提交至后端的 Payload 转换映射表:
 * -------------------------------------------------------------------------
 * 1. 2参极简调用: notify('标题', '正文')
 *    => { channel_id: '默认', title: '标题', content: '正文', options: {} }
 *
 * 2. 3参格式简写: notify('标题', '正文', 'markdown')
 *    => { channel_id: '默认', title: '标题', content: '正文', options: { format: 'markdown' } }
 *
 * 3. 3参渠道简写: notify('标题', '正文', 'ch-123')
 *    => { channel_id: 'ch-123', title: '标题', content: '正文', options: {} }
 *
 * 4. 3参标准对象: notify('标题', '正文', { channel_id: 'ch-123', format: 'html', group: '打卡' })
 *    => { channel_id: 'ch-123', title: '标题', content: '正文', options: { format: 'html', group: '打卡' } }
 *
 * 5. 3参扩展配置: notify('标题', '正文', { channel_id: 'ch-123', options: { format: 'markdown', group: '打卡' } })
 *    => { channel_id: 'ch-123', title: '标题', content: '正文', options: { format: 'markdown', group: '打卡' } }
 * =========================================================================
 */
function notify(title, content, options) {
    const optionsObj = parseAndBuildOptions(options);

    // 提取 channel_id 至顶层并从 optionsObj 移除
    let channelId = optionsObj.channel_id;
    delete optionsObj.channel_id;

    // 补全默认渠道 (若未显式指定)
    if (!channelId && process.env.BHPKG_NOTIFY_CHANNEL) {
        channelId = process.env.BHPKG_NOTIFY_CHANNEL;
    }

    const token = process.env.BHPKG_NOTIFY_TOKEN;
    if (!token || !channelId) return;

    const notifyUrl = process.env.BHPKG_NOTIFY_URL || 'http://localhost:8052/api/v1/notify/send';
    const parsedUrl = new URL(notifyUrl);
    const protocol = parsedUrl.protocol === 'https:' ? https : http;
    
    const data = JSON.stringify({
        channel_id: channelId,
        title: title || '系统通知',
        content: content,
        options: optionsObj
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

/** 场景 A: 格式字符串构建 */
function buildFromFormatString(format) {
    return { format };
}

/** 场景 B: 渠道 ID 字符串构建 */
function buildFromChannelString(channelId) {
    return { channel_id: channelId };
}

/** 场景 C: 配置对象构建 (支持平铺与嵌套 options 展平) */
function buildFromOptionsObject(options) {
    if (!options || typeof options !== 'object') return {};

    const result = {};
    for (const [key, value] of Object.entries(options)) {
        if (key === 'channelId') {
            result.channel_id = value;
        } else if (key === 'options' && typeof value === 'object' && value !== null) {
            Object.assign(result, value);
        } else {
            result[key] = value;
        }
    }

    return result;
}

/** 场景策略分发路由器 */
function parseAndBuildOptions(options) {
    if (options === undefined || options === null) return {};

    if (typeof options === 'string') {
        return VALID_FORMATS.has(options)
            ? buildFromFormatString(options)
            : buildFromChannelString(options);
    }

    if (typeof options === 'object') {
        return buildFromOptionsObject(options);
    }

    return {};
}

module.exports = { notify };
