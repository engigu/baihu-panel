<script setup lang="ts">
import { ref, onMounted, onUnmounted, watch, nextTick } from 'vue'
import { AlertCircle, Loader2 } from 'lucide-vue-next'
import { Terminal } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import { UnicodeGraphemesAddon } from '@xterm/addon-unicode-graphemes'
import { useTheme } from '@/composables/useTheme'
import '@xterm/xterm/css/xterm.css'

interface Props {
  content?: string
  loading?: boolean
  loadingText?: string
  emptyTitle?: string
  emptyDescription?: string
}

const props = withDefaults(defineProps<Props>(), {
  content: '',
  loading: false,
  loadingText: '正在获取日志内容',
  emptyTitle: '未检测到输出内容',
  emptyDescription: '此任务执行期间未产生标准输出（Stdout）或错误输出（Stderr）日志。'
})

const terminalRef = ref<HTMLDivElement | null>(null)
let terminal: Terminal | null = null
let fitAddon: FitAddon | null = null
let resizeObserver: ResizeObserver | null = null

const { resolvedTheme } = useTheme()

let lastContentLength = 0
let lastContent = ''

function getTheme() {
  const isDark = resolvedTheme.value === 'dark'
  return {
    background: isDark ? '#00000000' : '#e4e4e7',
    foreground: isDark ? '#d4d4d4' : '#333333',
    cursor: '#00000000',
    selectionBackground: isDark ? 'rgba(255, 255, 255, 0.3)' : 'rgba(0, 0, 0, 0.2)',
  }
}

function fit() {
  if (!terminal || !fitAddon || !terminalRef.value) return
  try {
    fitAddon.fit()
  } catch (e) {}
}

function initTerminal() {
  if (!terminalRef.value || terminal) return

  terminal = new Terminal({
    cursorBlink: false,
    disableStdin: true,
    fontSize: 12,
    lineHeight: 1.25,
    fontFamily: 'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace',
    theme: getTheme(),
    allowTransparency: true,
    scrollback: 5000,
    // terminal.unicode 属 proposed API，加载 Unicode 宽度表插件前必须放开
    allowProposedApi: true,
  })

  fitAddon = new FitAddon()
  terminal.loadAddon(fitAddon)
  // xterm 内置宽度表为 Unicode 6，emoji 被判 1 列而字形约占 2.5 列，letterSpacing
  // 被压成负值导致 emoji 覆盖右侧文字；此插件提供 Unicode 15 宽度表与字素簇合并
  terminal.loadAddon(new UnicodeGraphemesAddon())
  terminal.open(terminalRef.value)

  // 禁用手机端点击弹出键盘
  const textarea = terminalRef.value.querySelector('.xterm-helper-textarea') as HTMLTextAreaElement
  if (textarea) {
    textarea.readOnly = true
  }

  // 优化手机端滑动体验
  let lastTouchY = 0
  terminalRef.value.addEventListener('touchstart', (e) => {
    if (e.touches && e.touches[0]) {
      lastTouchY = e.touches[0].clientY
    }
  }, { passive: true })

  terminalRef.value.addEventListener('touchmove', (e) => {
    if (terminal && e.touches && e.touches[0]) {
      const currentY = e.touches[0].clientY
      const deltaY = lastTouchY - currentY
      lastTouchY = currentY
      
      const viewport = terminalRef.value?.querySelector('.xterm-viewport')
      if (viewport) {
        viewport.scrollTop += deltaY * 2.5
      }
      
      if (e.cancelable) {
        e.preventDefault()
      }
    }
  }, { passive: false })

  // 支持 Ctrl+C 复制选中内容
  terminal.attachCustomKeyEventHandler((e) => {
    if (e.ctrlKey && e.code === 'KeyC' && e.type === 'keydown') {
      const selection = terminal?.getSelection()
      if (selection) {
        navigator.clipboard.writeText(selection)
        return false
      }
    }
    return true
  })

  fit()

  if (props.content) {
    writeContent(props.content)
  }

  requestAnimationFrame(() => {
    fit()
  })
}

