/**
 * 系统事件总线驱动 (带 SharedWorker 降级逻辑)
 */

export interface WSMessage {
    type: string;
    timestamp: number;
    payload: any;
}

export type MessageHandler = (msg: WSMessage) => void;

class EventBusDriver {
    private wsUrl: string;
    private handlers: Set<MessageHandler> = new Set();
    private worker: SharedWorker | null = null;
    private socket: WebSocket | null = null;
    private reconnectTimer: any = null;

    constructor() {
        // 获取基础路径配置
        const baseUrl = (window as any).__BASE_URL__ || '';
        const apiVersion = (window as any).__API_VERSION__ || '/api/v1';
        
        // 自动计算 WebSocket 地址
        let host = window.location.host;
        let protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';

        // 如果 BASE_URL 是绝对路径 (如 http://api.example.com)
        if (baseUrl.startsWith('http')) {
            const url = new URL(baseUrl);
            host = url.host;
            protocol = url.protocol === 'https:' ? 'wss:' : 'ws:';
        }

        // 拼接最终的 WS 地址
        const path = baseUrl.startsWith('http') ? '' : baseUrl;
        this.wsUrl = `${protocol}//${host}${path}${apiVersion}/ws/events`;
        
        // 记录基础路径用于 worker 加载
        this.workerPath = `${path}/workers/event-worker.js`.replace(/\/+/g, '/');
    }

    private workerPath: string;

    private initialized = false;

    /**
     * 初始化连接
     */
    init() {
        if (this.initialized) return;
        
        if (window.SharedWorker) {
            this.initSharedWorker();
        } else {
            this.initStandardWebSocket();
        }
        this.initialized = true;
    }

    private initSharedWorker() {
        try {
            this.worker = new SharedWorker(this.workerPath);
            this.worker.port.postMessage({
                type: 'init',
                data: { url: this.wsUrl }
            });

            this.worker.port.onmessage = (e) => {
                this.emit(e.data);
            };

            this.worker.port.start();
            console.log('[EventBus] SharedWorker initialized');
        } catch (e) {
            console.error('[EventBus] SharedWorker failed, falling back', e);
            this.initStandardWebSocket();
        }
    }

    private initStandardWebSocket() {
        if (this.socket) return;

        this.socket = new WebSocket(this.wsUrl);

        this.socket.onmessage = (e) => {
            try {
                const data = JSON.parse(e.data);
                this.emit(data);
            } catch (err) {
                console.error('[EventBus] Parse message error', err);
            }
        };

        this.socket.onclose = () => {
            this.socket = null;
            this.reconnect();
        };

        this.socket.onerror = () => {
            this.socket = null;
            this.reconnect();
        };
        
        console.log('[EventBus] Standard WebSocket initialized (Fallback)');
    }

    private reconnect() {
        if (this.reconnectTimer) clearTimeout(this.reconnectTimer);
        this.reconnectTimer = setTimeout(() => this.init(), 5000);
    }

    private emit(msg: WSMessage) {
        if (!msg || !msg.type) return;
        this.handlers.forEach(handler => handler(msg));
    }

    /**
     * 订阅消息
     */
    subscribe(handler: MessageHandler) {
        this.handlers.add(handler);
        return () => this.handlers.delete(handler);
    }
}

// 导出单例
export const eventBus = new EventBusDriver();
