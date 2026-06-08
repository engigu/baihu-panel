<script setup lang="ts">
import { ref } from 'vue'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription } from '@/components/ui/dialog'
import { Label } from '@/components/ui/label'
import { Wifi, WifiOff } from 'lucide-vue-next'
import { type Agent } from '@/api'
import { AGENT_STATUS } from '@/constants'

const isOpen = ref(false)
const viewingAgent = ref<Agent | null>(null)

function isOnline(agent: Agent): boolean {
  return agent.status === AGENT_STATUS.ONLINE
}

function openDialog(agent: Agent) {
  viewingAgent.value = agent
  isOpen.value = true
}

defineExpose({ openDialog })
</script>

<template>
  <Dialog v-model:open="isOpen">
    <DialogContent class="sm:max-w-md md:max-w-lg">
      <DialogHeader>
        <DialogTitle>Agent 详情</DialogTitle>
        <DialogDescription class="sr-only">显示 Agent 的详细配置和状态信息</DialogDescription>
      </DialogHeader>
      <div v-if="viewingAgent" class="space-y-3">
        <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
          <div class="flex items-center justify-between sm:block">
            <Label class="text-muted-foreground text-xs">ID</Label>
            <div class="text-sm font-medium">#{{ viewingAgent.id }}</div>
          </div>
          <div class="flex items-center justify-between sm:block">
            <Label class="text-muted-foreground text-xs">名称</Label>
            <div class="text-sm font-medium">{{ viewingAgent.name }}</div>
          </div>
          <div class="flex items-center justify-between sm:block">
            <Label class="text-muted-foreground text-xs">IP 地址</Label>
            <div class="text-sm">{{ viewingAgent.ip || '-' }}</div>
          </div>
          <div class="flex items-center justify-between sm:block">
            <Label class="text-muted-foreground text-xs">主机名</Label>
            <div class="text-sm">{{ viewingAgent.hostname || '-' }}</div>
          </div>
          <div class="flex items-center justify-between sm:block">
            <Label class="text-muted-foreground text-xs">操作系统</Label>
            <div class="text-sm">{{ viewingAgent.os || '-' }}</div>
          </div>
          <div class="flex items-center justify-between sm:block">
            <Label class="text-muted-foreground text-xs">架构</Label>
            <div class="text-sm">{{ viewingAgent.arch || '-' }}</div>
          </div>
          <div class="flex items-center justify-between sm:block">
            <Label class="text-muted-foreground text-xs">版本</Label>
            <div class="text-sm">{{ viewingAgent.version || '-' }}</div>
          </div>
          <div class="flex items-center justify-between sm:block">
            <Label class="text-muted-foreground text-xs">构建时间</Label>
            <div class="text-sm">{{ viewingAgent.build_time || '-' }}</div>
          </div>
          <div class="flex items-center justify-between sm:block">
            <Label class="text-muted-foreground text-xs">在线状态</Label>
            <div class="flex items-center gap-2">
              <Wifi v-if="isOnline(viewingAgent)" class="h-4 w-4 text-green-500" />
              <WifiOff v-else class="h-4 w-4 text-muted-foreground" />
              <span class="text-sm">{{ isOnline(viewingAgent) ? '在线' : '离线' }}</span>
            </div>
          </div>
          <div class="flex items-center justify-between sm:block">
            <Label class="text-muted-foreground text-xs">启用状态</Label>
            <div class="text-sm">{{ viewingAgent.enabled ? '已启用' : '已禁用' }}</div>
          </div>
          <div class="flex items-center justify-between sm:block">
            <Label class="text-muted-foreground text-xs">最后心跳</Label>
            <div class="text-sm">{{ viewingAgent.last_seen || '-' }}</div>
          </div>
          <div class="flex items-center justify-between sm:block">
            <Label class="text-muted-foreground text-xs">注册时间</Label>
            <div class="text-sm">{{ viewingAgent.created_at || '-' }}</div>
          </div>
          <div class="flex items-center justify-between sm:block">
            <Label class="text-muted-foreground text-xs">任务排队</Label>
            <div class="text-sm">
              {{ viewingAgent.scheduler_config ? `自定义 (并发: ${viewingAgent.scheduler_config.worker_count}, 队列: ${viewingAgent.scheduler_config.queue_size}, 严格排队: ${viewingAgent.scheduler_config.strict_queue ? '是' : '否'})` : '继承全局' }}
            </div>
          </div>
        </div>
        <div v-if="viewingAgent.description" class="pt-2 border-t">
          <Label class="text-muted-foreground text-xs">描述</Label>
          <div class="text-sm mt-1">{{ viewingAgent.description }}</div>
        </div>
      </div>
    </DialogContent>
  </Dialog>
</template>
