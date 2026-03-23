<script setup lang="ts">
import { ref, watch, onUnmounted, nextTick } from 'vue'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { X, Search } from 'lucide-vue-next'
import Ansi from 'ansi-to-vue3'

const props = defineProps<{
  open: boolean
  title: string
  content: string
  status?: string
}>()

const emit = defineEmits<{
  'update:open': [value: boolean]
}>()

const searchKeyword = ref('')
const scrollContainer = ref<HTMLElement | null>(null)
const shouldAutoScroll = ref(true)

function close() {
  emit('update:open', false)
}

// 处理滚动事件，判断用户是否手动向上滚动
function handleScroll() {
  if (!scrollContainer.value) return
  const { scrollTop, scrollHeight, clientHeight } = scrollContainer.value
  // 如果距离底部小于 50px，认为用户想继续跟随滚动
  const isAtBottom = scrollHeight - scrollTop - clientHeight < 50
  shouldAutoScroll.value = isAtBottom
}

// 滚动到底部
const scrollToBottom = async (smooth = true) => {
  await nextTick()
  if (scrollContainer.value && shouldAutoScroll.value) {
    scrollContainer.value.scrollTo({
      top: scrollContainer.value.scrollHeight,
      behavior: smooth ? 'smooth' : 'auto'
    })
  }
}

// 监听内容变化，实现自动滚动
watch(() => props.content, () => {
  if (props.open) {
    scrollToBottom(true)
  }
})

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
    searchKeyword.value = ''
    shouldAutoScroll.value = true // 每次重新打开都开启自动滚动
    toggleBodyScroll(true)
    scrollToBottom(false) // 首次打开立刻定位，不滑动
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
    <div v-if="open" class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-2 sm:p-4"
      @click.self="close">
      <div
        class="bg-background rounded-lg shadow-lg flex flex-col w-full sm:w-[90vw] md:w-[80vw] max-w-5xl h-[90vh] sm:h-[85vh]">
        <div
          class="flex flex-col sm:flex-row sm:items-center justify-between px-3 sm:px-4 py-2 sm:py-3 border-b shrink-0 gap-2">
          <div class="flex items-center gap-3 min-w-0">
            <span class="text-sm font-medium truncate">{{ title }}</span>
            <div v-if="status"
              class="flex items-center gap-1.5 px-2 py-0.5 rounded text-[10px] font-bold uppercase transition-colors shrink-0"
              :class="status === 'success' ? 'bg-green-500/10 text-green-500 border border-green-500/20' :
                status === 'failed' ? 'bg-red-500/10 text-red-500 border border-red-500/20' :
                  'bg-yellow-500/10 text-yellow-500 border border-yellow-500/20'">
              <span v-if="status === 'running'" class="relative flex h-1.5 w-1.5 mr-0.5">
                <span
                  class="animate-ping absolute inline-flex h-full w-full rounded-full bg-yellow-400 opacity-75"></span>
                <span class="relative inline-flex rounded-full h-1.5 w-1.5 bg-yellow-500"></span>
              </span>
              {{ status === 'success' ? '成功' : status === 'failed' ? '失败' : '执行中' }}
            </div>
          </div>
          <div class="flex items-center gap-2">
            <div class="relative flex-1 sm:flex-none">
              <Search class="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
              <Input v-model="searchKeyword" placeholder="搜索内容..." class="h-8 pl-9 w-full sm:w-56 text-sm" />
            </div>
            <Button variant="ghost" size="icon" class="h-7 w-7 shrink-0" @click="close">
              <X class="h-4 w-4" />
            </Button>
          </div>
        </div>
        <div ref="scrollContainer" class="flex-1 overflow-auto bg-black/5 dark:bg-white/5" @scroll="handleScroll">
          <div class="p-3 sm:p-4 text-xs font-mono whitespace-pre-wrap break-all leading-relaxed">
            <Ansi>{{ content }}</Ansi>
          </div>
        </div>
      </div>
    </div>
  </Teleport>
</template>
