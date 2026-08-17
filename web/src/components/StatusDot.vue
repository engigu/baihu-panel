<script setup lang="ts">
import { computed } from 'vue'

const props = withDefaults(
  defineProps<{
    state: string
    title?: string
  }>(),
  {
    state: 'none'
  }
)

const dotClasses = computed(() => {
  switch (props.state) {
    case 'running':
      return 'h-2 w-2 bg-amber-500 animate-pulse shadow-[0_0_8px_rgba(245,158,11,0.5)]'
    case 'pending':
      return 'h-2 w-2 bg-blue-400 animate-pulse'
    case 'success':
    case 'online':
      return 'h-1.5 w-1.5 bg-green-500'
    case 'failed':
      return 'h-1.5 w-1.5 bg-red-500'
    case 'timeout':
      return 'h-1.5 w-1.5 bg-orange-500'
    case 'cancelled':
      return 'h-1.5 w-1.5 bg-muted-foreground'
    case 'offline':
    case 'none':
    default:
      return 'h-1.5 w-1.5 bg-muted-foreground/20'
  }
})
</script>

<template>
  <div class="rounded-full shrink-0" :class="dotClasses" :title="title"></div>
</template>
