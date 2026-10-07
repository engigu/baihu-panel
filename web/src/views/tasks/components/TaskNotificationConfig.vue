<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Checkbox } from '@/components/ui/checkbox'
import { Switch } from '@/components/ui/switch'
import { Bell, BellOff, CheckCircle2, XCircle, Clock, FileText } from 'lucide-vue-next'
import { api, type NotifyChannel, type NotifyBinding } from '@/api'

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

const currentChannel = computed(() => {
  if (notifyWayId.value === 'none') return null
  return notifyChannels.value.find(c => c.id === notifyWayId.value) || null
})

const currentChannelName = computed(() => {
  if (notifyWayId.value === 'none') return '不启用通知'
  return currentChannel.value ? currentChannel.value.name : '已配置渠道'
})

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
      <!-- 通知渠道 (通栏左文右控卡片) -->
      <div class="grid grid-cols-1 sm:grid-cols-4 items-start gap-3">
        <Label class="sm:text-right text-xs text-foreground/70 uppercase tracking-wider font-bold pt-2.5">通知渠道</Label>
        <div class="sm:col-span-3">
          <div class="p-3 rounded-xl bg-muted/15 border border-muted-foreground/10 flex flex-col sm:flex-row sm:items-center justify-between gap-3">
            <div class="space-y-0.5">
              <div class="flex items-center gap-2">
                <Bell v-if="notifyWayId !== 'none'" class="h-4 w-4 text-primary shrink-0" />
                <BellOff v-else class="h-4 w-4 text-muted-foreground shrink-0" />
                <span class="text-xs font-bold text-foreground">
                  {{ currentChannelName }}
                </span>
                <span class="text-[10px] px-1.5 py-0.5 rounded font-medium" :class="notifyWayId !== 'none' ? 'bg-primary/10 text-primary' : 'bg-muted text-muted-foreground'">
                  {{ notifyWayId !== 'none' ? '已配置推送' : '未启用' }}
                </span>
              </div>
              <p class="text-[11px] text-muted-foreground">
                {{ notifyWayId !== 'none' ? '任务运行状态变化时，将向该渠道发送推送通知' : '当前不向外部推送任何执行结果通知' }}
              </p>
            </div>
            <div class="self-start sm:self-auto shrink-0">
              <Select v-model="notifyWayId">
                <SelectTrigger class="w-48 h-8 bg-background border-muted-foreground/15 px-2.5 text-xs">
                  <SelectValue placeholder="选择通知渠道..." class="truncate">
                    <div class="flex items-center gap-1.5 truncate">
                      <div class="w-1.5 h-1.5 rounded-full" :class="notifyWayId !== 'none' ? 'bg-primary' : 'bg-muted-foreground/40'" />
                      <span>{{ currentChannelName }}</span>
                    </div>
                  </SelectValue>
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="none" class="text-xs">
                    <div class="flex items-center gap-2">
                      <div class="w-1.5 h-1.5 rounded-full bg-muted-foreground/40" />
                      <span>不启用通知</span>
                    </div>
                  </SelectItem>
                  <SelectItem v-for="ch in notifyChannels" :key="ch.id" :value="ch.id" class="text-xs">
                    <div class="flex items-center gap-2">
                      <div class="w-1.5 h-1.5 rounded-full" :class="ch.enabled ? 'bg-emerald-500' : 'bg-muted-foreground'" />
                      <span class="font-medium">{{ ch.name }}</span>
                    </div>
                  </SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>
        </div>
      </div>

      <template v-if="notifyWayId !== 'none'">
        <!-- 通知时机 -->
        <div class="grid grid-cols-1 sm:grid-cols-4 items-start gap-3">
          <Label class="sm:text-right text-xs text-foreground/70 uppercase tracking-wider font-bold pt-2.5">通知时机</Label>
          <div class="sm:col-span-3 space-y-2.5">
            <div class="grid grid-cols-1 sm:grid-cols-3 gap-2.5">
              <!-- 成功时 -->
              <div
                class="p-2.5 rounded-xl border cursor-pointer transition-all flex items-center justify-between select-none"
                :class="notifyOnSuccess
                  ? 'border-emerald-500/40 bg-emerald-500/10 text-emerald-700 dark:text-emerald-300 shadow-xs'
                  : 'border-muted-foreground/15 bg-muted/10 hover:bg-muted/20 text-muted-foreground'"
                @click="notifyOnSuccess = !notifyOnSuccess"
              >
                <div class="flex items-center gap-2 min-w-0">
                  <CheckCircle2 class="h-4 w-4 shrink-0" :class="notifyOnSuccess ? 'text-emerald-500' : 'text-muted-foreground/60'" />
                  <span class="text-xs font-semibold">执行成功</span>
                </div>
                <Checkbox :checked="notifyOnSuccess" class="pointer-events-none data-[state=checked]:bg-emerald-500 data-[state=checked]:border-emerald-500" />
              </div>

              <!-- 失败时 -->
              <div
                class="p-2.5 rounded-xl border cursor-pointer transition-all flex items-center justify-between select-none"
                :class="notifyOnFailure
                  ? 'border-rose-500/40 bg-rose-500/10 text-rose-700 dark:text-rose-300 shadow-xs'
                  : 'border-muted-foreground/15 bg-muted/10 hover:bg-muted/20 text-muted-foreground'"
                @click="notifyOnFailure = !notifyOnFailure"
              >
                <div class="flex items-center gap-2 min-w-0">
                  <XCircle class="h-4 w-4 shrink-0" :class="notifyOnFailure ? 'text-rose-500' : 'text-muted-foreground/60'" />
                  <span class="text-xs font-semibold">执行失败</span>
                </div>
                <Checkbox :checked="notifyOnFailure" class="pointer-events-none data-[state=checked]:bg-rose-500 data-[state=checked]:border-rose-500" />
              </div>

              <!-- 超时时 -->
              <div
                class="p-2.5 rounded-xl border cursor-pointer transition-all flex items-center justify-between select-none"
                :class="notifyOnTimeout
                  ? 'border-amber-500/40 bg-amber-500/10 text-amber-700 dark:text-amber-300 shadow-xs'
                  : 'border-muted-foreground/15 bg-muted/10 hover:bg-muted/20 text-muted-foreground'"
                @click="notifyOnTimeout = !notifyOnTimeout"
              >
                <div class="flex items-center gap-2 min-w-0">
                  <Clock class="h-4 w-4 shrink-0" :class="notifyOnTimeout ? 'text-amber-500' : 'text-muted-foreground/60'" />
                  <span class="text-xs font-semibold">执行超时</span>
                </div>
                <Checkbox :checked="notifyOnTimeout" class="pointer-events-none data-[state=checked]:bg-amber-500 data-[state=checked]:border-amber-500" />
              </div>
            </div>
          </div>
        </div>

        <!-- 消息内容 -->
        <div class="grid grid-cols-1 sm:grid-cols-4 items-start gap-3">
          <Label class="sm:text-right text-xs text-foreground/70 uppercase tracking-wider font-bold pt-2.5">消息内容</Label>
          <div class="sm:col-span-3">
            <div class="p-3 rounded-xl bg-muted/15 border border-muted-foreground/10 space-y-3">
              <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
                <div class="space-y-0.5">
                  <div class="flex items-center gap-2">
                    <FileText class="h-4 w-4 text-muted-foreground shrink-0" />
                    <span class="text-xs font-bold text-foreground">附带运行日志</span>
                    <span class="text-[10px] px-1.5 py-0.5 rounded font-medium" :class="notifyIncludeLog ? 'bg-primary/10 text-primary' : 'bg-muted text-muted-foreground'">
                      {{ notifyIncludeLog ? '已包含' : '仅基础信息' }}
                    </span>
                  </div>
                  <p class="text-[11px] text-muted-foreground">
                    在推送的消息中附加该任务末尾的运行日志输出
                  </p>
                </div>
                <div class="self-start sm:self-auto shrink-0">
                  <Switch v-model="notifyIncludeLog" class="data-[state=checked]:bg-primary" />
                </div>
              </div>

              <!-- 开启日志后的字数设置 -->
              <div v-if="notifyIncludeLog" class="pt-2.5 border-t border-muted-foreground/10 flex items-center justify-between animate-in fade-in slide-in-from-top-1 duration-200">
                <span class="text-xs text-muted-foreground">日志截取长度限制</span>
                <div class="flex items-center gap-1.5">
                  <span class="text-xs text-muted-foreground font-medium">保留末尾</span>
                  <input
                    type="text"
                    inputmode="numeric"
                    :value="notifyLogLimit"
                    @input="(e: any) => notifyLogLimit = Number(e.target.value.replace(/\D/g, ''))"
                    class="w-20 h-8 text-xs font-semibold text-center bg-background rounded-md border border-muted-foreground/20 focus:outline-none focus:ring-1 focus:ring-primary"
                  />
                  <span class="text-xs text-muted-foreground font-medium">字</span>
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

