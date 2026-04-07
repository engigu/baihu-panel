<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { Button } from '@/components/ui/button'
import { RefreshCw, Info } from 'lucide-vue-next'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'
import XTerminal from '@/components/XTerminal.vue'
import { api } from '@/api'

const terminalRef = ref<InstanceType<typeof XTerminal> | null>(null)
const cmds = ref<{ name: string, description: string }[]>([])

onMounted(async () => {
  try {
    const res = await api.terminal.cmds()
    cmds.value = res
  } catch (error) {
    console.error('Failed to load terminal commands', error)
  }
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
            <div class="space-y-2">
              <h4 class="text-sm font-medium text-white mb-2 pb-2 border-b border-[#3c3c3c]">内置命令说明</h4>
              <div v-if="cmds.length === 0" class="text-xs text-gray-500">获取中...</div>
              <div v-for="cmd in cmds" :key="cmd.name"
                class="flex flex-col space-y-1 text-xs border-b border-[#3c3c3c] pb-2 last:border-0 last:pb-0">
                <span class="font-bold text-blue-400">baihu {{ cmd.name }}</span>
                <span class="text-gray-400">{{ cmd.description }}</span>
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
      <XTerminal ref="terminalRef" :font-size="13" @status-change="handleStatusChange" />
    </div>
  </div>
</template>
