<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Button } from '@/components/ui/button'
import { Textarea } from '@/components/ui/textarea'
import { Badge } from '@/components/ui/badge'
import { api } from '@/api'
import { toast } from 'vue-sonner'
import { Save, RefreshCcw, Info, ChevronDown, ChevronUp, Shield, Terminal, Zap, Search, FileText } from 'lucide-vue-next'

const props = defineProps<{
  activeTab?: string,
}>()

const emit = defineEmits(['update:activeTab'])

const loading = ref(false)
const saving = ref(false)

const prefix = ref('')
const templates = ref<Record<string, string>>({})
const expandedEvents = ref<Record<string, boolean>>({})
const showHelp = ref(false)

const eventGroups = [
  {
    title: '系统与安全事件',
    description: '配置帐号登录、安全告警等系统级事件的通知推送',
    events: [
      {
        id: 'user_login',
        name: '用户登录 (成功/失败)',
        keys: { title: 'notify_template_user_login_title', text: 'notify_template_user_login_text' },
        variables: ['username', 'ip', 'status_label', 'message']
      },
      {
        id: 'brute_force_login',
        name: '密码尝试破解',
        keys: { title: 'notify_template_brute_force_login_title', text: 'notify_template_brute_force_login_text' },
        variables: ['ip', 'username']
      },
      {
        id: 'password_changed',
        name: '密码修改',
        keys: { title: 'notify_template_password_changed_title', text: 'notify_template_password_changed_text' },
        variables: ['username']
      }
    ]
  },
  {
    title: '任务执行事件',
    description: '配置定时任务执行状态的通知行为（支持全局模板定制）',
    events: [
      {
        id: 'task_success',
        name: '任务成功',
        keys: { title: 'notify_template_task_success_title', text: 'notify_template_task_success_text' },
        variables: ['task_id', 'task_name', 'start_time', 'duration', 'output']
      },
      {
        id: 'task_failed',
        name: '任务失败',
        keys: { title: 'notify_template_task_failed_title', text: 'notify_template_task_failed_text' },
        variables: ['task_id', 'task_name', 'start_time', 'duration', 'error', 'output']
      },
      {
        id: 'task_timeout',
        name: '任务超时',
        keys: { title: 'notify_template_task_timeout_title', text: 'notify_template_task_timeout_text' },
        variables: ['task_id', 'task_name', 'start_time', 'duration', 'output']
      }
    ]
  }
]

async function loadSettings() {
  loading.value = true
  try {
    const res = await api.settings.getSection('notify')
    prefix.value = res.notify_prefix || '[白虎面板]'
    templates.value = res
  } catch (e: any) {
    toast.error('加载配置失败: ' + e.message)
  } finally {
    loading.value = false
  }
}

async function saveSettings() {
  saving.value = true
  try {
    const data: Record<string, string> = {
      notify_prefix: prefix.value,
      ...templates.value
    }
    await api.settings.setSection('notify', data)
    toast.success('模板配置已保存')
  } catch (e: any) {
    toast.error('保存失败: ' + e.message)
  } finally {
    saving.value = false
  }
}

function insertVariable(eventKey: string, variable: string) {
  const input = document.querySelector(`[data-event-key="${eventKey}"]`) as HTMLTextAreaElement | HTMLInputElement
  const tag = `{{${variable}}}`
  
  if (input) {
    const start = input.selectionStart || 0
    const end = input.selectionEnd || 0
    const val = templates.value[eventKey] || ''
    templates.value[eventKey] = val.substring(0, start) + tag + val.substring(end)
    
    // 重新聚焦并设置光标
    setTimeout(() => {
      input.focus()
      input.setSelectionRange(start + tag.length, start + tag.length)
    }, 0)
  } else {
    const current = templates.value[eventKey] || ''
    templates.value[eventKey] = current + tag
  }
}

function toggleExpand(id: string) {
  expandedEvents.value[id] = !expandedEvents.value[id]
}



onMounted(() => {
  loadSettings()

})
</script>

