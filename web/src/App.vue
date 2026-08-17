<script setup lang="ts">
import { onMounted } from 'vue'
import { RouterView } from 'vue-router'
import { Toaster } from '@/components/ui/sonner'
import { LogOut } from 'lucide-vue-next'
import { activeInterconnectNodeId, activeInterconnectNodeName, setActiveInterconnectNodeId } from '@/api'

import { isWindowsPlatform } from '@/windows/platform'

function exitTravel() {
  setActiveInterconnectNodeId('')
  window.location.href = '/'
}

onMounted(() => {
  // 全局应用设备差异化垂直抗锯齿
  const isMac = /Mac|iPod|iPhone|iPad/.test(navigator.userAgent)
  document.documentElement.classList.add(isMac ? 'antialiased' : 'subpixel-antialiased')

  // Windows 平台检测，用于在 CSS 中调整字体优先级
  if (isWindowsPlatform()) {
    document.documentElement.classList.add('is-windows')
  }
})
</script>

<template>
  <RouterView />
  <Toaster position="bottom-right" :duration="2000" />
  
  <!-- 穿越子节点时显示的悬浮“返回主节点”控制条 -->
  <div v-if="activeInterconnectNodeId" class="fixed bottom-4 left-4 z-[9999] group">
    <button 
      @click="exitTravel" 
      class="flex items-center gap-1.5 px-2.5 py-1 rounded-full text-[10px] font-medium 
             bg-amber-500/85 hover:bg-amber-500 text-white shadow-md shadow-amber-500/10
             transition-all duration-300 border border-amber-400/10 backdrop-blur-sm
             hover:scale-105 active:scale-95 cursor-pointer"
    >
      <LogOut class="h-3 w-3 transition-transform group-hover:-translate-x-0.5" />
      <span>返回主节点 <template v-if="activeInterconnectNodeName">当前:({{ activeInterconnectNodeName }})</template></span>
    </button>
  </div>
</template>
