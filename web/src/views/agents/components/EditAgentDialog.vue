<script setup lang="ts">
import { ref } from 'vue'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter, DialogDescription } from '@/components/ui/dialog'
import { Label } from '@/components/ui/label'
import { Input } from '@/components/ui/input'
import { Switch } from '@/components/ui/switch'
import { Button } from '@/components/ui/button'
import { api, type Agent } from '@/api'
import { toast } from 'vue-sonner'

const emit = defineEmits<{
  (e: 'updated'): void
}>()

const isOpen = ref(false)
const editingAgent = ref<Agent | null>(null)
const customScheduler = ref(false)
const formData = ref({
  name: '',
  description: '',
  scheduler_config: {
    worker_count: 1,
    queue_size: 100,
    rate_interval: 200,
    verbose: false,
    strict_queue: false
  }
})

function openDialog(agent: Agent) {
  editingAgent.value = agent
  customScheduler.value = !!agent.scheduler_config
  formData.value = {
    name: agent.name,
    description: agent.description,
    scheduler_config: agent.scheduler_config ? {
      worker_count: agent.scheduler_config.worker_count,
      queue_size: agent.scheduler_config.queue_size,
      rate_interval: agent.scheduler_config.rate_interval,
      verbose: agent.scheduler_config.verbose,
      strict_queue: agent.scheduler_config.strict_queue
    } : {
      worker_count: 1,
      queue_size: 100,
      rate_interval: 200,
      verbose: false,
      strict_queue: false
    }
  }
  isOpen.value = true
}

async function updateAgent() {
  if (!editingAgent.value || !formData.value.name.trim()) return
  try {
    const payload = {
      name: formData.value.name,
      description: formData.value.description,
      enabled: editingAgent.value.enabled,
      scheduler_config: customScheduler.value ? formData.value.scheduler_config : null
    }
    await api.agents.update(editingAgent.value.id, payload)
    isOpen.value = false
    emit('updated')
    toast.success('更新成功')
  } catch (e: unknown) {
    toast.error((e as Error).message || '更新失败')
  }
}

defineExpose({ openDialog })
</script>

<template>
  <Dialog v-model:open="isOpen">
    <DialogContent>
      <DialogHeader>
        <DialogTitle>编辑 Agent</DialogTitle>
        <DialogDescription class="sr-only">修改 Agent 的名称和描述信息</DialogDescription>
      </DialogHeader>
      <div class="space-y-4">
        <div class="space-y-1.5">
          <Label class="text-xs font-medium text-foreground">名称</Label>
          <Input v-model="formData.name" placeholder="Agent 名称" class="h-9" />
        </div>
        <div class="space-y-1.5">
          <Label class="text-xs font-medium text-foreground">描述</Label>
          <Input v-model="formData.description" placeholder="描述信息（可选）" class="h-9" />
        </div>
        <div class="flex items-center justify-between rounded-lg border p-3 shadow-sm">
          <div class="space-y-0.5">
            <Label class="text-sm font-medium">自定义调度配置</Label>
            <div class="text-xs text-muted-foreground">开启后可独立配置该 Agent 的并发限制与任务排队参数</div>
          </div>
          <Switch v-model="customScheduler" />
        </div>
        <div v-if="customScheduler" class="p-4 rounded-lg border border-border bg-muted/20 space-y-4 mt-2">
          <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <div class="space-y-1.5">
              <Label class="text-xs font-medium text-foreground">并发限制数 (Workers)</Label>
              <Input type="number" v-model.number="formData.scheduler_config.worker_count" :min="1" class="h-9" />
              <p class="text-[10px] text-muted-foreground">同一时间最大并行任务数</p>
            </div>
            <div class="space-y-1.5">
              <Label class="text-xs font-medium text-foreground">最大队列数 (Queue Size)</Label>
              <Input type="number" v-model.number="formData.scheduler_config.queue_size" :min="1" class="h-9" />
              <p class="text-[10px] text-muted-foreground">并发满时等待排队的任务数</p>
            </div>
          </div>
          <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <div class="space-y-1.5">
              <Label class="text-xs font-medium text-foreground">调度频率限制 (Rate Interval)</Label>
              <div class="relative">
                <Input type="number" v-model.number="formData.scheduler_config.rate_interval" :min="0" class="h-9 pr-10" />
                <span class="absolute right-3 top-1/2 -translate-y-1/2 text-xs text-muted-foreground">ms</span>
              </div>
              <p class="text-[10px] text-muted-foreground">两次调度启动的最小间隔时间</p>
            </div>
            <div class="space-y-1.5">
              <Label class="text-xs font-medium text-foreground">执行降级策略</Label>
              <div class="flex items-center justify-between rounded-md border border-input bg-card px-3 h-9 shadow-sm">
                <span class="text-xs text-muted-foreground">队列满时拒绝执行</span>
                <Switch v-model="formData.scheduler_config.strict_queue" class="scale-90" />
              </div>
              <p class="text-[10px] text-muted-foreground">开启后拒绝执行；关闭则同步直接执行</p>
            </div>
          </div>
          <div class="rounded-md bg-yellow-500/10 border border-yellow-500/20 p-2.5 text-[10px] text-yellow-600 dark:text-yellow-400 leading-relaxed">
            <strong>提示：</strong>开启严格排队后，服务端将不再拦截此 Agent 上已运行任务的并行触发，而是交由 Agent 本地队列调度。若要严格保证任务不并行，请将 <strong>并发限制数 (Workers)</strong> 设为 1。
          </div>
        </div>
      </div>
      <DialogFooter>
        <Button variant="outline" @click="isOpen = false">取消</Button>
        <Button @click="updateAgent">保存</Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>
