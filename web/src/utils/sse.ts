// SSE 日志流消费：应用更新/回滚等长操作共用（POST + text/event-stream）。
// 服务端事件约定：message（日志行）/ done（操作成功）/ conflict（409 安全阀）/ error（操作失败）

export type SseEventType = 'message' | 'done' | 'conflict' | 'error'

export interface SseHandlers {
  onDone?: () => void | Promise<void>
  onConflict?: (obj: any) => void
  onError?: (msg: string) => void
  onLog?: (line: string) => void
}

export async function consumeSseStream(
  path: string,
  body: Record<string, unknown>,
  handlers: SseHandlers,
  signal?: AbortSignal
): Promise<void> {
  const response = await fetch(path, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', 'Accept': 'text/event-stream' },
    body: JSON.stringify(body),
    signal
  })

  if (!response.ok) {
    const errText = await response.text()
    throw new Error(errText || `HTTP ${response.status}`)
  }

  const reader = response.body?.getReader()
  if (!reader) throw new Error('浏览器不支持 ReadableStream')

  const decoder = new TextDecoder()
  let buffer = ''
  // currentEvent 声明在分片循环外以跨 chunk 生效；按 SSE 规范在每次事件分发后重置为 message
  let currentEvent: SseEventType = 'message'
  while (true) {
    const { done, value } = await reader.read()
    if (done) break

    buffer += decoder.decode(value, { stream: true })
    const lines = buffer.split('\n')
    buffer = lines.pop() || ''

    for (const line of lines) {
      const trimmed = line.trim()
      if (!trimmed) continue
      if (trimmed.startsWith('event:')) {
        currentEvent = trimmed.substring(6).trim() as SseEventType
      } else if (trimmed.startsWith('data:')) {
        const dataStr = line.substring(line.indexOf('data:') + 5).trim()
        if (currentEvent === 'done') {
          await handlers.onDone?.()
        } else if (currentEvent === 'conflict') {
          try {
            handlers.onConflict?.(JSON.parse(dataStr))
          } catch {
            handlers.onError?.(dataStr)
          }
        } else if (currentEvent === 'error') {
          try {
            const errObj = JSON.parse(dataStr)
            handlers.onError?.(errObj.error || '操作失败')
          } catch {
            handlers.onError?.(dataStr)
          }
        } else {
          handlers.onLog?.(dataStr)
        }
        // SSE 规范：事件分发后事件类型重置，后续无 event: 前缀的 data 行一律按 message 处理
        currentEvent = 'message'
      }
    }
  }
}
