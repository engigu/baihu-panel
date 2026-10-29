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

    // 兼容旧版 text 入参
    if ('text' in optionsObj && (content === undefined || content === null || content === '')) {
        content = optionsObj.text;
        delete optionsObj.text;
    }

    // 补全默认渠道 (若未显式指定)
    if (!channelId && process.env.BHPKG_NOTIFY_CHANNEL) {
        channelId = process.env.BHPKG_NOTIFY_CHANNEL;
    }

    const token = process.env.BHPKG_NOTIFY_TOKEN;
    if (!token || !channelId) {
        const missing = [];
        if (!token) missing.push('BHPKG_NOTIFY_TOKEN');
        if (!channelId) missing.push('BHPKG_NOTIFY_CHANNEL');
        const err = `缺少必要的配置或环境变量以发送通知: ${missing.join(', ')}。请在白虎面板的任务设置中配置指定的 Key 或显式指定渠道。`;
        console.error(`[baihu.notify] ${err}`);
        return Promise.resolve({ success: false, error: err });
    }

    const notifyUrl = process.env.BHPKG_NOTIFY_URL || 'http://localhost:8052/api/v1/notify/send';
    let parsedUrl;
    try {
        parsedUrl = new URL(notifyUrl);
    } catch (e) {
        const err = `无效的通知接口地址: ${notifyUrl} (${e.message})`;
        console.error(`[baihu.notify] ${err}`);
        return Promise.resolve({ success: false, error: err });
    }

    const protocol = parsedUrl.protocol === 'https:' ? https : http;
    
    const data = JSON.stringify({
        channel_id: channelId,
        title: title || '系统通知',
        content: content !== undefined && content !== null ? String(content) : '',
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

    return new Promise((resolve) => {
        const req = protocol.request(optionsHttp, (res) => {
            let body = '';
            res.setEncoding('utf8');
            res.on('data', (chunk) => body += chunk);
            res.on('end', () => {
                let parsed = null;
                try {
                    parsed = body ? JSON.parse(body) : null;
                } catch (_) {}

                if (res.statusCode >= 200 && res.statusCode < 300) {
                    if (parsed && typeof parsed === 'object' && parsed.code !== undefined && parsed.code !== 200) {
                        const errMsg = parsed.msg || parsed.message || '未知业务错误';
                        console.error(`[baihu.notify] 面板业务错误 [${parsed.code}]: ${errMsg}`);
                        resolve({ success: false, status: res.statusCode, code: parsed.code, error: errMsg, data: parsed });
                    } else if (parsed && typeof parsed === 'object' && parsed.data && parsed.data.success === false) {
                        const errMsg = parsed.data.error || '渠道投递失败';
                        console.error(`[baihu.notify] 渠道发送失败: ${errMsg}`);
                        resolve({ success: false, status: res.statusCode, code: parsed.code, error: errMsg, data: parsed });
                    } else {
                        resolve({ success: true, status: res.statusCode, data: parsed || body });
                    }
                } else {
                    const errMsg = (parsed && (parsed.msg || parsed.message)) || body || `HTTP ${res.statusCode}`;
                    console.error(`[baihu.notify] 面板返回错误 [${res.statusCode}]: ${errMsg}`);
                    resolve({ success: false, status: res.statusCode, error: errMsg, data: parsed });
                }
            });
        });

        req.on('error', (e) => {
            const errMsg = `发送失败: ${e.code || e.message} (url=${notifyUrl})`;
            console.error(`[baihu.notify] ${errMsg}`);
            resolve({ success: false, error: errMsg, code: e.code });
        });

        req.write(data);
        req.end();
    });
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
