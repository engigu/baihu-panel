<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Checkbox } from '@/components/ui/checkbox'
import { Switch } from '@/components/ui/switch'
import { Bell } from 'lucide-vue-next'
import { api, type NotifyChannel, type NotifyBinding } from '@/api'
import { cn } from '@/lib/utils'

const props = defineProps<{
  taskId?: string
}>()

const notifyChannels = ref<NotifyChannel[]>([])
const notifyWayId = ref<string>('none')
const notifyOnSuccess = ref(false)
const notifyOnFailure = ref(false)
const notifyOnTimeout = ref(false)
const notifyIncludeLog = ref(false)
const notifyLogLimit = ref(1000)

onMounted(async () => {
  try {
    notifyChannels.value = await api.notify.getChannels()
  } catch (e) {
    console.error('Fetch channels failed', e)
  }
})

function resetConfig() {
  notifyWayId.value = 'none'
  notifyOnSuccess.value = false
  notifyOnFailure.value = false
  notifyOnTimeout.value = false
  notifyIncludeLog.value = false
  notifyLogLimit.value = 1000
}

async function loadConfig(taskId?: string) {
  if (!taskId) {
    resetConfig()
    return
  }

  try {
    const allBindings = await api.notify.getBindings()
    // 过滤出该任务的所有绑定
    const taskBindings = allBindings.filter(b => b.data_id === taskId && b.type === 'task')

    if (taskBindings.length > 0 && taskBindings[0]) {
      notifyWayId.value = taskBindings[0].way_id || 'none'
      
      // 直接设置，无需 setTimeout，内部已经安全了
      notifyOnSuccess.value = taskBindings.some(b => b.event === 'task_success')
      notifyOnFailure.value = taskBindings.some(b => b.event === 'task_failed')
      notifyOnTimeout.value = taskBindings.some(b => b.event === 'task_timeout')
      
      const extraBinding = taskBindings.find(b => b.extra && b.extra !== '')
      if (extraBinding && extraBinding.extra) {
        try {
          const extra = JSON.parse(extraBinding.extra)
          notifyIncludeLog.value = !!extra.enable_log
          notifyLogLimit.value = extra.log_limit || 1000
        } catch {
          notifyIncludeLog.value = false
          notifyLogLimit.value = 1000
        }
      } else {
        notifyIncludeLog.value = false
        notifyLogLimit.value = 1000
      }
    } else {
      resetConfig()
    }
  } catch (e) {
    console.error('Load notifications failed', e)
    resetConfig()
  }
}

async function saveConfig(taskId: string) {
  try {
    const bindings: Partial<NotifyBinding>[] = []

    if (notifyWayId.value !== 'none') {
      const events = [
        { type: 'task_success', enabled: notifyOnSuccess.value },
        { type: 'task_failed', enabled: notifyOnFailure.value },
        { type: 'task_timeout', enabled: notifyOnTimeout.value }
      ]

      const extra = JSON.stringify({
        enable_log: notifyIncludeLog.value,
        log_limit: notifyLogLimit.value
      })

      for (const event of events) {
        if (event.enabled) {
          bindings.push({
            event: event.type,
            way_id: notifyWayId.value,
            extra: extra
          })
        }
      }
    }

    // 调用批量保存，后端会自动清理该任务旧的绑定
    await api.notify.saveBindingsBatch({
      type: 'task',
      data_id: taskId,
      bindings: bindings
    })
  } catch (e) {
    console.error('Save notifications failed', e)
  }
}

defineExpose({
  loadConfig,
  saveConfig
})
</script>

