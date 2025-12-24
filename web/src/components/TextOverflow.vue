<script setup lang="ts">
import { ref } from 'vue'
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog'

const props = withDefaults(
  defineProps<{
    text: string
    title?: string
  }>(),
  {
    title: '详情'
  }
)

const showDialog = ref(false)

function handleClick() {
  if (props.text && props.text !== '-') {
    showDialog.value = true
  }
}
</script>

<template>
  <span
    class="truncate block cursor-pointer hover:text-primary"
    :title="text || '-'"
    @click="handleClick"
  >
    {{ text || '-' }}
  </span>

  <Dialog v-model:open="showDialog">
    <DialogContent class="sm:max-w-[600px]">
      <DialogHeader>
        <DialogTitle>{{ title }}</DialogTitle>
      </DialogHeader>
      <div class="max-h-[400px] overflow-y-auto">
        <pre class="text-sm whitespace-pre-wrap break-all font-mono bg-muted p-3 rounded-lg">{{ text }}</pre>
      </div>
    </DialogContent>
  </Dialog>
</template>
