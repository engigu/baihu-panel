<script setup lang="ts">
import { ref, watch } from 'vue'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { Switch } from '@/components/ui/switch'
import { api, type Workflow } from '@/api'
import { toast } from 'vue-sonner'

const props = defineProps<{
  modelValue: boolean
  workflow: Partial<Workflow>
  isEdit: boolean
}>()

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  'saved': []
}>()

const open = ref(props.modelValue)
const isSubmitting = ref(false)
const formData = ref<Partial<Workflow>>({ ...props.workflow })

watch(() => props.modelValue, (val) => {
  open.value = val
  if (val) {
    formData.value = { ...props.workflow }
  }
})

watch(open, (val) => {
  emit('update:modelValue', val)
})

async function save() {
  if (!formData.value.name?.trim()) {
    toast.error('请输入工作流名称')
    return
  }

  isSubmitting.value = true
  try {
    const payload = {
      ...formData.value,
      enabled: formData.value.enabled ?? true
    }

    if (props.isEdit && formData.value.id) {
      await api.workflows.update(formData.value.id, payload)
      toast.success('工作流已更新')
    } else {
      await api.workflows.create(payload)
      toast.success('工作流已创建')
    }
    emit('saved')
    open.value = false
  } catch (err: any) {
    toast.error(err.message || '保存失败')
  } finally {
    isSubmitting.value = false
  }
}
</script>

<template>
  <Dialog :open="open" @update:open="open = $event">
    <DialogContent class="sm:max-w-[425px]">
      <DialogHeader>
        <DialogTitle>{{ isEdit ? '编辑工作流' : '创建工作流' }}</DialogTitle>
        <DialogDescription>
          工作流可以串联多个任务，当上游任务完成时自动触发下游任务。
        </DialogDescription>
      </DialogHeader>

      <div class="grid gap-4 py-4">
        <div class="grid grid-cols-4 items-center gap-4">
          <Label for="name" class="text-right">名称</Label>
          <Input id="name" v-model="formData.name" placeholder="输入名称, 如: 每日构建报表" class="col-span-3" />
        </div>

        <div class="grid grid-cols-4 items-center gap-4">
          <Label for="desc" class="text-right">描述</Label>
          <Textarea id="desc" v-model="formData.description" placeholder="关于该流的简要说明..." class="col-span-3 min-h-[80px]" />
        </div>

        <!-- Optional global schedule -->
        <div class="grid grid-cols-4 items-center gap-4">
          <Label for="schedule" class="text-right whitespace-nowrap overflow-hidden text-ellipsis">定时运行 (Cron) <span class="text-[10px] text-muted-foreground ml-1">(可选)</span></Label>
          <Input id="schedule" v-model="formData.schedule" placeholder="* * * * * *" class="col-span-3" />
        </div>

        <div class="grid grid-cols-4 items-center gap-4">
          <Label class="text-right">启用状态</Label>
          <div class="col-span-3 flex items-center h-9">
            <Switch :checked="formData.enabled !== false" @update:checked="formData.enabled = $event" />
          </div>
        </div>
      </div>

      <DialogFooter>
        <Button variant="outline" @click="open = false" :disabled="isSubmitting">取消</Button>
        <Button type="submit" @click="save" :disabled="isSubmitting">
          {{ isSubmitting ? '保存中...' : '保存' }}
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>
