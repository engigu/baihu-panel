<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Checkbox } from '@/components/ui/checkbox'
import { Switch } from '@/components/ui/switch'
import { Bell } from 'lucide-vue-next'
import { api, type NotifyChannel, type NotifyBinding } from '@/api'
import { cn } from '@/lib/utils'
import { TEMPLATE_TAGS } from '@/utils/template'
import { ChevronDown, Info, Type, FileText, Zap, Search } from 'lucide-vue-next'

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

// 模板配置 (v2)
const templates = ref<Record<string, { title: string; content: string }>>({
  task_success: { title: '任务 [{{task_name}}] 成功', content: '任务 #{{task_id}} {{task_name}}\n状态: 成功\n执行时间: {{start_time}}\n耗时: {{duration}}ms' },
  task_failed: { title: '任务 [{{task_name}}] 失败', content: '任务 #{{task_id}} {{task_name}}\n状态: 失败\n错误原因: {{output}}' },
  task_timeout: { title: '任务 [{{task_name}}] 超时', content: '任务 #{{task_id}} {{task_name}}\n状态: 超时\n耗时: {{duration}}ms' }
})

// 控制折叠状态
const activeEvent = ref<string | null>(null)
const showHelp = ref(false)

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
          
          // 加载 v2 模板
          if (extra.version === 'v2' && extra.templates) {
            templates.value = { ...templates.value, ...extra.templates }
          }
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
        log_limit: notifyLogLimit.value,
        version: 'v2',
        templates: templates.value
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

function insertTag(event: string, field: 'title' | 'content', tag: string) {
  const key = `${event}_${field}`
  const input = document.querySelector(`[data-event-key="${key}"]`) as HTMLTextAreaElement | HTMLInputElement
  const tagStr = `{{${tag}}}`
  
  if (input) {
    const start = input.selectionStart || 0
    const end = input.selectionEnd || 0
    const val = field === 'title' ? templates.value[event].title : templates.value[event].content
    const newVal = val.substring(0, start) + tagStr + val.substring(end)
    
    if (field === 'title') {
      templates.value[event].title = newVal
    } else {
      templates.value[event].content = newVal
    }
    
    // 重新聚焦并设置光标
    setTimeout(() => {
      input.focus()
      input.setSelectionRange(start + tagStr.length, start + tagStr.length)
    }, 0)
  } else {
    if (field === 'title') {
      templates.value[event].title += tagStr
    } else {
      templates.value[event].content += tagStr
    }
  }
}

function toggleEvent(key: string) {
  activeEvent.value = activeEvent.value === key ? null : key
}

