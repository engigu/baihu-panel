package eventbus

import "sync"

// Event 统一定义的事件体内包含的数据
type Event struct {
	Type    string
	Payload interface{}
}

// Handler 事件具体的执行句柄
type Handler func(event Event)

// Subscriber 事件订阅者接口，各业务 Service 若关注系统总线可实现此接口
type Subscriber interface {
	SubscribeEvents(bus *EventBus)
}

type EventBus struct {
	handlers map[string][]Handler
	mu       sync.RWMutex
}

func New() *EventBus {
	return &EventBus{
		handlers: make(map[string][]Handler),
	}
}

// Subscribe 注册订阅事件
func (bus *EventBus) Subscribe(eventType string, handler Handler) {
	bus.mu.Lock()
	defer bus.mu.Unlock()
	bus.handlers[eventType] = append(bus.handlers[eventType], handler)
}

// Publish 异步抛出事件
func (bus *EventBus) Publish(event Event) {
	bus.mu.RLock()
	handlers := bus.handlers[event.Type]
	bus.mu.RUnlock()

	for _, handler := range handlers {
		// 采用 Goroutine 异步不阻塞核心主线
		go handler(event)
	}
}

// 全局唯一的事件总线实例
var DefaultBus = New()
