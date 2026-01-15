<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
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
}>()

const terminalRef = ref<HTMLDivElement | null>(null)
let terminal: Terminal | null = null
let fitAddon: FitAddon | null = null
let ws: WebSocket | null = null
let isPtyMode = false
let inputBuffer = ''
let commandHistory: string[] = []
let historyIndex = -1

function initTerminal(forceConnect = false) {
  if (!terminalRef.value) return

  // 清理旧终端
  if (terminal) {
    terminal.dispose()
    terminal = null
  }
  
  // 确保旧的 WebSocket 完全关闭
  if (ws) {
    if (ws.readyState === WebSocket.OPEN || ws.readyState === WebSocket.CONNECTING) {
      ws.close()
    }
    ws = null
  }
  
  inputBuffer = ''
  isPtyMode = false

  terminal = new Terminal({
    cursorBlink: true,
    fontSize: props.fontSize,
    fontFamily: 'Consolas, Monaco, monospace',
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
  
  terminal.focus()

  // autoConnect 或者强制连接时才连接
  if (props.autoConnect || forceConnect) {
    // 延迟连接，确保终端完全初始化和旧连接完全关闭
    setTimeout(() => {
      connectWebSocket()
    }, 200)
  }

  // 清除当前输入行（Windows 模式用）
  function clearLine() {
    for (let i = 0; i < inputBuffer.length; i++) {
      terminal?.write('\b \b')
    }
  }

  // 处理用户输入
  terminal.onData((data) => {
    if (!ws || ws.readyState !== WebSocket.OPEN) return

    if (isPtyMode) {
      ws.send(data)
      return
    }

    if (data === '\r') {
      terminal?.write('\r\n')
      if (inputBuffer.trim()) {
        commandHistory.push(inputBuffer)
        historyIndex = commandHistory.length
        ws.send(inputBuffer + '\r\n')
      }
      inputBuffer = ''
    } else if (data === '\x1b[A') {
      if (commandHistory.length > 0 && historyIndex > 0) {
        clearLine()
        historyIndex--
        inputBuffer = commandHistory[historyIndex] ?? ''
        terminal?.write(inputBuffer)
      }
    } else if (data === '\x1b[B') {
      clearLine()
      if (historyIndex < commandHistory.length - 1) {
        historyIndex++
        inputBuffer = commandHistory[historyIndex] ?? ''
        terminal?.write(inputBuffer)
      } else {
        historyIndex = commandHistory.length
        inputBuffer = ''
      }
    } else if (data === '\x7f' || data === '\b') {
      if (inputBuffer.length > 0) {
        inputBuffer = inputBuffer.slice(0, -1)
        terminal?.write('\b \b')
      }
    } else if (data === '\x03') {
      ws.send('\x03')
      inputBuffer = ''
      historyIndex = commandHistory.length
      terminal?.write('^C\r\n')
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
    terminal?.writeln('\x1b[31m无法创建 WebSocket 连接\x1b[0m')
    return
  }

  ws.onopen = () => {
    terminal?.writeln('\x1b[32m已连接到终端\x1b[0m')
    emit('connected')

    // 如果有初始命令，延迟发送（PTY 会自动回显命令）
    if (props.initialCommand) {
      setTimeout(() => {
        if (ws && ws.readyState === WebSocket.OPEN) {
          ws.send(props.initialCommand + '\r')
        }
      }, 100)
    }
    terminal?.focus()
  }

  ws.onmessage = (event) => {
    if (event.data === '__PTY_MODE__') {
      isPtyMode = true
      return
    }
    if (event.data === '__PIPE_MODE__') {
      isPtyMode = false
      return
    }
    terminal?.write(event.data)
  }

  ws.onclose = () => {
    terminal?.writeln('')
    terminal?.writeln('\x1b[31m连接已断开\x1b[0m')
    emit('disconnected')
  }

  ws.onerror = () => {
    terminal?.writeln('\x1b[31m连接错误\x1b[0m')
  }
}

function reconnect() {
  if (ws) {
    ws.close()
  }
  inputBuffer = ''
  isPtyMode = false
  terminal?.clear()
  connectWebSocket()
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