function getEventLabel(key: string) {
  const map: Record<string, string> = {
    task_success: '任务成功',
    task_failed: '任务失败',
    task_timeout: '任务超时'
  }
  return map[key] || key
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
                <Checkbox :id="`ns-${taskId || 'new'}`" v-model="notifyOnSuccess" class="border-muted-foreground/30 data-[state=checked]:bg-primary data-[state=checked]:border-primary" />
                <label :for="`ns-${taskId || 'new'}`" class="text-xs font-medium shrink-0 cursor-pointer group-hover:text-primary transition-colors text-foreground/80">成功时</label>
              </div>
              <div class="flex items-center gap-2 group">
                <Checkbox :id="`nf-${taskId || 'new'}`" v-model="notifyOnFailure" class="border-muted-foreground/30 data-[state=checked]:bg-primary data-[state=checked]:border-primary" />
                <label :for="`nf-${taskId || 'new'}`" class="text-xs font-medium shrink-0 cursor-pointer group-hover:text-primary transition-colors text-foreground/80">失败时</label>
              </div>
              <div class="flex items-center gap-2 group">
                <Checkbox :id="`nt-${taskId || 'new'}`" v-model="notifyOnTimeout" class="border-muted-foreground/30 data-[state=checked]:bg-primary data-[state=checked]:border-primary" />
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

            <!-- 模板编辑器 (v2) -->
            <div class="space-y-4 mt-6">
              <div class="flex items-center justify-between px-1">
                <div class="flex items-center gap-2">
                  <FileText class="w-3.5 h-3.5 text-primary" />
                  <span class="text-[11px] font-bold text-foreground/70 uppercase tracking-widest">推送模板定制 (v2)</span>
                </div>
                <button @click="showHelp = !showHelp" class="flex items-center gap-1 text-[10px] text-primary/70 hover:text-primary transition-colors font-medium">
                  <Info class="w-3 h-3" />
                  {{ showHelp ? '隐藏指引' : '高级语法指引' }}
                </button>
              </div>

              <!-- 语法指引面板 -->
              <div v-if="showHelp" class="p-4 rounded-xl bg-primary/5 border border-primary/20 space-y-3 animate-in zoom-in-95 duration-200">
                <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
                  <div class="space-y-2">
                    <h4 class="text-[11px] font-bold text-primary flex items-center gap-1.5">
                      <Zap class="w-3 h-3" /> 条件分支 (If/Else)
                    </h4>
                    <p class="text-[10px] text-muted-foreground leading-relaxed">
                      支持根据变量状态切换内容。注意：逻辑判断内部变量需带点前缀 <code class="bg-primary/10 px-1 rounded text-primary">.</code>
                    </p>
                    <div v-pre class="bg-background/60 p-2 rounded border border-primary/10 font-mono text-[9px] text-foreground/80">
                      {{ if eq .status "success" }} ✅ {{ else }} ❌ {{ end }}
                    </div>
                  </div>
                  <div class="space-y-2">
                    <h4 class="text-[11px] font-bold text-primary flex items-center gap-1.5">
                      <Search class="w-3 h-3" /> 关键字过滤 (Contains)
                    </h4>
                    <p class="text-[10px] text-muted-foreground leading-relaxed">
                      检测日志输出中是否包含特定字符。常用于异常告警。
                    </p>
                    <div v-pre class="bg-background/60 p-2 rounded border border-primary/10 font-mono text-[9px] text-foreground/80">
                      {{ if contains .output "Error" }} 🚨 发现报错! {{ end }}
                    </div>
                  </div>
                </div>
                <div class="pt-2 border-t border-primary/10">
                  <p class="text-[10px] text-muted-foreground italic">
                    提示：在“正文模板”中你可以自由编排。例如：只需在成功时发简报，失败时才发详细日志。
                  </p>
                </div>
              </div>
              
              <div v-for="event in ['task_success', 'task_failed', 'task_timeout']" :key="event" 
                :class="cn('border rounded-xl transition-all overflow-hidden', 
                   activeEvent === event ? 'border-primary/30 ring-1 ring-primary/10 shadow-sm' : 'border-muted-foreground/10 bg-muted/5')">
                
                <!-- Header -->
                <div @click="toggleEvent(event)" class="flex items-center justify-between px-4 py-3 cursor-pointer hover:bg-muted/30 transition-colors">
                  <div class="flex items-center gap-3">
                    <div :class="cn('h-1.5 w-1.5 rounded-full', 
                      event === 'task_success' ? 'bg-green-500' : (event === 'task_failed' ? 'bg-destructive' : 'bg-amber-500'))" />
                    <span class="text-xs font-bold">{{ getEventLabel(event) }}</span>
                    <span class="text-[10px] text-muted-foreground font-mono opacity-60">ID: {{ event }}</span>
                  </div>
                  <ChevronDown :class="cn('w-4 h-4 text-muted-foreground transition-transform duration-300', activeEvent === event && 'rotate-180')" />
                </div>

                <!-- Body -->
                <div v-if="activeEvent === event" class="p-4 pt-0 space-y-4 animate-in slide-in-from-top-2 duration-300">
                  <!-- Tag Bar -->
                  <div class="p-2.5 rounded-lg bg-background/50 border border-dashed border-muted-foreground/20">
                    <div class="flex items-center gap-1.5 mb-2 px-1">
                      <Info class="w-3 h-3 text-primary" />
                      <span class="text-[10px] text-muted-foreground font-medium">可用参数 (点击插入):</span>
                    </div>
                    <div class="flex flex-wrap gap-1.5">
                      <button v-for="tag in TEMPLATE_TAGS" :key="tag.value"
                        @click="insertTag(event, 'content', tag.label)"
                        class="px-2 py-1 rounded-md bg-muted hover:bg-primary/10 hover:text-primary text-[10px] font-mono transition-all border border-transparent hover:border-primary/20">
                        {{ tag.label }}
                      </button>
                    </div>
                  </div>

                  <!-- Title Input -->
                  <div class="space-y-1.5">
                    <div class="flex items-center gap-2 px-1">
                      <Type class="w-3 h-3 text-muted-foreground" />
                      <label class="text-[10px] font-bold text-muted-foreground uppercase">推送标题模板</label>
                    </div>
                    <input v-model="templates[event].title" :data-event-key="event + '_title'" class="w-full h-9 px-3 bg-background border border-muted-foreground/20 rounded-md text-xs focus:ring-1 focus:ring-primary/30 outline-none transition-all" />
                  </div>

                  <!-- Content Input -->
                  <div class="space-y-1.5">
                    <div class="flex items-center gap-2 px-1">
                      <FileText class="w-3 h-3 text-muted-foreground" />
                      <label class="text-[10px] font-bold text-muted-foreground uppercase">推送正文模板</label>
                    </div>
                    <textarea v-model="templates[event].content" :data-event-key="event + '_content'" rows="4" 
                      class="w-full p-3 bg-background border border-muted-foreground/20 rounded-md text-xs focus:ring-1 focus:ring-primary/30 outline-none transition-all resize-none font-mono" />
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

