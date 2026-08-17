<script setup lang="ts">
import { ref } from 'vue'
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle, DialogFooter } from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { api, type Task, type EnvVar } from '@/api'
import { toast } from 'vue-sonner'
import { AlertCircle, Terminal, RefreshCw, CalendarClock, Link } from 'lucide-vue-next'
import { ENV_TYPE } from '@/constants'

const isOpen = ref(false)
const isLoading = ref(false)
const envVar = ref<EnvVar | null>(null)
const tasks = ref<Task[]>([])

async function open(env: EnvVar) {
  envVar.value = env
  isOpen.value = true
  await loadTasks()
}

async function loadTasks() {
  if (!envVar.value?.id) return
  
  isLoading.value = true
  try {
    const res = await api.env.tasks(envVar.value.id)
    tasks.value = res || []
  } catch {
    toast.error('加载关联任务失败')
  } finally {
    isLoading.value = false
  }
}

function getTriggerIcon(type: string) {
  if (type === 'cron') return CalendarClock
  if (type === 'startup') return RefreshCw
  return Terminal
}

function getTriggerText(type: string) {
  if (type === 'cron') return '定时执行'
  if (type === 'startup') return '启动执行'
  return '终端任务'
}

defineExpose({
  open
})
</script>

<template>
  <Dialog v-model:open="isOpen">
    <DialogContent class="w-[calc(100vw-2rem)] max-w-md min-w-0 flex flex-col max-h-[85vh]">
      <DialogHeader class="shrink-0 text-left">
        <DialogTitle class="flex items-center gap-2">
          <Link class="h-4 w-4" />
          <span>依赖该{{ envVar?.type === ENV_TYPE.SECRET ? '机密' : '变量' }}的任务</span>
        </DialogTitle>
        <DialogDescription class="truncate" :title="envVar?.name">
          {{ envVar?.name }}
        </DialogDescription>
      </DialogHeader>

      <div class="flex-1 overflow-y-auto min-h-[200px] -mx-1 px-1">
        <div v-if="isLoading" class="flex items-center justify-center h-full text-muted-foreground text-sm py-12">
          加载中...
        </div>
        
        <div v-else-if="tasks.length === 0" class="flex flex-col items-center justify-center h-full text-muted-foreground text-sm py-12 gap-3">
          <div class="h-10 w-10 rounded-full bg-muted flex items-center justify-center">
            <AlertCircle class="h-5 w-5 opacity-50" />
          </div>
          <span>当前没有任务依赖此{{ envVar?.type === ENV_TYPE.SECRET ? '机密' : '变量' }}</span>
        </div>

        <div v-else class="space-y-2">
          <div v-for="task in tasks" :key="task.id" class="flex flex-col p-3 rounded-lg border bg-card/50 hover:bg-muted/50 transition-colors">
            <div class="flex items-start justify-between gap-3">
              <div class="flex flex-col min-w-0">
                <span class="font-medium text-sm truncate" :title="task.name">{{ task.name }}</span>
                <div class="flex items-center gap-1.5 mt-1">
                  <Badge variant="secondary" class="text-[9px] h-4 px-1 rounded uppercase font-medium">
                    <component :is="getTriggerIcon(task.trigger_type)" class="h-2.5 w-2.5 mr-1 inline-block" />
                    {{ getTriggerText(task.trigger_type) }}
                  </Badge>
                  <span v-if="task.schedule" class="text-[10px] text-muted-foreground bg-muted px-1 rounded truncate">
                    {{ task.schedule }}
                  </span>
                </div>
              </div>
              
              <div class="shrink-0 pt-0.5">
                <div v-if="task.enabled" class="px-1.5 py-0.5 rounded text-[10px] bg-green-500/10 text-green-600 dark:text-green-400 font-medium whitespace-nowrap">
                  已启用
                </div>
                <div v-else class="px-1.5 py-0.5 rounded text-[10px] bg-muted text-muted-foreground font-medium whitespace-nowrap">
                  已禁用
                </div>
              </div>
            </div>
            <div v-if="task.remark" class="mt-2 text-xs text-muted-foreground line-clamp-2">
              {{ task.remark }}
            </div>
          </div>
        </div>
      </div>

      <DialogFooter class="shrink-0 mt-4 sm:mt-4">
        <Button variant="outline" @click="isOpen = false">关闭</Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>
