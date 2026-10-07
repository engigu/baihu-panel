<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import { ScrollArea } from '@/components/ui/scroll-area'
import { FolderSync, Plus, X, Filter, RotateCcw, Trash2, Info, Zap, Clock, Sparkles } from 'lucide-vue-next'
import { api, type Task, type Agent } from '@/api'
import { TASK_TYPE, TRIGGER_TYPE } from '@/constants'
import { toast } from 'vue-sonner'
import { formatDateTime } from '@/utils/date'

import TaskNotificationConfig from './components/TaskNotificationConfig.vue'
import TaskAdvancedConfig from './components/TaskAdvancedConfig.vue'
import TaskCronConfig from './components/TaskCronConfig.vue'
import TaskTagsConfig from './components/TaskTagsConfig.vue'

interface Props {
  open: boolean;
  task?: Partial<Task>;
  isEdit: boolean;
}
const props = defineProps<Props>();

const emit = defineEmits<{
  (e: 'update:open', value: boolean): void;
  (e: 'saved'): void;
}>();

// 默认提供的 gitignore 过滤目录与文件
const DEFAULT_IGNORE_RULES = [
  '.git/',
  'node_modules/',
  '__pycache__/',
  '.idea/',
  '.vscode/',
  '*.log',
  '.DS_Store'
]

// 常用推荐预设
const SUGGESTED_PRESETS = [
  '.git/',
  'node_modules/',
  '__pycache__/',
  '.idea/',
  '.vscode/',
  '*.log',
  '.DS_Store',
  'dist/',
  'build/',
  'temp/',
  '*.tmp'
]

const form = ref<Partial<Task>>({})
const allAgents = ref<Agent[]>([])
const selectedAgentId = ref<string>('')
const cleanTargetEnabled = ref(false)
const syncMode = ref<'hybrid' | 'realtime' | 'cron'>('hybrid')
const debounceDelay = ref<number>(8)
const syncOnOnline = ref<boolean>(true)
const dirMappings = ref<{ id: string; source_path: string; target_path: string; remark?: string }[]>([])
const ignoreRules = ref<string[]>([...DEFAULT_IGNORE_RULES])
const newRuleInput = ref('')

const notificationConfigRef = ref<InstanceType<typeof TaskNotificationConfig> | null>(null)

const onlineAgents = computed(() => {
  return allAgents.value.filter((a: Agent) => a.enabled)
})

function addDirMapping() {
  dirMappings.value.push({
    id: String(Date.now()),
    source_path: '',
    target_path: '',
    remark: ''
  })
}

function removeDirMapping(index: number) {
  dirMappings.value.splice(index, 1)
}

function addIgnoreRule(rule?: string) {
  const target = (rule || newRuleInput.value).trim()
  if (!target) return
  if (ignoreRules.value.includes(target)) {
    toast.info(`规则 "${target}" 已存在`)
    newRuleInput.value = ''
    return
  }
  ignoreRules.value.push(target)
  newRuleInput.value = ''
}

function removeIgnoreRule(index: number) {
  ignoreRules.value.splice(index, 1)
}

function resetDefaultRules() {
  ignoreRules.value = [...DEFAULT_IGNORE_RULES]
  toast.success('已恢复为默认过滤规则')
}

function clearAllRules() {
  ignoreRules.value = []
}