<template>
  <section class="space-y-4">
    <div class="flex items-center gap-2 mb-2">
      <div class="h-4 w-1 bg-primary rounded-full shadow-sm shadow-primary/20" />
      <h3 class="text-sm font-bold text-foreground/90">通知配置</h3>
    </div>

    <div class="grid gap-5 pl-3 border-l border-muted">
      <div class="grid grid-cols-1 sm:grid-cols-4 items-center gap-3">
        <Label class="sm:text-right text-xs text-foreground/70 uppercase tracking-wider font-bold">通知渠道</Label>
        <div class="sm:col-span-3">
          <Select v-model="notifyWayId">
            <SelectTrigger class="h-9 bg-muted/20 border-muted-foreground/15 transition-all focus:bg-background/50">
              <SelectValue placeholder="不启用通知" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="none">不启用通知</SelectItem>
              <SelectItem v-for="ch in notifyChannels" :key="ch.id" :value="ch.id">
                {{ ch.name }}
              </SelectItem>
            </SelectContent>
          </Select>
        </div>
      </div>

      <template v-if="notifyWayId !== 'none'">
        <div class="grid grid-cols-1 sm:grid-cols-4 items-start gap-3">
          <Label class="sm:text-right text-xs text-foreground/70 uppercase tracking-wider font-bold pt-2.5">通知时机</Label>
          <div class="sm:col-span-3 space-y-3">
            <div class="flex flex-wrap gap-4 p-3 rounded-lg bg-muted/20 border border-muted-foreground/10 items-center transition-all hover:bg-muted/30">
              <div class="flex items-center gap-2 group">
                <Checkbox :id="`ns-${taskId || 'new'}`" :checked="notifyOnSuccess" @update:checked="notifyOnSuccess = $event as boolean" class="border-muted-foreground/30 data-[state=checked]:bg-primary data-[state=checked]:border-primary" />
                <label :for="`ns-${taskId || 'new'}`" class="text-xs font-medium shrink-0 cursor-pointer group-hover:text-primary transition-colors text-foreground/80">成功时</label>
              </div>
              <div class="flex items-center gap-2 group">
                <Checkbox :id="`nf-${taskId || 'new'}`" :checked="notifyOnFailure" @update:checked="notifyOnFailure = $event as boolean" class="border-muted-foreground/30 data-[state=checked]:bg-primary data-[state=checked]:border-primary" />
                <label :for="`nf-${taskId || 'new'}`" class="text-xs font-medium shrink-0 cursor-pointer group-hover:text-primary transition-colors text-foreground/80">失败时</label>
              </div>
              <div class="flex items-center gap-2 group">
                <Checkbox :id="`nt-${taskId || 'new'}`" :checked="notifyOnTimeout" @update:checked="notifyOnTimeout = $event as boolean" class="border-muted-foreground/30 data-[state=checked]:bg-primary data-[state=checked]:border-primary" />
                <label :for="`nt-${taskId || 'new'}`" class="text-xs font-medium shrink-0 cursor-pointer group-hover:text-primary transition-colors text-foreground/80">超时时</label>
              </div>
            </div>

            <div class="p-3 rounded-xl bg-primary/5 border border-primary/10 space-y-3 transition-all hover:bg-primary/10">
              <div class="flex items-center justify-between">
                <div class="flex items-center gap-2 text-xs font-bold text-foreground/90">
                  <Bell :class="cn('h-3.5 w-3.5 transition-colors', notifyIncludeLog ? 'text-primary' : 'text-muted-foreground/50')" />
                  附带执行日志
                </div>
                <Switch v-model="notifyIncludeLog" class="data-[state=checked]:bg-primary" />
              </div>
              
              <div v-if="notifyIncludeLog" class="flex items-center gap-2 animate-in fade-in slide-in-from-top-1 duration-200 pl-5">
                <div class="flex items-center gap-2 px-3 py-1.5 rounded-full bg-background/80 border border-primary/20 shadow-sm transition-all focus-within:ring-2 focus-within:ring-primary/20">
                  <span class="text-[10px] text-foreground/60 font-medium whitespace-nowrap uppercase tracking-tighter">长度限制</span>
                  <div class="h-3 w-[1px] bg-muted-foreground/20" />
                  <div class="flex items-center gap-1">
                    <input type="text" inputmode="numeric" :value="notifyLogLimit" @input="(e: any) => notifyLogLimit = Number(e.target.value.replace(/\D/g, ''))" 
                      class="w-16 h-4 text-center text-[11px] font-bold font-mono bg-transparent border-none outline-none focus:ring-0 p-0 text-primary" />
                    <span class="text-[10px] text-foreground/40 font-bold">字</span>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </template>
    </div>
  </section>
</template>

<style scoped>
:deep(*) {
  text-rendering: optimizeLegibility;
}
:deep(label) {
  text-rendering: optimizeLegibility;
  letter-spacing: 0.01em;
}
</style>

