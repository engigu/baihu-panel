<script setup lang="ts">
import { ref, onMounted, computed, onUnmounted } from 'vue'
import { Button } from '@/components/ui/button'
import { RefreshCw, Info, Copy, Check } from 'lucide-vue-next'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'
import XTerminal from '@/components/XTerminal.vue'
import { api } from '@/api'

const terminalRef = ref<InstanceType<typeof XTerminal> | null>(null)
const cmds = ref<{ name: string, description: string }[]>([])
const windowWidth = ref(window.innerWidth)
const updateWidth = () => { windowWidth.value = window.innerWidth }

onMounted(async () => {
  window.addEventListener('resize', updateWidth)
  try {
    const res = await api.terminal.cmds()
    cmds.value = res
  } catch (error) {
    console.error('Failed to load terminal commands', error)
  }
})

onUnmounted(() => {
  window.removeEventListener('resize', updateWidth)
})

const terminalFontSize = computed(() => {
  if (windowWidth.value < 640) return 11
  if (windowWidth.value < 1024) return 12
  return 13
})

const isReconnecting = ref(false)

function reconnect() {
  isReconnecting.value = true
  terminalRef.value?.reconnect()
  // Add a slight delay to the animation so it's visible even for fast reconnections
  setTimeout(() => {
    isReconnecting.value = false
  }, 1000)
}

const statusState = ref<{ text: string; type: 'success' | 'error' | 'info' } | null>(null)
function handleStatusChange(status: any) {
  statusState.value = status
}

import { copyToClipboard } from '@/utils/clipboard'

const copiedCommand = ref<string | null>(null)
const handleCopy = async (text: string) => {
  const success = await copyToClipboard(text)
  if (success) {
    copiedCommand.value = text
    setTimeout(() => {
      if (copiedCommand.value === text) {
        copiedCommand.value = null
      }
    }, 2000)
  }
}
</script>

<template>
  <div class="flex flex-col h-[calc(100vh-120px)] sm:h-[calc(100vh-100px)]">
    <div class="flex items-center justify-between p-2 border border-[#3c3c3c] rounded-t-md bg-[#252526]">
      <div class="flex items-center gap-2">
        <span class="text-xs font-medium text-gray-300">终端</span>
        <Popover>
          <PopoverTrigger as-child>
            <div class="flex items-center gap-1 cursor-pointer text-gray-400 hover:text-white transition-colors"
              title="查看内置命令">
              <span class="text-xs">内置命令</span>
              <Info class="h-3.5 w-3.5" />

            </div>
          </PopoverTrigger>
          <PopoverContent align="start" side="bottom" :side-offset="8"
            class="w-80 border-[#3c3c3c] bg-[#252526] text-gray-300">
            <div class="flex flex-col">
              <h4 class="text-xs font-medium text-white mb-1 pb-1.5 border-b border-[#3c3c3c]">内置命令说明</h4>
              <div v-if="cmds.length === 0" class="text-xs text-gray-500 py-1">获取中...</div>
              <div v-for="cmd in cmds" :key="cmd.name"
                class="group flex flex-col py-1.5 border-b border-[#3c3c3c] last:border-0 relative">
                <div class="flex items-start justify-between gap-2">
                  <div class="flex flex-col min-w-0">
                    <span class="font-bold text-blue-400 text-xs">baihu {{ cmd.name }}</span>
                    <span class="text-gray-400 text-[11px] leading-tight mt-0.5">{{ cmd.description }}</span>
                  </div>
                  <button @click.stop.prevent="handleCopy(`baihu ${cmd.name}`)" class="opacity-0 group-hover:opacity-100 transition-opacity p-1 hover:bg-[#3c3c3c] rounded text-gray-400 hover:text-white focus:outline-none focus:ring-0 shrink-0 mt-0.5" :title="copiedCommand === `baihu ${cmd.name}` ? '已复制' : '复制命令'">
                    <Check v-if="copiedCommand === `baihu ${cmd.name}`" class="h-3 w-3 text-green-500" />
                    <Copy v-else class="h-3 w-3" />
                  </button>
                </div>
              </div>
            </div>
          </PopoverContent>
        </Popover>
      </div>
      <div class="flex items-center gap-3">
        <!-- 终端状态指示 (右上角) -->
        <div v-if="statusState" class="flex items-center gap-1.5 transition-all animate-in fade-in slide-in-from-right-1">
          <div :class="[
            'w-1.5 h-1.5 rounded-full',
            statusState.type === 'success' ? 'bg-green-500 shadow-[0_0_8px_rgba(34,197,94,0.4)]' :
              statusState.type === 'error' ? 'bg-red-500 shadow-[0_0_8px_rgba(239,68,68,0.4)]' :
                'bg-blue-400 animate-pulse'
          ]" />
          <span :class="[
            'text-[11px] font-medium transition-colors',
            statusState.type === 'success' ? 'text-green-500/90' :
              statusState.type === 'error' ? 'text-red-500/90' :
                'text-blue-400/90'
          ]">
            {{ statusState.text }}
          </span>
        </div>

        <Button variant="ghost" size="icon" class="h-6 w-6 text-gray-400 hover:text-white" @click="reconnect"
          :disabled="isReconnecting" title="重新连接">
          <RefreshCw class="h-3 w-3" :class="{ 'animate-spin': isReconnecting }" />
        </Button>
      </div>
    </div>
    <div class="flex-1 border border-[#3c3c3c] border-t-0 rounded-b-md overflow-hidden bg-[#1e1e1e]">
      <XTerminal ref="terminalRef" :font-size="terminalFontSize" @status-change="handleStatusChange" />
    </div>
  </div>
</template>
