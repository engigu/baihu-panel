<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Button } from '@/components/ui/button'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle
} from '@/components/ui/alert-dialog'
import { api } from '@/api'
import { toast } from 'vue-sonner'

interface SchedulerSettings {
  worker_count: string
  queue_size: string
  rate_interval: string
}

const form = ref<SchedulerSettings>({
  worker_count: '4',
  queue_size: '100',
  rate_interval: '200'
})
const loading = ref(false)
const showConfirm = ref(false)

async function loadSettings() {
  try {
    const res = await api.settings.getScheduler()
    form.value = res
  } catch {}
}

function confirmSave() {
  showConfirm.value = true
}

async function saveSettings() {
  showConfirm.value = false
  loading.value = true
  try {
    await api.settings.updateScheduler({
      worker_count: String(form.value.worker_count),
      queue_size: String(form.value.queue_size),
      rate_interval: String(form.value.rate_interval)
    })
    toast.success('保存成功，调度配置已重新加载')
  } catch {
    toast.error('保存失败')
  } finally {
    loading.value = false
  }
}

onMounted(loadSettings)
</script>

<template>
  <div class="space-y-4">
    <div class="grid grid-cols-4 items-center gap-4">
      <Label class="text-right">Worker 数量</Label>
      <div class="col-span-3 flex items-center gap-2">
        <Input v-model="form.worker_count" type="number" class="w-24" />
        <span class="text-xs text-muted-foreground">并发执行任务的 worker 数量</span>
      </div>
    </div>
    <div class="grid grid-cols-4 items-center gap-4">
      <Label class="text-right">队列大小</Label>
      <div class="col-span-3 flex items-center gap-2">
        <Input v-model="form.queue_size" type="number" class="w-24" />
        <span class="text-xs text-muted-foreground">任务队列缓冲区大小</span>
      </div>
    </div>
    <div class="grid grid-cols-4 items-center gap-4">
      <Label class="text-right">速率间隔</Label>
      <div class="col-span-3 flex items-center gap-2">
        <Input v-model="form.rate_interval" type="number" class="w-24" />
        <span class="text-xs text-muted-foreground">ms，任务启动间隔（200ms = 每秒最多5个）</span>
      </div>
    </div>
    <div class="flex justify-end pt-2">
      <Button @click="confirmSave" :disabled="loading">
        {{ loading ? '保存中...' : '保存设置' }}
      </Button>
    </div>

    <AlertDialog :open="showConfirm" @update:open="showConfirm = $event">
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>确认保存</AlertDialogTitle>
          <AlertDialogDescription>
            保存后调度配置将立即生效，正在执行的任务不受影响。确定要保存吗？
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>取消</AlertDialogCancel>
          <AlertDialogAction @click="saveSettings">确认保存</AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  </div>
</template>
