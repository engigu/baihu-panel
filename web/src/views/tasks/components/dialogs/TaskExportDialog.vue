<script setup lang="ts">
import BaihuDialog from '@/components/ui/BaihuDialog.vue'
import { Button } from '@/components/ui/button'
import { Copy } from 'lucide-vue-next'
import { copyToClipboard } from '@/utils/clipboard'
import { toast } from 'vue-sonner'

const props = defineProps<{
  open: boolean
  commandText: string
}>()

const emit = defineEmits<{
  'update:open': [value: boolean]
}>()

function copyText() {
  if (!props.commandText) return
  copyToClipboard(props.commandText).then((success) => {
    if (success) {
      toast.success('同步指令已复制到剪贴板')
      emit('update:open', false)
    } else {
      toast.error('复制失败，请手动选择复制')
    }
  })
}
</script>

<template>
  <BaihuDialog :open="open" title="导出同步指令" @update:open="$emit('update:open', $event)">
    <div class="space-y-4 py-2">
      <div class="text-xs text-muted-foreground leading-relaxed">
        您可以复制并分享此指令，在其他部署了白虎面板的系统上导入该仓库同步任务：
      </div>
      <div class="relative">
        <textarea
          readonly
          :value="commandText"
          rows="6"
          class="w-full p-4 font-mono text-[11px] leading-normal bg-muted/50 border border-muted-foreground/10 rounded-lg resize-none select-all focus:outline-none focus:ring-1 focus:ring-primary shadow-inner"
        />
      </div>
    </div>
    <template #footer>
      <Button variant="ghost" size="sm" @click="$emit('update:open', false)">关闭</Button>
      <Button size="sm" class="gap-1.5 shadow-md shadow-primary/20 font-medium" @click="copyText">
        <Copy class="h-3.5 w-3.5" />
        <span>一键复制</span>
      </Button>
    </template>
  </BaihuDialog>
</template>
