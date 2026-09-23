<script setup lang="ts">
import { ref } from 'vue'
import BaihuDialog from '@/components/ui/BaihuDialog.vue'
import { Button } from '@/components/ui/button'
import { Copy, Check } from 'lucide-vue-next'
import { toast } from 'vue-sonner'

import { copyToClipboard } from '@/utils/clipboard'

const props = withDefaults(
  defineProps<{
    text: string
    title?: string
    disableDialog?: boolean
  }>(),
  {
    title: '详情',
    disableDialog: false
  }
)

const showDialog = ref(false)
const copied = ref(false)

function handleClick() {
  if (props.disableDialog) return
  if (props.text && props.text !== '-') {
    showDialog.value = true
  }
}

async function handleCopy() {
  if (!props.text || props.text === '-') return
  const success = await copyToClipboard(props.text)
  if (success) {
    copied.value = true
    toast.success('已复制到剪贴板')
    setTimeout(() => {
      copied.value = false
    }, 2000)
  } else {
    toast.error('复制失败')
  }
}
</script>

<template>
  <span v-bind="$attrs" 
    class="truncate block transition-colors" 
    :class="[!disableDialog ? 'cursor-pointer hover:text-primary' : '']"
    :title="text || '-'"
    @click="e => { if (!disableDialog) { e.stopPropagation(); handleClick(); } }">
    {{ text || '-' }}
  </span>

  <BaihuDialog v-model:open="showDialog" :title="title">
    <div class="space-y-2.5">
      <!-- 顶部操作栏：包含一键复制按钮 -->
      <div class="flex items-center justify-end -mt-1">
        <Button
          variant="outline"
          size="sm"
          class="h-7 px-2.5 text-xs gap-1.5 transition-all hover:bg-accent"
          @click="handleCopy"
        >
          <Check v-if="copied" class="h-3.5 w-3.5 text-emerald-500" />
          <Copy v-else class="h-3.5 w-3.5 text-muted-foreground" />
          <span>{{ copied ? '已复制' : '复制内容' }}</span>
        </Button>
      </div>

      <div class="max-h-[60vh] overflow-y-auto custom-scrollbar">
        <div class="p-4 bg-muted/30 rounded-xl border border-border/50 select-text">
          <p class="text-[13px] leading-relaxed text-foreground/90 break-all whitespace-pre-wrap font-mono">
            {{ text }}
          </p>
        </div>
      </div>
    </div>
  </BaihuDialog>
</template>
