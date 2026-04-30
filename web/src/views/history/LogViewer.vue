<script setup lang="ts">
import { ref, watch, onUnmounted } from 'vue'
import LogDetailCard from '@/components/LogDetailCard.vue'
import type { TaskLog } from '@/api'

const props = withDefaults(defineProps<{
  open: boolean
  log: TaskLog | null
  content: string
  title?: string
  loading?: boolean
  variant?: 'full' | 'simple'
  isStopping?: boolean
  emptyTitle?: string
  emptyDescription?: string
}>(), {
  variant: 'simple',
  isStopping: false
})

const emit = defineEmits<{
  'update:open': [value: boolean]
  'stop': []
}>()

const isFullscreen = ref(false)

function close() {
  isFullscreen.value = false
  emit('update:open', false)
}

// 统一控制 Body 滚动
function toggleBodyScroll(lock: boolean) {
  if (lock) {
    document.body.style.overflow = 'hidden'
  } else {
    document.body.style.overflow = ''
  }
}

// 监听打开状态
watch(() => props.open, (val) => {
  if (val) {
    isFullscreen.value = false
    toggleBodyScroll(true)
  } else {
    toggleBodyScroll(false)
  }
}, { immediate: true })

// 确保组件卸载时恢复滚动
onUnmounted(() => {
  toggleBodyScroll(false)
})
</script>

<template>
  <Teleport to="body">
    <div v-if="open" class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-0 sm:p-4"
      @click.self="close">
      <div
        :class="[
          'bg-background shadow-2xl flex flex-col transition-all duration-300 overflow-hidden',
          isFullscreen 
            ? 'fixed inset-0 w-screen h-screen z-[60] rounded-none' 
            : 'w-full sm:w-[90vw] md:w-[80vw] max-w-5xl h-[90vh] sm:h-[85vh] rounded-xl'
        ]">
        <LogDetailCard 
          class="flex-1"
          :log="log" 
          :content="content" 
          :title="title"
          :loading="loading"
          :is-stopping="isStopping"
          :variant="isFullscreen ? 'simple' : variant"
          :empty-title="emptyTitle"
          :empty-description="emptyDescription"
          @close="close"
          @maximize="isFullscreen = !isFullscreen"
          @stop="$emit('stop')"
        />
      </div>
    </div>
  </Teleport>
</template>
