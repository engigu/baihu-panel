const http = require('http');
const https = require('https');
const { URL } = require('url');

/**
 * 内部 API 请求辅助函数
 */
function request(urlStr, method = 'GET', data = null) {
    const token = process.env.BHPKG_OPENAPI_TOKEN || process.env.OPENAPI_TOKEN || process.env.BHPKG_NOTIFY_TOKEN;
    if (!token) {
        throw new Error(`没有正确配置或缺少 BHPKG_OPENAPI_TOKEN 环境变量以使用 task 函数。请在白虎面板的任务设置中配置这些 Key。`);
    }

    const parsedUrl = new URL(urlStr);
    const protocol = parsedUrl.protocol === 'https:' ? https : http;
    
    let payload = '';
    const headers = {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${token}`
    };

    if (data !== null) {
        payload = JSON.stringify(data);
        headers['Content-Length'] = Buffer.byteLength(payload);
    }

    const options = {
        hostname: parsedUrl.hostname,
        port: parsedUrl.port,
        path: parsedUrl.pathname + (parsedUrl.search || ''),
        method: method,
        headers: headers
    };

    return new Promise((resolve, reject) => {
        const req = protocol.request(options, (res) => {
            let body = '';
            res.setEncoding('utf8');
            res.on('data', (chunk) => body += chunk);
            res.on('end', () => {
                if (res.statusCode >= 200 && res.statusCode < 300) {
                    try {
                        const parsed = body ? JSON.parse(body) : {};
                        if (parsed && typeof parsed === 'object' && parsed.code !== undefined && parsed.code !== 200) {
                            reject(new Error(`请求失败 [${parsed.code}]: ${parsed.msg || parsed.message || '未知错误'}`));
                        } else {
                            resolve(parsed);
                        }
                    } catch (e) {
                        resolve(body);
                    }
                } else {
                    let errMsg = body;
                    try {
                        const parsed = JSON.parse(body);
                        errMsg = parsed.msg || parsed.message || body;
                    } catch(e) {}
                    reject(new Error(`请求失败 [${res.statusCode}]: ${errMsg}`));
                }
            });
        });
        
        req.on('error', (e) => reject(e));
        if (payload) {
            req.write(payload);
        }
        req.end();
    });
}

function getBaseUrl() {
    const url = process.env.BHPKG_OPENAPI_URL || process.env.OPENAPI_URL;
    if (url) {
        if (url.endsWith('/env')) return url.slice(0, -4);
        if (url.endsWith('/env/')) return url.slice(0, -5);
        return url;
    }

    const notifyUrl = process.env.BHPKG_NOTIFY_URL || 'http://localhost:8052/api/v1/notify/send';
    const targets = ['/api/v1/notify/send/', '/api/v1/notify/send', '/api/v1/notify/', '/api/v1/notify'];
    for (const target of targets) {
        if (notifyUrl.includes(target)) {
            return notifyUrl.replace(target, '/open2api/v1');
        }
    }
    return 'http://localhost:8052/open2api/v1';
}

/**
 * 获取全部任务列表
 */
async function getTasks() {
    const url = `${getBaseUrl()}/tasks`;
    const res = await request(url, 'GET');
    return res.data || [];
}

/**
 * 根据 ID 获取单个任务信息
 */
async function getTask(id) {
    const url = `${getBaseUrl()}/tasks/${id}`;
    const res = await request(url, 'GET');
    return res.data;
}

/**
 * 根据 ID 更新指定任务
 */
async function updateTask(id, name, command, remark, pin_type, trigger_type, schedule, timeout, work_dir, retry_count, retry_interval, random_range, enabled) {
    const url = `${getBaseUrl()}/tasks/${id}`;
    const payload = {};
    if (name !== undefined) payload.name = name;
    if (command !== undefined) payload.command = command;
    if (remark !== undefined) payload.remark = remark;
    if (pin_type !== undefined) payload.pin_type = pin_type;
    if (trigger_type !== undefined) payload.trigger_type = trigger_type;
    if (schedule !== undefined) payload.schedule = schedule;
    if (timeout !== undefined) payload.timeout = timeout;
    if (work_dir !== undefined) payload.work_dir = work_dir;
    if (retry_count !== undefined) payload.retry_count = retry_count;
    if (retry_interval !== undefined) payload.retry_interval = retry_interval;
    if (random_range !== undefined) payload.random_range = random_range;
    if (enabled !== undefined) payload.enabled = enabled;

    const res = await request(url, 'PUT', payload);
    return res.data;
}

/**
 * 根据 ID 删除任务
 */
async function deleteTask(id) {
    const url = `${getBaseUrl()}/tasks/${id}`;
    await request(url, 'DELETE');
    return true;
}

/**
 * 触发运行指定任务
 */
async function executeTask(id) {
    const url = `${getBaseUrl()}/execute/task/${id}`;
    const res = await request(url, 'POST');
    return res.data;
}

/**
 * 根据日志 ID 停止正在运行的任务
 */
async function stopTask(logId) {
    const url = `${getBaseUrl()}/tasks/stop/${logId}`;
    const res = await request(url, 'POST');
    return res.data;
}

/**
 * 获取最近的任务执行结果列表
 */
async function getLastResults() {
    const url = `${getBaseUrl()}/execute/results`;
    const res = await request(url, 'GET');
    return res.data || [];
}

module.exports = {
    getTasks,
    getTask,
    updateTask,
    deleteTask,
    executeTask,
    stopTask,
    getLastResults
};
