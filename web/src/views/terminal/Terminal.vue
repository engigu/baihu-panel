<script setup lang="ts">
import { ref } from 'vue'
import { Button } from '@/components/ui/button'
import { RefreshCw } from 'lucide-vue-next'
import XTerminal from '@/components/XTerminal.vue'

const terminalRef = ref<InstanceType<typeof XTerminal> | null>(null)

function reconnect() {
  terminalRef.value?.reconnect()
}
</script>

<template>
  <div class="flex flex-col h-[calc(100vh-120px)] sm:h-[calc(100vh-100px)]">
    <div class="flex items-center justify-between p-2 border rounded-t-md bg-[#252526]">
      <span class="text-xs font-medium text-gray-300">终端</span>
      <Button variant="ghost" size="icon" class="h-6 w-6 text-gray-400 hover:text-white" @click="reconnect" title="重新连接">
        <RefreshCw class="h-3 w-3" />
      </Button>
    </div>
    <div class="flex-1 border border-t-0 rounded-b-md overflow-hidden">
      <XTerminal ref="terminalRef" :font-size="13" />
    </div>
  </div>
</template>