watch(() => props.open, async (val: boolean) => {
  if (val) {
    await loadAgents()

    form.value = {
      retry_count: props.task?.retry_count ?? 0,
      retry_interval: props.task?.retry_interval ?? 0,
      random_range: props.task?.random_range ?? 0,
      timeout: props.task?.timeout ?? 30,
      pin_type: props.task?.pin_type ?? 'none',
      schedule: props.task?.schedule || '0 0 4 * * *',
      ...props.task
    }

    // 解析已有的同步配置
    try {
      const configStr = props.task?.unified_config || '{}'
      const parsed = JSON.parse(configStr)
      const syncConfig = parsed?.agent_sync_script
      if (syncConfig) {
        cleanTargetEnabled.value = !!syncConfig.clean_target
        syncMode.value = syncConfig.sync_mode || 'hybrid'
        debounceDelay.value = syncConfig.debounce_delay ?? 8
        syncOnOnline.value = syncConfig.sync_on_online ?? true
        dirMappings.value = Array.isArray(syncConfig.dir_mappings)
          ? JSON.parse(JSON.stringify(syncConfig.dir_mappings))
          : []
        if (Array.isArray(syncConfig.ignore_rules)) {
          ignoreRules.value = [...syncConfig.ignore_rules]
        } else {
          ignoreRules.value = [...DEFAULT_IGNORE_RULES]
        }
      } else {
        cleanTargetEnabled.value = false
        syncMode.value = 'hybrid'
        debounceDelay.value = 8
        syncOnOnline.value = true
        dirMappings.value = []
        ignoreRules.value = [...DEFAULT_IGNORE_RULES]
      }
    } catch {
      cleanTargetEnabled.value = false
      syncMode.value = 'hybrid'
      debounceDelay.value = 8
      syncOnOnline.value = true
      dirMappings.value = []
      ignoreRules.value = [...DEFAULT_IGNORE_RULES]
    }

    if (dirMappings.value.length === 0) {
      addDirMapping()
    }

    // 解析目标 Agent
    if (props.task?.agent_id) {
      selectedAgentId.value = String(props.task.agent_id)
    } else if (onlineAgents.value.length > 0 && onlineAgents.value[0]?.id) {
      selectedAgentId.value = String(onlineAgents.value[0].id)
    } else {
      selectedAgentId.value = ''
    }

    await notificationConfigRef.value?.loadConfig(props.isEdit ? props.task?.id : undefined)
  }
})

async function loadAgents() {
  try {
    allAgents.value = await api.agents.list()
  } catch {
    allAgents.value = []
  }
}

async function save() {
  try {
    if (!form.value.name?.trim()) {
      toast.error('请输入同步任务名称')
      return
    }
    if (!selectedAgentId.value || selectedAgentId.value === 'local') {
      toast.error('请选择一个目标 Agent 节点')
      return
    }
    if (dirMappings.value.length === 0) {
      toast.error('请至少添加一个目录或文件映射')
      return
    }
    for (let i = 0; i < dirMappings.value.length; i++) {
      const m = dirMappings.value[i]
      if (!m || !m.source_path?.trim() || !m.target_path?.trim()) {
        toast.error(`第 ${i + 1} 个映射条目的源路径和目标路径均不能为空`)
        return
      }
    }

    form.value.type = TASK_TYPE.AGENT_SYNC_SCRIPT
    form.value.trigger_type = TRIGGER_TYPE.CRON
    form.value.agent_id = selectedAgentId.value

    const cmdParts = ['baihu', 'agentsync']
    if (selectedAgentId.value) {
      cmdParts.push('--agent', selectedAgentId.value)
    }
    for (const m of dirMappings.value) {
      const src = m.source_path?.trim()
      const dst = m.target_path?.trim()
      if (src && dst) {
        cmdParts.push('--mapping', `"${src}:${dst}"`)
      }
    }
    if (cleanTargetEnabled.value) {
      cmdParts.push('--clean-target')
    }
    const cleanIgnores = ignoreRules.value.map(r => r.trim()).filter(Boolean)
    if (cleanIgnores.length > 0) {
      cmdParts.push('--ignore', `"${cleanIgnores.join(',')}"`)
    }
    form.value.command = cmdParts.join(' ')

    let config: Record<string, any> = {}
    const configStr = props.task?.unified_config || form.value.unified_config
    if (configStr) {
      try {
        const parsed = JSON.parse(configStr)
        if (parsed && typeof parsed === 'object') {
          config = parsed
        }
      } catch {}
    }

    config.agent_sync_script = {
      agent_id: selectedAgentId.value,
      clean_target: cleanTargetEnabled.value,
      sync_mode: syncMode.value,
      debounce_delay: Math.max(1, Number(debounceDelay.value) || 8),
      sync_on_online: syncOnOnline.value,
      ignore_rules: ignoreRules.value.map(r => r.trim()).filter(Boolean),
      dir_mappings: dirMappings.value.map(m => ({
        source_type: 'dir',
        source_path: m.source_path.trim(),
        target_path: m.target_path.trim(),
        remark: m.remark ? m.remark.trim() : ''
      }))
    }

    form.value.unified_config = JSON.stringify(config)

    if (props.isEdit && form.value.id) {
      const task = await api.tasks.update(form.value.id, form.value)
      await notificationConfigRef.value?.saveConfig(task.id)
      toast.success('Agent 脚本同步任务已更新')
    } else {
      const task = await api.tasks.create(form.value)
      await notificationConfigRef.value?.saveConfig(task.id)
      toast.success('Agent 脚本同步任务已创建')
    }

    emit('update:open', false)
    emit('saved')
  } catch (error: any) {
    toast.error('保存失败', {
      description: error.message || '未知错误'
    })
  }
}
</script>