function writeContent(content: string) {
  if (!terminal) return
  
  if (content.length > lastContentLength && content.substring(0, lastContentLength) === lastContent) {
    // 增量更新：只截取新的部分
    const diff = content.substring(lastContentLength)
    const formattedDiff = diff.replace(/\r?\n/g, '\r\n')
    terminal.write(formattedDiff)
  } else {
    // 全量更新
    const formattedContent = content.replace(/\r?\n/g, '\r\n')
    terminal.reset()
    terminal.write(formattedContent)
  }
  
  lastContentLength = content.length
  lastContent = content
}

watch(resolvedTheme, () => {
  if (terminal) {
    terminal.options.theme = getTheme()
  }
})

watch(() => props.content, (newContent) => {
  if (newContent && newContent.trim()) {
    if (!terminal) {
      nextTick(() => {
        initTerminal()
      })
    } else {
      nextTick(() => {
        writeContent(newContent)
      })
    }
  } else {
    terminal?.reset()
    lastContentLength = 0
    lastContent = ''
  }
})

onMounted(() => {
  if (terminalRef.value) {
    resizeObserver = new ResizeObserver(() => {
      fit()
    })
    resizeObserver.observe(terminalRef.value)
  }
  if (props.content && props.content.trim()) {
    nextTick(() => {
      initTerminal()
    })
  }
})

onUnmounted(() => {
  if (resizeObserver) {
    resizeObserver.disconnect()
  }
  terminal?.dispose()
  terminal = null
  fitAddon = null
})
</script>

<template>
  <div class="flex-1 flex flex-col h-full w-full relative min-h-0 lg:min-h-full min-w-0">
    <!-- 加载状态遮罩 -->
    <div
      v-if="loading"
      class="absolute inset-0 z-10 flex flex-col items-center justify-center p-4 select-none text-center bg-card/80 backdrop-blur-xs"
    >
      <Loader2 class="h-10 w-10 animate-spin text-primary/30 mb-4" />
      <span class="text-sm text-muted-foreground font-medium animate-pulse">{{ loadingText }}</span>
    </div>

    <!-- 空状态 -->
    <div
      v-else-if="!content || !content.trim()"
      class="flex-1 flex flex-col items-center justify-center py-10 lg:py-0 select-none text-center min-h-[160px]"
    >
      <div class="w-14 h-14 rounded-3xl bg-muted/20 flex items-center justify-center mb-4 border border-muted-foreground/10 mx-auto">
        <AlertCircle class="h-7 w-8 text-muted-foreground/20" />
      </div>
      <span class="text-sm text-muted-foreground font-medium">{{ emptyTitle }}</span>
      <p class="text-[11px] text-muted-foreground/40 mt-1.5 max-w-[280px] leading-relaxed mx-auto">
        {{ emptyDescription }}
      </p>
    </div>

    <!-- 正常内容 -->
    <div 
      v-show="content && content.trim()" 
      class="flex-1 w-full min-w-0 p-2 overflow-hidden h-[200px] lg:h-full lg:min-h-0"
    >
      <div ref="terminalRef" class="w-full h-full min-w-0 log-terminal"></div>
    </div>
  </div>
</template>

<style scoped>
.log-terminal {
  min-width: 0;
}

.log-terminal :deep(.xterm) {
  padding: 0.5rem;
  width: 100%;
  height: 100%;
}

.log-terminal :deep(.xterm-screen) {
  width: 100% !important;
}

.log-terminal :deep(.xterm-viewport) {
  scrollbar-width: thin;
  scrollbar-color: rgba(150, 150, 150, 0.3) transparent;
  -webkit-overflow-scrolling: touch;
  overscroll-behavior-y: contain;
}

.log-terminal :deep(.xterm-viewport::-webkit-scrollbar) {
  width: 6px;
  height: 6px;
}

.log-terminal :deep(.xterm-viewport::-webkit-scrollbar-track) {
  background: transparent;
}

.log-terminal :deep(.xterm-viewport::-webkit-scrollbar-thumb) {
  background: rgba(150, 150, 150, 0.3);
  border-radius: 4px;
}

.log-terminal :deep(.xterm-viewport::-webkit-scrollbar-thumb:hover) {
  background: rgba(150, 150, 150, 0.5);
}
</style>
