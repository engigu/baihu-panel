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
    <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
      <div class="space-y-1.5">
        <Label class="text-xs font-medium text-foreground">并发限制数 (Workers)</Label>
        <Input type="number" v-model="form.worker_count" :min="1" class="h-9" />
        <p class="text-[10px] text-muted-foreground">并发执行任务的 worker 数量</p>
      </div>
      <div class="space-y-1.5">
        <Label class="text-xs font-medium text-foreground">最大队列数 (Queue Size)</Label>
        <Input type="number" v-model="form.queue_size" :min="1" class="h-9" />
        <p class="text-[10px] text-muted-foreground">任务队列缓冲区大小</p>
      </div>
    </div>
    <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
      <div class="space-y-1.5">
        <Label class="text-xs font-medium text-foreground">调度频率限制 (Rate Interval)</Label>
        <div class="relative">
          <Input type="number" v-model="form.rate_interval" :min="0" class="h-9 pr-10" />
          <span class="absolute right-3 top-1/2 -translate-y-1/2 text-xs text-muted-foreground">ms</span>
        </div>
        <p class="text-[10px] text-muted-foreground">两次调度启动的最小间隔时间（例如 200ms = 每秒最多 5 个）</p>
      </div>
    </div>

    <div class="rounded-md bg-yellow-500/10 border border-yellow-500/20 p-2.5 text-[10px] text-yellow-600 dark:text-yellow-400 leading-relaxed mt-2">
      <strong>提示：</strong>此处配置仅对<strong>主服务</strong>的调度生效。如需修改 Agent 节点的调度设置，请前往 <strong>Agent 节点</strong> 页面为每个节点进行单独配置。
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