<template>
  <Dialog :open="open" @update:open="emit('update:open', $event)">
    <DialogContent class="max-w-[95vw] sm:max-w-[650px] xl:max-w-[850px] p-0 overflow-hidden border-none bg-background shadow-2xl transition-all duration-300" style="text-rendering: optimizeLegibility;" @openAutoFocus.prevent @pointerDownOutside.prevent @interactOutside.prevent>
      <div class="flex flex-col max-h-[85vh]">
        <DialogHeader class="px-6 pr-12 pt-6 pb-2 shrink-0 border-b border-muted/50">
          <DialogTitle class="text-xl font-bold py-2 flex items-center gap-2">
            <FolderSync class="h-5 w-5 text-amber-500" />
            <span>{{ isEdit ? '编辑 Agent 脚本同步' : '新建 Agent 脚本同步' }}</span>
          </DialogTitle>
          <DialogDescription class="sr-only">
            通过长连接分发面板本地脚本与目录至远程 Agent 节点，支持 gitignore 规则智能排除与实时监听同步
          </DialogDescription>
        </DialogHeader>

        <ScrollArea class="flex-1 min-h-0 px-6">
          <div class="space-y-8 py-6 pb-10">
            <!-- 基本信息 -->
            <section class="space-y-4">
              <div class="flex items-center gap-2 mb-2">
                <div class="h-4 w-1 bg-amber-500 rounded-full shadow-sm shadow-amber-500/20" />
                <h3 class="text-sm font-bold text-foreground/90">基本信息</h3>
              </div>
              <div class="grid gap-5 pl-3 border-l border-muted">
                <div class="grid grid-cols-1 sm:grid-cols-4 items-center gap-3">
                  <Label class="sm:text-right text-xs text-foreground/70 uppercase tracking-wider font-bold">任务名称</Label>
                  <Input v-model="form.name" placeholder="例如: 同步公共脚本至机房节点" class="sm:col-span-3 h-9 bg-muted/20 border-muted-foreground/15 transition-all focus:bg-background/50 text-sm font-medium" />
                </div>
                <div class="grid grid-cols-1 sm:grid-cols-4 items-center gap-3">
                  <Label class="sm:text-right text-xs text-foreground/70 uppercase tracking-wider font-bold">任务备注</Label>
                  <Input v-model="form.remark" placeholder="输入同步任务功能备注 (可选)" class="sm:col-span-3 h-9 bg-muted/20 border-muted-foreground/15 transition-all focus:bg-background/50 text-sm font-medium" />
                </div>
                <TaskTagsConfig v-model="form.tags" />
                <div class="grid grid-cols-1 sm:grid-cols-4 items-center gap-3">
                  <Label class="sm:text-right text-xs text-foreground/70 uppercase tracking-wider font-bold">目标 Agent</Label>
                  <div class="sm:col-span-3">
                    <Select v-model="selectedAgentId">
                      <SelectTrigger class="h-9 bg-muted/20 border-muted-foreground/15 px-3 text-sm">
                        <SelectValue placeholder="请选择目标 Agent 节点..." />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem v-for="agent in onlineAgents" :key="agent.id" :value="String(agent.id)" class="text-xs sm:text-sm">
                          <div class="flex items-center gap-2">
                            <div class="w-1.5 h-1.5 rounded-full" :class="agent.status === 'online' ? 'bg-green-500' : 'bg-muted-foreground'" />
                            <span class="font-medium">{{ agent.name }}</span>
                            <span class="text-[11px] text-muted-foreground">({{ agent.ip || '未知 IP' }})</span>
                          </div>
                        </SelectItem>
                      </SelectContent>
                    </Select>
                  </div>
                </div>
              </div>
            </section>

            <!-- 映射编排 -->
            <section class="space-y-4">
              <div class="flex items-center justify-between mb-2">
                <div class="flex items-center gap-2">
                  <div class="h-4 w-1 bg-amber-500 rounded-full shadow-sm shadow-amber-500/20" />
                  <h3 class="text-sm font-bold text-foreground/90">路径映射编排</h3>
                </div>
                <div class="flex items-center space-x-2 bg-muted/10 px-3 py-1 rounded-full border border-muted-foreground/10">
                  <Switch :checked="cleanTargetEnabled" @update:checked="cleanTargetEnabled = $event" id="clean-target" class="scale-90" />
                  <Label for="clean-target" class="text-[11px] font-medium cursor-pointer text-muted-foreground">同步前清空目标目录</Label>
                </div>
              </div>

              <div class="pl-3 border-l border-amber-500/20 space-y-3">
                <div class="p-3 rounded-xl bg-amber-500/5 border border-amber-500/10 text-amber-600 dark:text-amber-400 text-[11px] leading-relaxed font-medium">
                  <div class="flex items-center gap-2.5">
                    <FolderSync class="h-4 w-4 shrink-0 text-amber-500" />
                    <p>将面板指定的脚本目录或文件同步至目标 Agent，自动打包传输并在远端落盘。</p>
                  </div>
                </div>

                <!-- 映射列表 -->
                <div class="space-y-2.5">
                  <div v-for="(mapping, idx) in dirMappings" :key="mapping.id || idx" class="p-3 rounded-xl bg-muted/10 border border-muted-foreground/10 space-y-2 relative group">
                    <div class="flex items-center justify-between">
                      <span class="text-[11px] font-bold text-foreground/70 uppercase">映射条目 #{{ idx + 1 }}</span>
                      <Button variant="ghost" size="icon" class="h-6 w-6 text-muted-foreground hover:text-destructive" @click="removeDirMapping(idx)">
                        <X class="h-3.5 w-3.5" />
                      </Button>
                    </div>
                    <div class="grid grid-cols-1 sm:grid-cols-2 gap-2.5">
                      <div class="space-y-1">
                        <Label class="text-[10px] text-muted-foreground">面板源路径 (相对 scripts 目录)</Label>
                        <Input v-model="mapping.source_path" placeholder="例如: apps/jdpro 或 test.js" class="h-8 text-xs font-mono bg-background/50" />
                      </div>
                      <div class="space-y-1">
                        <Label class="text-[10px] text-muted-foreground">Agent 目标路径 (相对 Agent scripts)</Label>
                        <Input v-model="mapping.target_path" placeholder="例如: jdpro 或 scripts/test.js" class="h-8 text-xs font-mono bg-background/50" />
                      </div>
                    </div>
                    <div class="space-y-1">
                      <Label class="text-[10px] text-muted-foreground">条目备注 (可选)</Label>
                      <Input v-model="mapping.remark" placeholder="如: 京东定时执行主脚本库" class="h-8 text-xs bg-background/50" />
                    </div>
                  </div>

                  <Button variant="outline" size="sm" class="w-full h-8 text-xs border-dashed border-amber-500/30 text-amber-600 dark:text-amber-400 hover:bg-amber-500/5" @click="addDirMapping">
                    <Plus class="h-3.5 w-3.5 mr-1" /> 添加目录或文件映射
                  </Button>
                </div>
              </div>
            </section>

            <!-- 过滤规则编排 (Gitignore 规则) -->
            <section class="space-y-4">
              <div class="flex items-center justify-between mb-2">
                <div class="flex items-center gap-2">
                  <div class="h-4 w-1 bg-amber-500 rounded-full shadow-sm shadow-amber-500/20" />
                  <h3 class="text-sm font-bold text-foreground/90">排除过滤规则 (.gitignore 规范)</h3>
                </div>
                <div class="flex items-center gap-2">
                  <Button variant="ghost" size="sm" class="h-7 px-2 text-[11px] text-muted-foreground hover:text-foreground" @click="resetDefaultRules">
                    <RotateCcw class="h-3 w-3 mr-1" /> 恢复默认
                  </Button>
                  <Button v-if="ignoreRules.length > 0" variant="ghost" size="sm" class="h-7 px-2 text-[11px] text-muted-foreground hover:text-destructive" @click="clearAllRules">
                    <Trash2 class="h-3 w-3 mr-1" /> 清空
                  </Button>
                </div>
              </div>

              <div class="pl-3 border-l border-amber-500/20 space-y-3">
                <div class="p-3 rounded-xl bg-muted/20 border border-muted-foreground/10 text-muted-foreground text-[11px] leading-relaxed">
                  <div class="flex items-start gap-2">
                    <Info class="h-4 w-4 shrink-0 text-amber-500 mt-0.5" />
                    <div>
                      支持标准 <code class="px-1 py-0.5 rounded bg-muted font-mono text-foreground font-semibold">.gitignore</code> 语法：
                      <span class="text-foreground/80">以 <code class="font-mono">/</code> 结尾仅匹配目录（跳过子孙递归），支持 <code class="font-mono">*</code> 通配符，支持以 <code class="font-mono">!</code> 取反包含。</span>
                    </div>
                  </div>
                </div>

                <!-- 推荐预设快捷添加 -->
                <div class="space-y-1.5">
                  <div class="text-[11px] text-muted-foreground/80 font-medium">推荐预设 (点击可快速加入)：</div>
                  <div class="flex flex-wrap gap-1.5">
                    <button
                      v-for="preset in SUGGESTED_PRESETS"
                      :key="preset"
                      type="button"
                      class="px-2 py-0.5 rounded text-[11px] font-mono transition-all border"
                      :class="ignoreRules.includes(preset)
                        ? 'bg-amber-500/10 text-amber-600 dark:text-amber-400 border-amber-500/30 opacity-70 cursor-default'
                        : 'bg-background hover:bg-muted text-muted-foreground border-muted-foreground/20 hover:text-foreground cursor-pointer'"
                      :disabled="ignoreRules.includes(preset)"
                      @click="addIgnoreRule(preset)"
                    >
                      + {{ preset }}
                    </button>
                  </div>
                </div>

                <!-- 已启用的过滤规则列表 -->
                <div class="space-y-2">
                  <div class="flex items-center justify-between">
                    <span class="text-[11px] text-muted-foreground/80 font-medium">已生效规则 ({{ ignoreRules.length }} 条)：</span>
                  </div>

                  <div v-if="ignoreRules.length === 0" class="text-xs text-muted-foreground py-3 text-center border border-dashed rounded-xl bg-muted/5">
                    未配置任何排除规则（将同步映射目录下的全部文件）
                  </div>

                  <div v-else class="flex flex-wrap gap-1.5 p-2 rounded-xl bg-muted/10 border border-muted-foreground/10 max-h-40 overflow-y-auto">
                    <span
                      v-for="(rule, idx) in ignoreRules"
                      :key="rule + idx"
                      class="inline-flex items-center gap-1 px-2.5 py-1 rounded-md text-xs font-mono bg-background border border-muted-foreground/20 text-foreground shadow-2xs group"
                    >
                      <Filter class="h-3 w-3 text-amber-500/80 shrink-0" />
                      <span>{{ rule }}</span>
                      <button
                        type="button"
                        class="ml-1 text-muted-foreground hover:text-destructive rounded-full p-0.5 transition-colors"
                        @click="removeIgnoreRule(idx)"
                      >
                        <X class="h-3 w-3" />
                      </button>
                    </span>
                  </div>

                  <!-- 自定义输入规则 -->
                  <div class="flex items-center gap-2 pt-1">
                    <Input
                      v-model="newRuleInput"
                      placeholder="输入自定义 gitignore 规则，如: *.tmp 或 build/ (按回车添加)"
                      class="h-8 text-xs font-mono bg-background/50 flex-1"
                      @keydown.enter.prevent="addIgnoreRule()"
                    />
                    <Button variant="outline" size="sm" class="h-8 text-xs px-3 border-amber-500/30 text-amber-600 dark:text-amber-400 hover:bg-amber-500/5" @click="addIgnoreRule()">
                      <Plus class="h-3.5 w-3.5 mr-1" /> 添加规则
                    </Button>
                  </div>
                </div>
              </div>
            </section>

            <!-- 调度与同步策略 -->
            <section class="space-y-4">
              <div class="flex items-center gap-2 mb-2">
                <div class="h-4 w-1 bg-amber-500 rounded-full shadow-sm shadow-amber-500/20" />
                <h3 class="text-sm font-bold text-foreground/90">调度与同步策略</h3>
              </div>

              <div class="grid gap-5 pl-3 border-l border-muted">
                <!-- 模式切换卡片 -->
                <div class="space-y-2">
                  <Label class="text-xs text-foreground/70 uppercase tracking-wider font-bold">同步触发模式</Label>
                  <div class="grid grid-cols-1 sm:grid-cols-3 gap-2.5">
                    <!-- 混合模式 -->
                    <div
                      class="p-3 rounded-xl border cursor-pointer transition-all flex flex-col justify-between"
                      :class="syncMode === 'hybrid'
                        ? 'border-amber-500/40 bg-amber-500/5 dark:bg-amber-500/[0.07] shadow-xs'
                        : 'border-border/60 bg-muted/10 hover:bg-muted/20 opacity-70 hover:opacity-100'"
                      @click="syncMode = 'hybrid'"
                    >
                      <div class="flex items-center justify-between mb-1.5">
                        <span class="text-xs font-semibold text-foreground flex items-center gap-1.5">
                          <Sparkles class="h-3.5 w-3.5 text-amber-500/80 dark:text-amber-400/80" />
                          <span>混合模式</span>
                        </span>
                        <span class="text-[9px] px-1.5 py-0.5 rounded bg-amber-500/10 text-amber-600/90 dark:text-amber-400/80 border border-amber-500/20 font-mono font-medium">推荐</span>
                      </div>
                      <p class="text-[10px] text-muted-foreground/80 leading-relaxed">
                        文件变动即时防抖推送，同时维持定时周期全量核对兜底。
                      </p>
                    </div>

                    <!-- 纯实时监听 -->
                    <div
                      class="p-3 rounded-xl border cursor-pointer transition-all flex flex-col justify-between"
                      :class="syncMode === 'realtime'
                        ? 'border-amber-500/40 bg-amber-500/5 dark:bg-amber-500/[0.07] shadow-xs'
                        : 'border-border/60 bg-muted/10 hover:bg-muted/20 opacity-70 hover:opacity-100'"
                      @click="syncMode = 'realtime'"
                    >
                      <div class="flex items-center justify-between mb-1.5">
                        <span class="text-xs font-semibold text-foreground flex items-center gap-1.5">
                          <Zap class="h-3.5 w-3.5 text-amber-500/80 dark:text-amber-400/80" />
                          <span>实时监听</span>
                        </span>
                      </div>
                      <p class="text-[10px] text-muted-foreground/80 leading-relaxed">
                        仅在本地文件被修改、重命名或保存时秒级下发，不占定时计划。
                      </p>
                    </div>

                    <!-- 仅定时模式 -->
                    <div
                      class="p-3 rounded-xl border cursor-pointer transition-all flex flex-col justify-between"
                      :class="syncMode === 'cron'
                        ? 'border-amber-500/40 bg-amber-500/5 dark:bg-amber-500/[0.07] shadow-xs'
                        : 'border-border/60 bg-muted/10 hover:bg-muted/20 opacity-70 hover:opacity-100'"
                      @click="syncMode = 'cron'"
                    >
                      <div class="flex items-center justify-between mb-1.5">
                        <span class="text-xs font-semibold text-foreground flex items-center gap-1.5">
                          <Clock class="h-3.5 w-3.5 text-amber-500/80 dark:text-amber-400/80" />
                          <span>计划定时</span>
                        </span>
                      </div>
                      <p class="text-[10px] text-muted-foreground/80 leading-relaxed">
                        关闭实时文件变动监听，仅按设定的标准 Cron 定时规则执行同步。
                      </p>
                    </div>
                  </div>
                </div>

                <!-- 实时同步选项 (realtime 或 hybrid 模式下展示) -->
                <div v-if="syncMode === 'realtime' || syncMode === 'hybrid'" class="p-3.5 rounded-xl bg-amber-500/5 border border-amber-500/15 space-y-3.5">
                  <div class="flex items-center justify-between">
                    <div class="space-y-0.5">
                      <Label class="text-xs font-bold text-foreground flex items-center gap-1.5">
                        <Zap class="h-3.5 w-3.5 text-amber-500" />
                        <span>实时变更防抖延迟 (秒)</span>
                      </Label>
                      <p class="text-[10px] text-muted-foreground">连续频繁保存时合并等待多少秒再触发打包，避免频繁推送 (默认 8 秒)</p>
                    </div>
                    <Input
                      type="number"
                      v-model.number="debounceDelay"
                      min="1"
                      max="60"
                      class="w-20 h-8 text-center text-xs font-mono bg-background"
                    />
                  </div>

                  <div class="flex items-center justify-between pt-2 border-t border-amber-500/10">
                    <div class="space-y-0.5">
                      <Label for="sync-on-online" class="text-xs font-bold text-foreground cursor-pointer">
                        Agent 重新上线后自动同步
                      </Label>
                      <p class="text-[10px] text-muted-foreground">当目标 Agent 从离线重连成功并握手后，自动触发一次同步补偿</p>
                    </div>
                    <Switch
                      id="sync-on-online"
                      :checked="syncOnOnline"
                      @update:checked="syncOnOnline = $event"
                      class="scale-90"
                    />
                  </div>
                </div>

                <!-- 定时调度配置 (cron 或 hybrid 模式下展示) -->
                <div v-if="syncMode === 'cron' || syncMode === 'hybrid'" class="space-y-4 pt-1">
                  <div class="flex items-center gap-2">
                    <Clock class="h-4 w-4 text-primary" />
                    <span class="text-xs font-bold text-foreground/80">Cron 定时规则配置</span>
                  </div>
                  <TaskCronConfig v-model="form.schedule" />
                </div>

                <!-- 高级运行策略与日志清理 (实时/定时/混合模式均支持) -->
                <div class="space-y-4 pt-2">
                  <div class="flex items-center gap-2">
                    <div class="h-3 w-1 bg-amber-500 rounded-full" />
                    <span class="text-xs font-bold text-foreground/80">高级选项与日志清除策略</span>
                  </div>
                  <TaskAdvancedConfig v-model="form" :show-random="syncMode !== 'realtime'" />
                </div>
              </div>
            </section>

            <!-- 通知配置 -->
            <TaskNotificationConfig ref="notificationConfigRef" :task-id="isEdit ? task?.id : undefined" />
          </div>
        </ScrollArea>

        <div class="flex items-center justify-between px-6 py-4 bg-muted/20 border-t shrink-0 backdrop-blur-sm">
          <div class="text-[10px] text-muted-foreground/40 italic flex flex-col leading-tight select-none pointer-events-none">
            <span>最后编辑于:</span>
            <span :title="form.updated_at || ''">{{ isEdit ? (form.updated_at ? formatDateTime(form.updated_at) : '刚才') : '现在' }}</span>
          </div>
          <div class="flex gap-3">
            <Button variant="ghost" size="sm" class="hover:bg-muted font-medium text-xs px-6" @click="emit('update:open', false)">取消</Button>
            <Button size="sm" class="px-8 font-semibold text-xs shadow-lg shadow-amber-500/20 transition-all hover:scale-105 active:scale-95 bg-amber-500 hover:bg-amber-600 text-white" @click="save">
              确定保存
            </Button>
          </div>
        </div>
      </div>
    </DialogContent>
  </Dialog>
</template>
