<script setup lang="ts">
import { ref, onMounted, onUnmounted, watch } from 'vue'
import { Terminal } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import '@xterm/xterm/css/xterm.css'

const props = withDefaults(
  defineProps<{
    fontSize?: number
    autoConnect?: boolean
    initialCommand?: string
  }>(),
  {
    fontSize: 13,
    autoConnect: true,
    initialCommand: ''
  }
)

const emit = defineEmits<{
  connected: []
  disconnected: []
  success: []
  failed: []
  'status-change': [status: { text: string; type: 'success' | 'error' | 'info' } | null]
}>()

const terminalRef = ref<HTMLDivElement | null>(null)
let terminal: Terminal | null = null
let fitAddon: FitAddon | null = null
let ws: WebSocket | null = null
let isPtyMode = false
let inputBuffer = ''
let cmdHistory: string[] = []
let historyIdx = -1

const statusMessage = ref<{ text: string; type: 'success' | 'error' | 'info' } | null>(null)

watch(() => props.fontSize, (newVal) => {
  if (terminal && newVal) {
    terminal.options.fontSize = newVal
    handleResize()
  }
})

watch(statusMessage, (newVal) => {
  emit('status-change', newVal)
})

function initTerminal(forceConnect = false) {
  if (!terminalRef.value) return

  // 清理旧终端与 DOM 内容
  if (terminal) {
    terminal.dispose()
    terminal = null
  }
  terminalRef.value.innerHTML = ''

  // 确保旧的 WebSocket 完全关闭
  if (ws) {
    if (ws.readyState === WebSocket.OPEN || ws.readyState === WebSocket.CONNECTING) {
      ws.close()
    }
    ws = null
  }

  inputBuffer = ''
  isPtyMode = false
  statusMessage.value = { text: '正在初始化终端...', type: 'info' }

  terminal = new Terminal({
    cursorBlink: true,
    convertEol: true,
    fontSize: props.fontSize,
    lineHeight: 1.25,
    fontFamily: 'Consolas, Monaco, "Courier New", monospace',
    theme: {
      background: '#1e1e1e',
      foreground: '#d4d4d4',
      cursor: '#d4d4d4',
    }
  })

  fitAddon = new FitAddon()
  terminal.loadAddon(fitAddon)
  terminal.open(terminalRef.value)

  // 延迟调用 fit，确保 DOM 已渲染
  setTimeout(() => {
    try {
      fitAddon?.fit()
    } catch (e) {
      // 忽略 fit 错误
    }
  }, 50)

  // 支持 Ctrl+C 复制选中内容
  terminal.attachCustomKeyEventHandler((e) => {
    if (e.ctrlKey && e.code === 'KeyC' && e.type === 'keydown') {
      const selection = terminal?.getSelection()
      if (selection) {
        navigator.clipboard.writeText(selection)
        return false // 阻止默认的终端 Ctrl+C 行为
      }
    }
    return true
  })

  terminal.focus()

  // autoConnect 或者强制连接时才连接
  if (props.autoConnect || forceConnect) {
    // 延迟连接，确保终端完全初始化和旧连接完全关闭
    setTimeout(() => {
      connectWebSocket()
    }, 200)
  }

  // 监听按键事件（优先拦截 ArrowUp / ArrowDown 填充历史命令）
  terminal.attachCustomKeyEventHandler((e) => {
    if (e.ctrlKey && e.code === 'KeyC' && e.type === 'keydown') {
      const selection = terminal?.getSelection()
      if (selection) {
        navigator.clipboard.writeText(selection)
        return false
      }
    }

    if (e.type === 'keydown') {
      if (e.key === 'ArrowUp') {
        if (cmdHistory.length > 0) {
          if (historyIdx > 0) historyIdx--
          else historyIdx = 0

          const cmd = cmdHistory[historyIdx] || ''
          // 发送 \x15 (Ctrl+U) 清空提示符已有输入，然后补上历史命令
          if (ws && ws.readyState === WebSocket.OPEN) {
            ws.send('\x15' + cmd)
          }
        }
        return false // 阻止转义字符，避免在 ConPTY 下产生无效打字
      }

      if (e.key === 'ArrowDown') {
        if (cmdHistory.length > 0) {
          if (historyIdx < cmdHistory.length - 1) {
            historyIdx++
            const cmd = cmdHistory[historyIdx] || ''
            if (ws && ws.readyState === WebSocket.OPEN) {
              ws.send('\x15' + cmd)
            }
          } else {
            historyIdx = cmdHistory.length
            if (ws && ws.readyState === WebSocket.OPEN) {
              ws.send('\x15')
            }
          }
        }
        return false
      }
    }
    return true
  })

  // 处理用户输入
  terminal.onData((data) => {
    if (!ws || ws.readyState !== WebSocket.OPEN) return

    if (isPtyMode) {
      if (data === '\r') {
        if (inputBuffer.trim() && cmdHistory[cmdHistory.length - 1] !== inputBuffer.trim()) {
          cmdHistory.push(inputBuffer.trim())
          historyIdx = cmdHistory.length
        }
        inputBuffer = ''
      } else if (data === '\x7f' || data === '\b') {
        if (inputBuffer.length > 0) inputBuffer = inputBuffer.slice(0, -1)
      } else if (data >= ' ') {
        inputBuffer += data
      }
      ws.send(data)
      return
    }

    if (data === '\r') {
      terminal?.write('\r\n')
      if (inputBuffer.trim()) {
        ws.send(inputBuffer + '\r\n')
      }
      inputBuffer = ''
    } else if (data === '\x7f' || data === '\b') {
      if (inputBuffer.length > 0) {
        inputBuffer = inputBuffer.slice(0, -1)
        terminal?.write('\b \b')
      }
    } else if (data === '\x03') {
      ws.send('\x03')
      inputBuffer = ''
      terminal?.write('^C\r\n')
    } else if (data === '\x0c') {
      if (!inputBuffer) {
        ws.send('clear\r\n')
      } else {
        terminal?.clear()
        terminal?.write(inputBuffer)
      }
    } else if (data >= ' ' || data === '\t') {
      inputBuffer += data
      terminal?.write(data)
    }
  })
}

