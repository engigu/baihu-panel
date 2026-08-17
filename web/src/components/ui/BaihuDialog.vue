<script setup lang="ts">
import { X } from 'lucide-vue-next'
import * as Icons from 'lucide-vue-next'
import { computed } from 'vue'
import { 
  Dialog, 
  DialogContent, 
  DialogTitle, 
  DialogDescription,
  DialogClose
} from '@/components/ui/dialog'

interface Props {
  open: boolean
  title?: string
  description?: string
  icon?: string
  size?: 'sm' | 'md' | 'lg' | 'xl' | 'full'
  showClose?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  title: '',
  description: '',
  size: 'md',
  showClose: true,
  icon: ''
})

const iconComponent = computed(() => {
  if (!props.icon) return null
  return (Icons as any)[props.icon]
})

const emit = defineEmits<{
  'update:open': [value: boolean]
}>()

const sizeClasses = {
  sm: 'sm:max-w-[425px]',
  md: 'sm:max-w-[600px]',
  lg: 'sm:max-w-[800px]',
  xl: 'sm:max-w-[1000px]',
  full: 'sm:max-w-[95vw] h-[90vh]'
}
</script>

<template>
  <Dialog :open="open" @update:open="emit('update:open', $event)">
    <DialogContent 
      :show-close-button="false"
      :class="[
        'overflow-hidden border-none p-0 bg-background/90 backdrop-blur-xl shadow-2xl ring-1 ring-black/5 dark:ring-white/10 rounded-xl',
        sizeClasses[size],
        'transition-all duration-300'
      ]"
    >
      <!-- 统一的 Header 风格 -->
      <div class="flex items-center px-4 h-11 border-b bg-muted/20 shrink-0 gap-2 overflow-hidden">
        <!-- 图标 -->
        <component :is="iconComponent" v-if="iconComponent" class="h-4 w-4 text-primary shrink-0" />
        <!-- 标题 (a11y 语义) -->
        <DialogTitle class="text-[13px] font-normal text-muted-foreground tracking-wide shrink-0 whitespace-nowrap">
          {{ title }}
        </DialogTitle>
        <!-- 描述 / 副标题插槽 (占满剩余宽度，截断溢出) -->
        <div class="flex-1 min-w-0 overflow-hidden flex items-center justify-between gap-2">
          <div class="truncate">
            <DialogDescription v-if="description || $slots.description" class="truncate text-xs leading-none">
              <slot name="description">{{ description }}</slot>
            </DialogDescription>
            <DialogDescription v-else class="sr-only">
              {{ title || '提示对话框' }}
            </DialogDescription>
          </div>
          <!-- 额外扩展插槽 -->
          <div v-if="$slots.extra" class="flex items-center shrink-0">
            <slot name="extra" />
          </div>
        </div>
        <!-- 关闭按钮 -->
        <DialogClose
          v-if="showClose"
          class="h-7 w-7 rounded-md opacity-70 transition-all hover:opacity-100 hover:bg-muted flex items-center justify-center shrink-0 ml-auto -mr-1"
        >
          <X class="h-4 w-4 text-muted-foreground" />
          <span class="sr-only">关闭</span>
        </DialogClose>
      </div>

      <!-- 内容区域 (保持整洁简洁) -->
      <div class="p-4 sm:p-6">
        <div class="animate-in fade-in slide-in-from-bottom-1 duration-400">
          <slot />
        </div>

        <!-- 底部插槽 -->
        <div v-if="$slots.footer" class="mt-8 flex flex-col-reverse sm:flex-row sm:justify-end sm:space-x-2 gap-2">
          <slot name="footer" />
        </div>
      </div>
    </DialogContent>
  </Dialog>
</template>

<style scoped>
:deep([data-state='open']) {
  animation: baihu-dialog-in 0.3s cubic-bezier(0.16, 1, 0.3, 1);
}

:deep([data-state='closed']) {
  animation: baihu-dialog-out 0.2s cubic-bezier(0.16, 1, 0.3, 1);
}

@keyframes baihu-dialog-in {
  from {
    opacity: 0;
    transform: translate(-50%, -48%) scale(0.96);
  }
  to {
    opacity: 1;
    transform: translate(-50%, -50%) scale(1);
  }
}

@keyframes baihu-dialog-out {
  from {
    opacity: 1;
    transform: translate(-50%, -50%) scale(1);
  }
  to {
    opacity: 0;
    transform: translate(-50%, -48%) scale(0.96);
  }
}
</style>
