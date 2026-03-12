<script setup lang="ts">
import { ref, onMounted, onUnmounted, watch } from 'vue'
import { Terminal } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import '@xterm/xterm/css/xterm.css'

const lightTheme = {
  background: '#f4f4f5', // zinc-100
  foreground: '#18181b', // zinc-900
  cursor: '#18181b',
  selectionBackground: 'rgba(0, 0, 0, 0.15)',
}

const darkTheme = {
  background: '#09090b', // zinc-950
  foreground: '#e4e4e7', // zinc-200
  cursor: '#e4e4e7',
  selectionBackground: 'rgba(255, 255, 255, 0.2)',
}

const props = withDefaults(
  defineProps<{
    content: string
    fontSize?: number
    theme?: 'dark' | 'light'
    autoScroll?: boolean
  }>(),
  {
    fontSize: 12,
    theme: 'dark',
    autoScroll: true
  }
)

const terminalRef = ref<HTMLDivElement | null>(null)
let terminal: Terminal | null = null
let fitAddon: FitAddon | null = null
const lightBackgroundClass = 'terminal-theme-light'
const darkBackgroundClass = 'terminal-theme-dark'

function getTheme() {
  return props.theme === 'dark' ? darkTheme : lightTheme
}

function initTerminal() {
  if (!terminalRef.value) return
  
  terminal = new Terminal({
    fontSize: props.fontSize,
    fontFamily: "'JetBrains Mono', 'Fira Code', monospace",
    theme: getTheme(),
    allowProposedApi: true,
    convertEol: true,
    disableStdin: true,
    cursorBlink: false,
    rows: 10,
  })

  fitAddon = new FitAddon()
  terminal.loadAddon(fitAddon)
  terminal.open(terminalRef.value)
  
  if (props.content) {
    terminal.write(props.content)
  }

  setTimeout(() => {
    fitAddon?.fit()
  }, 50)
}

watch(() => props.content, (newContent, oldContent) => {
  if (!terminal) return
  
  if (newContent.length < oldContent.length) {
    terminal.clear()
    terminal.write(newContent)
  } else {
    const appended = newContent.slice(oldContent.length)
    terminal.write(appended)
  }

  if (props.autoScroll) {
    terminal.scrollToBottom()
  }
})

watch(() => props.theme, () => {
  if (terminal) {
    terminal.options.theme = getTheme()
  }
})

function handleResize() {
  fitAddon?.fit()
}

onMounted(() => {
  window.addEventListener('resize', handleResize)
  setTimeout(initTerminal, 100)
})

onUnmounted(() => {
  window.removeEventListener('resize', handleResize)
  if (terminal) {
    terminal.dispose()
  }
})

defineExpose({
  fit: () => fitAddon?.fit(),
  clear: () => terminal?.clear()
})
</script>

<template>
  <div
    ref="terminalRef"
    class="w-full h-full min-h-0"
    :class="theme === 'dark' ? darkBackgroundClass : lightBackgroundClass"
  />
</template>

<style scoped>
.terminal-theme-light {
  background-color: #f4f4f5;
}

.terminal-theme-dark {
  background-color: #09090b;
}

:deep(.xterm) {
  padding: 8px;
}
:deep(.xterm-viewport),
:deep(.xterm-screen) {
  background-color: inherit !important;
}
</style>