<template>
  <div class="space-y-6">
    <!-- 外层主卡片 -->
    <Card>
      <CardHeader class="border-b bg-muted/10 !pb-0">
        <div class="flex items-start justify-between gap-4">
          <div class="flex-1 min-w-0">
            <CardTitle class="font-bold tracking-tight">通知模板</CardTitle>
            <CardDescription class="text-xs sm:text-sm mt-1 text-muted-foreground/80">
              定制消息格式，支持全局前缀与动态变量内置变量
            </CardDescription>
          </div>
          <div class="flex items-center gap-1.5 sm:gap-2 shrink-0">
            <Button variant="outline" size="sm" @click="loadSettings" :disabled="loading" class="h-8 sm:h-9 px-2 sm:px-3 text-xs sm:text-sm">
              <RefreshCcw class="w-3.5 h-3.5 sm:mr-2" :class="{ 'animate-spin': loading }" />
              <span class="hidden sm:inline">刷新</span>
            </Button>
            <Button size="sm" @click="saveSettings" :disabled="saving" class="h-8 sm:h-9 px-3 sm:px-4 text-xs sm:text-sm">
              <Save class="w-3.5 h-3.5 sm:mr-2" />
              <span class="hidden sm:inline">{{ saving ? '保存中...' : '提交修改' }}</span>
              <span class="sm:hidden">提交</span>
            </Button>
          </div>
        </div>
      </CardHeader>

      <CardContent class="p-3 sm:px-5 sm:py-4 space-y-4">
        <!-- 全局前缀设置 (内嵌卡片) -->
        <Card class="bg-muted/5 border-dashed shadow-sm">
          <CardHeader class="pb-2 px-4 pt-3">
            <div class="flex items-center gap-2">
              <div class="w-1.5 h-4 bg-primary rounded-full" />
              <CardTitle class="text-base font-bold">全局消息前缀</CardTitle>
            </div>
            <CardDescription class="text-[11px] sm:text-xs text-muted-foreground/80">该前缀会添加在所有通知标题的最前面，用于快速识别消息来源</CardDescription>
          </CardHeader>
          <CardContent class="px-4 pb-3 space-y-3">
            <div class="flex flex-col sm:flex-row gap-4 items-start sm:items-center">
              <div class="flex-1 w-full">
                <Label class="text-[10px] font-bold text-muted-foreground uppercase tracking-wider ml-1 mb-1.5 block">前缀文本</Label>
                <Input v-model="prefix" placeholder="例如: [生产环境]" 
                  class="h-9 text-sm bg-background border-muted-foreground/20 focus:border-primary/50" />
              </div>
              <div class="flex items-center gap-3 px-4 h-9 rounded-lg bg-muted/40 border border-dashed border-muted-foreground/20 text-[11px] text-muted-foreground shrink-0 w-full sm:w-auto mt-4 sm:mt-5">
                预览效果: <span class="font-mono text-primary font-bold tracking-tight">{{ prefix }} 用户登录成功</span>
              </div>
            </div>
          </CardContent>
        </Card>
        
        <!-- v2 语法指引 -->
        <div class="space-y-4">
          <div class="flex items-center justify-between px-1">
            <div class="flex items-center gap-2">
              <FileText class="w-3.5 h-3.5 text-primary" />
              <span class="text-[11px] font-bold text-foreground/70 uppercase tracking-widest">模板逻辑定制 (v2)</span>
            </div>
            <button @click="showHelp = !showHelp" class="flex items-center gap-1 text-[10px] text-primary/70 hover:text-primary transition-colors font-medium">
              <Zap class="w-3 h-3" />
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
                  {{ if eq .status_label "成功" }} 🟢 {{ else }} 🔴 {{ end }}
                </div>
              </div>
              <div class="space-y-2">
                <h4 class="text-[11px] font-bold text-primary flex items-center gap-1.5">
                  <Search class="w-3 h-3" /> 比较与判断
                </h4>
                <p class="text-[10px] text-muted-foreground leading-relaxed">
                  你可以通过 <code class="bg-primary/10 px-1 rounded text-primary">contains</code> 判断日志是否包含特定内容。
                </p>
                <div v-pre class="bg-background/60 p-2 rounded border border-primary/10 font-mono text-[9px] text-foreground/80">
                  {{ if contains .output "Error" }} ⚠️ 日志中发现错误! {{ end }}
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- 模板详情分组列表 -->
        <div class="space-y-5">
          <div v-for="group in eventGroups" :key="group.title" class="space-y-4">
            <!-- 分组标题 (参考事件绑定样式) -->
            <div class="flex items-start gap-2.5 px-1">
              <component :is="group.title === '任务执行事件' ? Terminal : Shield" class="w-5 h-5 text-primary shrink-0 mt-0.5" />
              <div>
                <h3 class="text-sm font-bold text-foreground leading-none">{{ group.title }}</h3>
                <p class="text-[11px] text-muted-foreground mt-1.5">{{ group.description }}</p>
              </div>
            </div>

            <!-- 事件网格 -->
            <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div v-for="event in group.events" :key="event.id" 
                class="flex flex-col p-4 rounded-xl border bg-background hover:border-primary/20 transition-all duration-200 shadow-sm"
                :class="{ 'ring-1 ring-primary/20 bg-accent/5': expandedEvents[event.id] }">
                
                <!-- 卡片头部 (可点击折叠) -->
                <div class="flex items-center justify-between cursor-pointer select-none" 
                    :class="{ 'mb-4 border-b pb-3 border-dashed': expandedEvents[event.id] }"
                    @click="toggleExpand(event.id)">
                  <div class="flex items-center gap-2.5">
                    <div class="w-7 h-7 rounded-lg bg-primary/5 flex items-center justify-center text-primary">
                      <Info class="w-3.5 h-3.5" />
                    </div>
                    <div class="flex flex-col">
                      <span class="text-sm font-bold">{{ event.name }}</span>
                      <span class="text-[10px] text-muted-foreground font-mono opacity-50">ID: {{ event.id }}</span>
                    </div>
                  </div>
                  <component :is="expandedEvents[event.id] ? ChevronUp : ChevronDown" class="w-4 h-4 text-muted-foreground" />
                </div>

                <!-- 展开后的编辑表单 -->
                <div v-if="expandedEvents[event.id]" class="space-y-4 animate-in fade-in slide-in-from-top-1 duration-200">
                  <!-- 可用参数 -->
                  <div class="flex flex-col gap-1.5 p-2.5 rounded-lg bg-muted/30 border border-dashed">
                    <span class="text-[9px] font-bold text-muted-foreground uppercase tracking-wider">可用参数 (点击插入):</span>
                    <div class="flex flex-wrap gap-1.5">
                      <Badge v-for="v in event.variables" :key="v" variant="secondary" 
                        class="cursor-pointer hover:bg-primary/20 hover:text-primary transition-colors py-0 px-2 font-mono text-[10px] border-none bg-muted-foreground/10"
                        @click="insertVariable(event.keys.text, v)"
                        v-text="'{{' + v + '}}'" />
                    </div>
                  </div>

                  <!-- 模板内容输入 -->
                  <div class="space-y-4">
                    <div class="space-y-1.5">
                      <Label class="text-[10px] font-bold text-muted-foreground uppercase tracking-wider ml-1">推送标题模板</Label>
                      <Input v-model="templates[event.keys.title]" placeholder="通知标题" 
                        :data-event-key="event.keys.title"
                        class="h-9 text-sm bg-background border-muted-foreground/20 focus:border-primary/50" />
                    </div>
                    <div class="space-y-1.5">
                      <Label class="text-[10px] font-bold text-muted-foreground uppercase tracking-wider ml-1">推送正文模板</Label>
                      <Textarea v-model="templates[event.keys.text]" 
                        :rows="4"
                        :data-event-key="event.keys.text"
                        placeholder="通知详细内容..." 
                        class="resize-none font-sans text-sm leading-relaxed bg-background border-muted-foreground/20 focus:border-primary/50" />
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </CardContent>
    </Card>
  </div>
</template>

<style scoped>
/* 移除不必要的移动端过重样式，保持清爽 */
</style>
