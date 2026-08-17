/**
 * Baihu Panel System Event SharedWorker
 * 
 * 这个 Worker 维护全浏览器唯一的一个 WebSocket 连接，
 * 并通过 postMessage 将收到的事件同步给所有打开的标签页。
 */

let ws = null;
const ports = new Set();
let reconnectTimer = null;
let lockReconnect = false;

// WebSocket 配置 (由第一个连接的页面通过消息传过来，或者使用默认值)
let wsUrl = '';

/**
 * 建立 WebSocket 连接
 */
function connect() {
    if (!wsUrl || ws) return;

    try {
        ws = new WebSocket(wsUrl);

        ws.onopen = () => {
            console.log('[SharedWorker] WebSocket 已连接');
            lockReconnect = false;
        };

        ws.onmessage = (event) => {
            // 广播消息给所有连接的端口
            broadcast(event.data);
        };

        ws.onclose = () => {
            console.log('[SharedWorker] WebSocket 已断开');
            ws = null;
            reconnect();
        };

        ws.onerror = (err) => {
            console.error('[SharedWorker] WebSocket 错误:', err);
            ws = null;
            reconnect();
        };
    } catch (e) {
        console.error('[SharedWorker] 建立连接失败:', e);
        reconnect();
    }
}

/**
 * 重连逻辑
 */
function reconnect() {
    if (lockReconnect) return;
    lockReconnect = true;

    if (reconnectTimer) clearTimeout(reconnectTimer);
    reconnectTimer = setTimeout(() => {
        connect();
        lockReconnect = false;
    }, 5000); // 5秒后尝试重连
}

/**
 * 广播消息给所有活跃标签页
 */
function broadcast(data) {
    const msg = typeof data === 'string' ? JSON.parse(data) : data;
    ports.forEach(port => {
        try {
            port.postMessage(msg);
        } catch (e) {
            // 如果端口失效，移除它
            ports.delete(port);
        }
    });
}

/**
 * 处理 SharedWorker 连接
 */
self.onconnect = (e) => {
    const port = e.ports[0];
    ports.add(port);

    port.onmessage = (event) => {
        const { type, data } = event.data;

        switch (type) {
            case 'init':
                // 初始化配置
                if (data.url) {
                    wsUrl = data.url;
                    if (!ws) connect();
                }
                break;
            case 'ping':
                port.postMessage({ type: 'pong' });
                break;
        }
    };

    port.start();

    // 发送一条初始消息确认连接成功
    port.postMessage({ type: 'worker_ready' });
};
