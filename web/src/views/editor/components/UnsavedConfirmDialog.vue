<script setup lang="ts">
import { Button } from '@/components/ui/button'
import BaihuDialog from '@/components/ui/BaihuDialog.vue'
import { AlertCircle } from 'lucide-vue-next'

const props = defineProps<{
  open: boolean
  path: string
}>()

const emit = defineEmits<{
  'update:open': [value: boolean]
  'confirm': []
}>()
</script>

<template>
  <BaihuDialog :open="open" @update:open="emit('update:open', $event)" title="未保存的更改" icon="AlertCircle" size="sm">
    <div class="flex flex-col sm:flex-row items-center sm:items-start gap-4 p-1">
      <div class="h-12 w-12 rounded-full bg-yellow-500/10 flex items-center justify-center shrink-0">
        <AlertCircle class="h-6 w-6 text-yellow-500" />
      </div>
      <div class="flex-1 text-center sm:text-left">
        <p class="text-sm text-foreground/90 leading-relaxed font-medium">文件尚未保存</p>
        <p class="text-[13px] text-muted-foreground mt-1">
          当前文件有未保存的更改。如果现在离开，您的更改将会丢失。
        </p>
        <div class="mt-3 px-2 py-1.5 bg-muted/50 rounded text-[11px] font-mono text-muted-foreground break-all border border-muted">
          {{ path }}
        </div>
      </div>
    </div>
    <template #footer>
      <div class="flex flex-col-reverse sm:flex-row gap-2 w-full sm:w-auto mt-2 sm:mt-0">
        <Button variant="outline" @click="emit('update:open', false)" class="w-full sm:w-24">继续编辑</Button>
        <Button variant="destructive" @click="emit('update:open', false); emit('confirm')" class="w-full sm:w-auto px-6">放弃更改</Button>
      </div>
    </template>
  </BaihuDialog>
</template>
