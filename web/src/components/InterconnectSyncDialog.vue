<script setup lang="ts">
import { ref, watch } from 'vue'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import { toast } from 'vue-sonner'
import * as interconnectApi from '@/api/interconnect'

const props = defineProps<{
  open: boolean
  title: string
  description?: string
  actionLabel?: string
}>()

const emit = defineEmits<{
  (e: 'update:open', val: boolean): void
  (e: 'confirm', nodeIds: string[]): void
}>()

const nodes = ref<interconnectApi.InterconnectNode[]>([])
const selectedNodeIds = ref<Set<string>>(new Set())
const loading = ref(false)
const syncing = ref(false)

watch(() => props.open, async (val) => {
  if (val) {
    selectedNodeIds.value.clear()
    loading.value = true
    try {
      nodes.value = await interconnectApi.getNodes()
    } catch {
      toast.error('获取互联节点失败')
    } finally {
      loading.value = false
    }
  }
})

function toggleNode(id: string) {
  if (selectedNodeIds.value.has(id)) {
    selectedNodeIds.value.delete(id)
  } else {
    selectedNodeIds.value.add(id)
  }
}

function handleConfirm() {
  if (selectedNodeIds.value.size === 0) {
    toast.error('请至少选择一个目标节点')
    return
  }
  emit('confirm', Array.from(selectedNodeIds.value))
}

defineExpose({
  setSyncing: (val: boolean) => syncing.value = val,
  close: () => emit('update:open', false)
})
</script>

<template>
  <Dialog :open="open" @update:open="emit('update:open', $event)">
    <DialogContent class="sm:max-w-[425px]">
      <DialogHeader>
        <DialogTitle>{{ title }}</DialogTitle>
        <DialogDescription v-if="description">{{ description }}</DialogDescription>
      </DialogHeader>

      <div class="py-4 max-h-[300px] overflow-y-auto">
        <div v-if="loading" class="text-center text-sm text-muted-foreground py-4">
          加载中...
        </div>
        <div v-else-if="nodes.length === 0" class="text-center text-sm text-muted-foreground py-4">
          暂无互联节点，请先在「互联管理」中添加节点。
        </div>
        <div v-else class="space-y-3">
          <div 
            v-for="node in nodes" 
            :key="node.id" 
            class="flex items-center space-x-3 p-2 rounded-md border cursor-pointer hover:bg-secondary/50 transition-colors"
            @click="toggleNode(node.id)"
          >
            <Checkbox :checked="selectedNodeIds.has(node.id)" @update:checked="toggleNode(node.id)" />
            <div class="flex-1">
              <p class="text-sm font-medium leading-none">{{ node.name }}</p>
              <p class="text-xs text-muted-foreground mt-1">{{ node.url }}</p>
            </div>
          </div>
        </div>
      </div>

      <DialogFooter>
        <Button variant="outline" @click="emit('update:open', false)" :disabled="syncing">取消</Button>
        <Button @click="handleConfirm" :disabled="nodes.length === 0 || syncing">
          <span v-if="syncing">处理中...</span>
          <span v-else>{{ actionLabel || '确认分发' }}</span>
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>