function connectWebSocket() {
  // 如果已有连接，先关闭
  if (ws && (ws.readyState === WebSocket.OPEN || ws.readyState === WebSocket.CONNECTING)) {
    ws.close()
    ws = null
  }

  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  const baseUrl = (window as any).__BASE_URL__ || ''
  const apiVersion = (window as any).__API_VERSION__ || '/api/v1'
  const wsUrl = `${protocol}//${window.location.host}${baseUrl}${apiVersion}/terminal/ws`

  try {
    ws = new WebSocket(wsUrl)
  } catch {
    statusMessage.value = { text: '无法创建 WebSocket 连接', type: 'error' }
    return
  }

  ws.onopen = () => {
    statusMessage.value = { text: '已连接到终端', type: 'success' }
    emit('connected')
    terminal?.focus()
    handleResize()
  }

  let initialCommandSent = false

  ws.onmessage = (event) => {
    if (event.data === '__PTY_MODE__' || event.data === '__PIPE_MODE__') {
      isPtyMode = (event.data === '__PTY_MODE__')
      handleResize()
      if (props.initialCommand && !initialCommandSent) {
        initialCommandSent = true
        const initCmd = props.initialCommand.trim()
        if (initCmd && cmdHistory[cmdHistory.length - 1] !== initCmd) {
          cmdHistory.push(initCmd)
          historyIdx = cmdHistory.length
        }
        setTimeout(() => {
          if (ws && ws.readyState === WebSocket.OPEN) {
            ws.send(props.initialCommand + '\r\n')
          }
        }, 400)
      }
      return
    }
    terminal?.write(event.data)

    // 检测结果标识（保持简单的单次消息检测，不使用持久缓冲区，避免重复触发通知）
    if (typeof event.data === 'string') {
      if (event.data.includes('__INSTALL_SUCCESS__')) {
        emit('success')
      } else if (event.data.includes('__INSTALL_FAILED__')) {
        emit('failed')
      }
    }
  }

  ws.onclose = () => {
    statusMessage.value = { text: '连接已断开', type: 'error' }
    emit('disconnected')
  }

  ws.onerror = () => {
    statusMessage.value = { text: '连接错误', type: 'error' }
  }
}

function reconnect() {
  initTerminal(true)
}

function dispose() {
  if (ws) {
    if (ws.readyState === WebSocket.OPEN || ws.readyState === WebSocket.CONNECTING) {
      ws.close()
    }
    ws = null
  }
  if (terminal) {
    terminal.dispose()
    terminal = null
  }
  inputBuffer = ''
  isPtyMode = false
}

function handleResize() {
  try {
    fitAddon?.fit()
    // 通知后端调整 PTY 尺寸
    if (isPtyMode && terminal && ws && ws.readyState === WebSocket.OPEN) {
      ws.send(JSON.stringify({
        type: 'resize',
        cols: terminal.cols,
        rows: terminal.rows
      }))
    }
  } catch (e) {
    console.warn('Terminal resize failed:', e)
  }
}

// 暴露方法给父组件
defineExpose({
  reconnect,
  dispose,
  initTerminal
})

onMounted(() => {
  window.addEventListener('resize', handleResize)
  // 延迟初始化，确保 DOM 完全渲染
  setTimeout(initTerminal, 150)
})

onUnmounted(() => {
  window.removeEventListener('resize', handleResize)
  dispose()
})
</script>

<template>
  <div ref="terminalRef" class="terminal-container w-full h-full !bg-[#1e1e1e]" />
</template>

<style scoped>
.terminal-container {
  background: #1e1e1e !important;
}

.terminal-container :deep(.xterm) {
  padding: 0;
}

.terminal-container :deep(.xterm-viewport) {
  scrollbar-width: thin;
  scrollbar-color: #4a4a4a #1e1e1e;
  background: #1e1e1e !important;
  -webkit-overflow-scrolling: touch;
  overscroll-behavior-y: contain;
}

.terminal-container :deep(.xterm-screen) {
  background: #1e1e1e !important;
}

.terminal-container :deep(.xterm-viewport::-webkit-scrollbar) {
  width: 8px;
}

.terminal-container :deep(.xterm-viewport::-webkit-scrollbar-track) {
  background: #1e1e1e;
}

.terminal-container :deep(.xterm-viewport::-webkit-scrollbar-thumb) {
  background: #4a4a4a;
  border-radius: 4px;
}

.terminal-container :deep(.xterm-viewport::-webkit-scrollbar-thumb:hover) {
  background: #5a5a5a;
}
</style>
