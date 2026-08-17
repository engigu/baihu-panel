<script setup lang="ts">
import { computed } from 'vue'
import { TASK_STATUS, TASK_STATUS_TEXT } from '@/constants'

const props = withDefaults(
  defineProps<{
    status: string
    showDot?: boolean
  }>(),
  {
    showDot: true
  }
)

const badgeClasses = computed(() => {
  switch (props.status) {
    case TASK_STATUS.SUCCESS:
      return 'border-green-500/20 text-green-600 dark:text-green-400 bg-green-500/10 dark:bg-green-500/20'
    case TASK_STATUS.FAILED:
      return 'border-red-500/20 text-red-600 dark:text-red-400 bg-red-500/10 dark:bg-red-500/20'
    case TASK_STATUS.RUNNING:
      return 'border-amber-500/20 text-amber-600 dark:text-amber-400 bg-amber-500/10 dark:bg-amber-500/20'
    case TASK_STATUS.PENDING:
    case 'queued':
      return 'border-blue-400/20 text-blue-500 dark:text-blue-400 bg-blue-400/10 dark:bg-blue-400/20'
    case TASK_STATUS.TIMEOUT:
      return 'border-orange-500/20 text-orange-600 dark:text-orange-400 bg-orange-500/10 dark:bg-orange-500/20'
    case TASK_STATUS.CANCELLED:
      return 'border-muted-foreground/10 text-muted-foreground bg-muted/50'
    default:
      return 'border-transparent text-secondary-foreground bg-secondary'
  }
})

const dotClasses = computed(() => {
  switch (props.status) {
    case TASK_STATUS.SUCCESS:
      return 'bg-green-500'
    case TASK_STATUS.FAILED:
      return 'bg-red-500'
    case TASK_STATUS.RUNNING:
      return 'bg-amber-500'
    case TASK_STATUS.PENDING:
    case 'queued':
      return 'bg-blue-400'
    case TASK_STATUS.TIMEOUT:
      return 'bg-orange-500'
    case TASK_STATUS.CANCELLED:
      return 'bg-muted-foreground'
    default:
      return 'bg-muted-foreground'
  }
})

const statusText = computed(() => {
  return TASK_STATUS_TEXT[props.status as keyof typeof TASK_STATUS_TEXT] || props.status
})
</script>

<template>
  <div 
    class="inline-flex items-center gap-1.5 px-2.5 py-0.5 text-[10px] font-normal rounded-full border transition-all duration-300 select-none"
    :class="badgeClasses"
  >
    <!-- 正在运行状态使用双层波纹微动效的呼吸灯 -->
    <span v-if="showDot && status === TASK_STATUS.RUNNING" class="relative flex h-1.5 w-1.5 shrink-0">
      <span class="animate-ping absolute inline-flex h-full w-full rounded-full bg-amber-400 opacity-75"></span>
      <span class="relative inline-flex rounded-full h-1.5 w-1.5 bg-amber-500"></span>
    </span>
    <!-- 其他状态使用常规实心圆点 -->
    <span 
      v-else-if="showDot" 
      class="h-1.5 w-1.5 rounded-full shrink-0" 
      :class="dotClasses"
    ></span>
    
    <span class="tracking-wide text-[10px]">{{ statusText }}</span>
  </div>
</template>
