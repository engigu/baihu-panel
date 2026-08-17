const http = require('http');
const https = require('https');
const { URL } = require('url');

/**
 * 内部 API 请求辅助函数
 */
function request(urlStr, method = 'GET', data = null) {
    const token = process.env.BHPKG_OPENAPI_TOKEN || process.env.OPENAPI_TOKEN || process.env.BHPKG_NOTIFY_TOKEN;
    if (!token) {
        throw new Error(`没有正确配置或缺少 BHPKG_OPENAPI_TOKEN 环境变量以使用 env 函数。请在白虎面板的任务设置中配置这些 Key。`);
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

function getEnvsUrl() {
    const url = process.env.BHPKG_OPENAPI_URL || process.env.OPENAPI_URL;
    if (url) return url;

    const notifyUrl = process.env.BHPKG_NOTIFY_URL || 'http://localhost:8052/api/v1/notify/send';
    const targets = ['/api/v1/notify/send/', '/api/v1/notify/send', '/api/v1/notify/', '/api/v1/notify'];
    for (const target of targets) {
        if (notifyUrl.includes(target)) {
            return notifyUrl.replace(target, '/open2api/v1/env');
        }
    }
    return 'http://localhost:8052/open2api/v1/env';
}

/**
 * 获取所有的环境变量列表
 */
async function getEnvs() {
    const url = `${getEnvsUrl()}/all`;
    const res = await request(url, 'GET');
    return res.data || [];
}

/**
 * 根据变量名获取环境变量，不存在则返回 null
 */
async function getEnv(name) {
    const envs = await getEnvs();
    for (const env of envs) {
        if (env.name === name) {
            return env;
        }
    }
    return null;
}

/**
 * 批量添加环境变量
 */
async function addEnvs(envsList) {
    const url = getEnvsUrl();
    const addedEnvs = [];
    for (const env of envsList) {
        if (!env.name || !env.value) {
            throw new Error("环境变量必须包含 'name' 和 'value'");
        }
        const res = await request(url, 'POST', env);
        if (res.data) {
            addedEnvs.push(res.data);
        }
    }
    return addedEnvs;
}

/**
 * 添加单个环境变量
 */
async function addEnv(name, value, remark = "", type = "normal", hidden = true, enabled = true) {
    const url = getEnvsUrl();
    const payload = {
        name,
        value,
        remark,
        type,
        hidden,
        enabled
    };
    const res = await request(url, 'POST', payload);
    return res.data;
}

/**
 * 根据 ID 更新环境变量
 */
async function updateEnv(id, name, value, remark = null, type = null, hidden = null, enabled = null) {
    const url = `${getEnvsUrl()}/${id}`;
    const payload = {};
    if (name !== null) payload.name = name;
    if (value !== null) payload.value = value;
    if (remark !== null) payload.remark = remark;
    if (type !== null) payload.type = type;
    if (hidden !== null) payload.hidden = hidden;
    if (enabled !== null) payload.enabled = enabled;

    const res = await request(url, 'PUT', payload);
    return res.data;
}

/**
 * 批量删除环境变量
 */
async function deleteEnvs(ids) {
    for (const id of ids) {
        await deleteEnv(id);
    }
    return true;
}

/**
 * 根据 ID 删除指定环境变量
 */
async function deleteEnv(id) {
    const url = `${getEnvsUrl()}/${id}`;
    await request(url, 'DELETE');
    return true;
}

module.exports = {
    getEnvs,
    getEnv,
    addEnvs,
    addEnv,
    updateEnv,
    deleteEnvs,
    deleteEnv
};
