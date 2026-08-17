import { onMounted, onUnmounted } from 'vue';
import { eventBus, type WSMessage } from '../utils/event-bus';

/**
 * Vue Composable: 使用系统事件总线
 * 
 * @param eventType 可选，指定监听的消息类型
 * @param handler 消息处理回调
 */
export function useEventBus(eventType: string | string[] | null, handler: (payload: any, type: string) => void) {
    let unsubscribe: (() => void) | null = null;

    const handleMessage = (msg: WSMessage) => {
        if (!eventType) {
            // 监听所有消息
            handler(msg.payload, msg.type);
            return;
        }

        const types = Array.isArray(eventType) ? eventType : [eventType];
        if (types.includes(msg.type)) {
            handler(msg.payload, msg.type);
        }
    };

    onMounted(() => {
        // 确保驱动已初始化 (单例内部会处理多次调用)
        eventBus.init();
        unsubscribe = eventBus.subscribe(handleMessage);
    });

    onUnmounted(() => {
        if (unsubscribe) unsubscribe();
    });
}
