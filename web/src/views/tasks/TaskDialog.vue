<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'
import { ScrollArea } from '@/components/ui/scroll-area'
import DirTreeSelect from '@/components/DirTreeSelect.vue'
import { Plus, ChevronDown, X, Search, Check, ChevronsUpDown, AlertCircle, Terminal, Zap, Loader2 } from 'lucide-vue-next'
import { cn } from '@/lib/utils'
import { api, type Task, type EnvVar, type Agent, type MiseLanguage } from '@/api'
import { PATHS, TRIGGER_TYPE } from '@/constants'
import { toast } from 'vue-sonner'
import { getCronDescription } from '@/utils/cron'
import TaskNotificationConfig from './components/TaskNotificationConfig.vue'

const props = defineProps<{
  open: boolean
  task?: Partial<Task>
  isEdit: boolean
}>()

const emit = defineEmits<{
  'update:open': [value: boolean]
  'saved': []
}>()

const cronPresets = [
  { label: '每5秒', value: '*/5 * * * * *' },
  { label: '每30秒', value: '*/30 * * * * *' },
  { label: '每分钟', value: '0 * * * * *' },
  { label: '每5分钟', value: '0 */5 * * * *' },
  { label: '每小时', value: '0 0 * * * *' },
  { label: '每天0点', value: '0 0 0 * * *' },
  { label: '每天8点', value: '0 0 8 * * *' },
  { label: '每周一', value: '0 0 0 * * 1' },
  { label: '每月1号', value: '0 0 0 1 * *' },
]

const form = ref<Partial<Task>>({})
const tagInput = ref('')
const cleanType = ref('none')
const cleanKeep = ref(30)
const allEnvVars = ref<EnvVar[]>([])
const allAgents = ref<Agent[]>([])
const selectedEnvIds = ref<string[]>([])
const selectedAgentId = ref<string>('local')
const selectedTriggerType = ref<string>('cron')
const envSearchQuery = ref('')
// 为每个执行位置保存独立的工作目录配置
const workDirCache = ref<Record<string, string>>({})
const concurrency = ref(0)
const concurrencyEnabled = ref(false)
const allEnvsEnabled = ref(false)
const SCRIPTS_DIR_PLACEHPLDER = '$SCRIPTS_DIR$'
const scriptsDir = ref<string>(PATHS.SCRIPTS_DIR)

const cronDescription = computed(() => {
  if (!form.value.schedule) return ''
  return getCronDescription(form.value.schedule, (navigator as any).language)
})



// 监听 concurrencyEnabled 的变化，同步到 concurrency
watch(concurrencyEnabled, (val: boolean) => {
  concurrency.value = val ? 1 : 0
})



function onAllEnvsChange(val: boolean) {
  allEnvsEnabled.value = val
}

function addTag() {
  const val = tagInput.value.trim()
  if (!val) return
  const currentTags = form.value.tags ? form.value.tags.split(',').filter(Boolean) : []
  if (!currentTags.includes(val)) {
    currentTags.push(val)
    form.value.tags = currentTags.join(',')
  }
  tagInput.value = ''
}

function removeTag(tagToRemove: string) {
  const currentTags = form.value.tags ? form.value.tags.split(',').filter(Boolean) : []
  form.value.tags = currentTags.filter((t: string) => t !== tagToRemove).join(',')
}

// 当前显示的工作目录（根据选择的执行位置）
const currentWorkDir = computed({
  get: () => workDirCache.value[selectedAgentId.value] || '',
  set: (val: string) => {
    workDirCache.value[selectedAgentId.value] = val
  }
})

const cleanConfig = computed(() => {
  if (!cleanType.value || cleanType.value === 'none' || cleanKeep.value <= 0) return ''
  return JSON.stringify({ type: cleanType.value, keep: cleanKeep.value })
})

const filteredEnvVars = computed(() => {
  return allEnvVars.value.filter((env: EnvVar) => {
    const matchSearch = !envSearchQuery.value || env.name.toLowerCase().includes(envSearchQuery.value.toLowerCase())
    const notSelected = !selectedEnvIds.value.includes(env.id)
    return matchSearch && notSelected
  })
})

const selectedEnvs = computed(() => {
  return selectedEnvIds.value
    .map((id: string) => allEnvVars.value.find((e: EnvVar) => e.id === id))
    .filter((e): e is EnvVar => e !== undefined)
})

const onlineAgents = computed(() => {
  return allAgents.value.filter((a: Agent) => a.enabled)
})

// 语言环境相关
const installedLangs = ref<MiseLanguage[]>([])
const loadingLangs = ref(false)
const selectedLangs = ref<{ name: string; version: string; availableVersions: string[] }[]>([])

const availablePlugins = ref<string[]>([])
const pluginSearch = ref('')
const versionSearch = ref('')

const notificationConfigRef = ref<InstanceType<typeof TaskNotificationConfig> | null>(null)

const filteredPlugins = computed(() => {
  if (!pluginSearch.value) return availablePlugins.value
  const s = pluginSearch.value.toLowerCase()
  return availablePlugins.value.filter((p: string) => p.toLowerCase().includes(s))
})

function getFilteredVersions(versions: string[]) {
  if (!versionSearch.value) return versions
  const s = versionSearch.value.toLowerCase()
  return versions.filter((v: string) => v.toLowerCase().includes(s))
}

async function fetchInstalledLangs() {
  loadingLangs.value = true
  try {
    installedLangs.value = await api.mise.list()
    const plugins = new Set<string>()
    installedLangs.value.forEach((l: MiseLanguage) => plugins.add(l.plugin))
    availablePlugins.value = Array.from(plugins).sort()
  } catch (e) {
    console.error('Fetch installed langs failed', e)
  } finally {
    loadingLangs.value = false
  }
}

function getLangIcon(plugin: string) {
  const name = plugin?.toLowerCase().trim()
  const mapping: Record<string, string> = {
    'python': 'python/python-original.svg',
    'node': 'nodejs/nodejs-original.svg',
    'nodejs': 'nodejs/nodejs-original.svg',
    'go': 'go/go-original.svg',
    'rust': 'rust/rust-original.svg',
    'ruby': 'ruby/ruby-plain.svg',
    'php': 'php/php-plain.svg',
    'java': 'java/java-plain.svg',
    'deno': 'deno/deno-plain.svg',
    'bun': 'bun/bun-plain.svg',
    'zig': 'zig/zig-original.svg',
    'dotnet': 'dot-net/dot-net-original.svg',
    '.net': 'dot-net/dot-net-original.svg',
    'elixir': 'elixir/elixir-original.svg',
    'erlang': 'erlang/erlang-original.svg',
    'crystal': 'crystal/crystal-original.svg',
    'lua': 'lua/lua-original.svg',
    'julia': 'julia/julia-original.svg',
    'nim': 'nim/nim-original.svg',
    'perl': 'perl/perl-original.svg',
    'scala': 'scala/scala-original.svg',
    'kotlin': 'kotlin/kotlin-original.svg',
    'clojure': 'clojure/clojure-line.svg',
    'dart': 'dart/dart-original.svg',
    'flutter': 'flutter/flutter-original.svg',
    'terraform': 'terraform/terraform-original.svg',
    'docker': 'docker/docker-original.svg',
    'kubernetes': 'kubernetes/kubernetes-plain.svg',
    'ansible': 'ansible/ansible-original.svg',
  }

  if (mapping[name]) {
    return `https://fastly.jsdelivr.net/gh/devicons/devicon/icons/${mapping[name]}`
  }
  return ''
}

function updateAvailableVersions(lang: { name: string; version: string; availableVersions: string[] }) {
  if (lang.name) {
    lang.availableVersions = installedLangs.value
      .filter((l: MiseLanguage) => l.plugin === lang.name)
      .map((l: MiseLanguage) => l.version)
      .sort((a: string, b: string) => b.localeCompare(a, undefined, { numeric: true }))
  } else {
    lang.availableVersions = []
  }
}

function addLang() {
  selectedLangs.value.push({ name: '', version: '', availableVersions: [] })
}

function removeLang(index: number) {
  selectedLangs.value.splice(index, 1)
}

function updateLangName(index: number, name: string) {
  const lang = selectedLangs.value[index]
  if (!lang) return
  lang.name = name
  lang.version = '' // reset version
  updateAvailableVersions(lang)
}

watch(() => props.open, async (val: boolean) => {
  if (val) {
    form.value = {
      retry_count: props.task?.retry_count ?? 0,
      retry_interval: props.task?.retry_interval ?? 0,
      random_range: props.task?.random_range ?? 0,
      timeout: props.task?.timeout ?? 30,
      ...props.task
    }
    // 解析清理配置
    if (props.task?.clean_config) {
      try {
        const config = JSON.parse(props.task.clean_config)
        cleanType.value = config.type || 'none'
        cleanKeep.value = config.keep || 30
      } catch {
        cleanType.value = 'none'
        cleanKeep.value = 30
      }
    } else {
      cleanType.value = 'none'
      cleanKeep.value = 30
    }
    // 解析任务配置
    try {
      // 确保 config 是有效的 JSON 对象字符串
      let configStr = props.task?.config
      // 如果是 null/undefined 或者空字符串，初始化为 '{}'
      if (!configStr) {
        configStr = '{}'
      }

      const parsed = JSON.parse(configStr)
      // 确保解析结果是对象
      if (parsed && typeof parsed === 'object') {
        const val = parsed['$task_concurrency']
        if (typeof val === 'number') {
          // 如果已存在并发配置，直接使用（0 或 1）
          concurrency.value = val
          concurrencyEnabled.value = val === 1
        } else {
          // 默认值：允许并发
          concurrency.value = 1
          concurrencyEnabled.value = true
        }

        // 解析全部环境变量配置
        allEnvsEnabled.value = !!parsed['$task_all_envs']
      } else {
        concurrency.value = 1
        concurrencyEnabled.value = true
        allEnvsEnabled.value = false
      }
    } catch {
      concurrency.value = 1
      concurrencyEnabled.value = true
    }
    // 解析环境变量
    if (props.task?.envs) {
      selectedEnvIds.value = props.task.envs.split(',').map((s: string) => s.trim()).filter(Boolean)
    } else {
      selectedEnvIds.value = []
    }
    // 解析语言环境
    selectedLangs.value = []
    if (props.task?.languages && Array.isArray(props.task.languages)) {
      selectedLangs.value = props.task.languages.map((l: any) => ({
        name: l.name || '',
        version: l.version || '',
        availableVersions: []
      }))
    }

    // 解析 Agent 和工作目录
    const agentId = props.task?.agent_id ? String(props.task.agent_id) : 'local'
    selectedAgentId.value = agentId
    // 解析触发类型
    selectedTriggerType.value = props.task?.trigger_type || TRIGGER_TYPE.CRON
    // 初始化工作目录缓存，将当前任务的工作目录保存到对应的执行位置
    workDirCache.value = {
      [agentId]: props.task?.work_dir || ''
    }
    envSearchQuery.value = ''
    // 加载数据
    await loadData()
    workDirCache.value = {
      [agentId]: agentId === 'local'
        ? normalizeLocalWorkDirForDisplay(props.task?.work_dir)
        : (props.task?.work_dir || '')
    }
    if (selectedAgentId.value === 'local') {
      await fetchInstalledLangs()
      // 更新所有语言的可用版本
      selectedLangs.value.forEach((lang: { name: string; version: string; availableVersions: string[] }) => {
        updateAvailableVersions(lang)
      })
    }
    // 加载通知配置
    await notificationConfigRef.value?.loadConfig(props.isEdit ? props.task?.id : undefined)
  }
})

async function loadData() {
  try {
    const [envs, agents, paths] = await Promise.all([
      api.env.all(),
      api.agents.list(),
      api.settings.getPaths().catch(() => ({ scripts_dir: PATHS.SCRIPTS_DIR }))
    ])
    allEnvVars.value = envs
    allAgents.value = agents
    scriptsDir.value = paths?.scripts_dir || PATHS.SCRIPTS_DIR
  } catch { /* ignore */ }
}

function addEnv(id: string) {
  if (!selectedEnvIds.value.includes(id)) {
    selectedEnvIds.value.push(id)
  }
}

function removeEnv(id: string) {
  selectedEnvIds.value = selectedEnvIds.value.filter((envId: string) => envId !== id)
}

function normalizeLocalWorkDirForDisplay(workDir?: string | null): string {
  if (!workDir) return ''
  if (workDir === SCRIPTS_DIR_PLACEHPLDER) return ''
  if (workDir.startsWith(`${SCRIPTS_DIR_PLACEHPLDER}/`)) {
    return workDir.slice(SCRIPTS_DIR_PLACEHPLDER.length + 1)
  }
  const base = scriptsDir.value || PATHS.SCRIPTS_DIR
  if (workDir === base) return ''
  if (workDir.startsWith(`${base}/`)) {
    return workDir.slice(base.length + 1)
  }
  return workDir
}

function encodeLocalWorkDir(workDir?: string | null): string {
  const value = workDir?.trim() || ''
  if (!value) return SCRIPTS_DIR_PLACEHPLDER
  if (value === SCRIPTS_DIR_PLACEHPLDER || value.startsWith(`${SCRIPTS_DIR_PLACEHPLDER}/`)) {
    return value
  }
  const base = scriptsDir.value || PATHS.SCRIPTS_DIR
  if (value === base) return SCRIPTS_DIR_PLACEHPLDER
  if (value.startsWith(`${base}/`)) {
    return `${SCRIPTS_DIR_PLACEHPLDER}/${value.slice(base.length + 1)}`
  }
  return `${SCRIPTS_DIR_PLACEHPLDER}/${value.replace(/^\/+/, '')}`
}

async function save() {
  try {
    form.value.clean_config = cleanConfig.value
    form.value.envs = selectedEnvIds.value.join(',')
    form.value.type = 'task'
    form.value.trigger_type = selectedTriggerType.value
    form.value.agent_id = selectedAgentId.value === 'local' ? null : selectedAgentId.value

    // 保存语言环境配置
    form.value.languages = selectedLangs.value.map((l: { name: string; version: string }) => ({
      name: l.name,
      version: l.version
    }))

    // 保存配置 - 确保 concurrency 字段被正确保存
    let config: Record<string, any> = {}

    // 如果 form.value.config 存在，先解析它以保留其他配置
    if (form.value.config) {
      try {
        const parsed = JSON.parse(form.value.config)
        if (parsed && typeof parsed === 'object') {
          config = parsed
        }
      } catch {
        config = {}
      }
    }

    // 更新并发控制字段 (1: 开启, 0: 关闭)
    config['$task_concurrency'] = concurrencyEnabled.value ? 1 : 0
    // 更新注入全部环境变量字段
    config['$task_all_envs'] = !!allEnvsEnabled.value

    // 重新序列化配置
    form.value.config = JSON.stringify(config)

    // 保存当前选择的执行位置对应的工作目录
    form.value.work_dir = selectedAgentId.value === 'local'
      ? encodeLocalWorkDir(currentWorkDir.value)
      : currentWorkDir.value

    if (props.isEdit && form.value.id) {
      const task = await api.tasks.update(form.value.id, form.value)
      await notificationConfigRef.value?.saveConfig(task.id)
      toast.success('任务已更新')
    } else {
      const task = await api.tasks.create(form.value)
      await notificationConfigRef.value?.saveConfig(task.id)
      toast.success('任务已创建')
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
    <DialogContent class="max-w-[95vw] sm:max-w-[600px] xl:max-w-[850px] p-0 overflow-hidden border-none bg-background shadow-2xl transition-all duration-300" style="text-rendering: optimizeLegibility;" @openAutoFocus.prevent>
      <div class="flex flex-col max-h-[85vh]">
        <DialogHeader class="px-6 pr-12 pt-6 pb-2 shrink-0 border-b border-muted/50">
          <DialogTitle class="text-xl font-bold py-2">
            {{ isEdit ? '编辑任务' : '新建任务' }}
          </DialogTitle>
        </DialogHeader>

        <ScrollArea class="flex-1 min-h-0 px-6">
          <div class="space-y-10 py-6 pb-10">
            <!-- 基本信息 Section -->
            <section class="space-y-4">
              <div class="flex items-center gap-2 mb-2">
                <div class="h-4 w-1 bg-primary rounded-full shadow-sm shadow-primary/20" />
                <h3 class="text-sm font-bold text-foreground/90">基本信息</h3>
              </div>
              <div class="grid gap-5 pl-3 border-l border-muted">
                <div class="grid grid-cols-1 sm:grid-cols-4 items-center gap-3">
                  <Label class="sm:text-right text-xs text-foreground/70 uppercase tracking-wider font-bold">任务名称</Label>
                  <Input v-model="form.name" placeholder="输入任务描述性名称" :class="cn('sm:col-span-3 h-9 bg-muted/20 border-muted-foreground/15 transition-all focus:bg-background/50', form.name ? 'text-sm font-medium' : 'text-[11px] font-normal')" />
                </div>
                <div class="grid grid-cols-1 sm:grid-cols-4 items-center gap-3">
                  <Label class="sm:text-right text-xs text-foreground/70 uppercase tracking-wider font-bold">任务备注</Label>
                  <Input v-model="form.remark" placeholder="输入任务备注信息 (可选)" :class="cn('sm:col-span-3 h-9 bg-muted/20 border-muted-foreground/15 transition-all focus:bg-background/50', form.remark ? 'text-sm font-medium' : 'text-[11px] font-normal')" />
                </div>
                <div class="grid grid-cols-1 sm:grid-cols-4 items-start gap-3">
                  <Label class="sm:text-right text-xs text-foreground/70 uppercase tracking-wider font-bold pt-2.5">任务标签</Label>
                  <div class="sm:col-span-3 space-y-2">
                    <div class="flex gap-2">
                      <div class="relative flex-1">
                        <Input v-model="tagInput" placeholder="输入标签按回车..." :class="cn('h-9 bg-muted/20 border-muted-foreground/15 transition-all pr-12', tagInput ? 'text-sm font-medium' : 'text-[11px] font-normal')" @keydown.enter.prevent="addTag" />
                        <Button type="button" variant="ghost" size="sm" class="absolute right-1 top-1 h-7 px-2 text-xs hover:bg-primary/10 hover:text-primary transition-colors" @click.prevent="addTag">添加</Button>
                      </div>
                    </div>
                    <div v-if="form.tags" class="flex flex-wrap gap-1.5 pt-1">
                      <span v-for="tag in form.tags.split(',').filter(Boolean)" :key="tag" class="flex items-center gap-1.5 bg-primary/5 text-primary px-2.5 py-1 rounded-full text-[11px] font-medium border border-primary/10 group transition-all hover:bg-primary/10">
                        {{ tag }}
                        <button type="button" class="text-primary/40 hover:text-destructive transition-colors shrink-0" @click.prevent="removeTag(tag)"><X class="h-3 w-3" /></button>
                      </span>
                    </div>
                  </div>
                </div>
                <div class="grid grid-cols-1 sm:grid-cols-4 items-center gap-3">
                  <Label class="sm:text-right text-xs text-foreground/70 uppercase tracking-wider font-bold">执行位置</Label>
                  <div class="sm:col-span-3">
                    <Select v-model="selectedAgentId">
                      <SelectTrigger class="h-9 bg-muted/20 border-muted-foreground/15"><SelectValue placeholder="选择执行节点" /></SelectTrigger>
                      <SelectContent>
                        <SelectItem value="local" class="flex items-center gap-2"><div class="flex items-center gap-2"><div class="w-1.5 h-1.5 rounded-full bg-blue-500" /><span>本地执行 (Local)</span></div></SelectItem>
                        <SelectItem v-for="agent in onlineAgents" :key="agent.id" :value="String(agent.id)"><div class="flex items-center gap-2"><div class="w-1.5 h-1.5 rounded-full" :class="agent.status === 'online' ? 'bg-green-500' : 'bg-muted-foreground'" /><span>{{ agent.name }}</span></div></SelectItem>
                      </SelectContent>
                    </Select>
                  </div>
                </div>
                <div class="grid grid-cols-1 sm:grid-cols-4 items-center gap-3">
                  <Label class="sm:text-right text-xs text-foreground/70 uppercase tracking-wider font-bold">触发方式</Label>
                  <div class="sm:col-span-3">
                    <Select v-model="selectedTriggerType">
                      <SelectTrigger class="h-9 bg-muted/20 border-muted-foreground/15"><SelectValue /></SelectTrigger>
                      <SelectContent>
                        <SelectItem :value="TRIGGER_TYPE.CRON">⏳ 定时周期触发</SelectItem>
                        <SelectItem :value="TRIGGER_TYPE.BAIHU_STARTUP">🚀 系统启动触发</SelectItem>
                      </SelectContent>
                    </Select>
                  </div>
                </div>
              </div>
            </section>

            <!-- 执行配置 Section -->
            <section class="space-y-4">
              <div class="flex items-center gap-2 mb-2">
                <div class="h-4 w-1 bg-primary rounded-full shadow-sm shadow-primary/20" />
                <h3 class="text-sm font-bold text-foreground/90">执行配置</h3>
              </div>
              <div class="grid gap-5 pl-3 border-l border-muted">
                <template v-if="selectedAgentId === 'local'">
                  <div class="grid grid-cols-1 sm:grid-cols-4 items-start gap-3">
                    <div class="sm:col-span-1" />
                    <div class="sm:col-span-3">
                      <div class="flex items-center gap-2.5 p-3 rounded-xl bg-amber-500/5 border border-amber-500/10 text-amber-600 dark:text-amber-400 text-[11px] leading-relaxed font-medium">
                        <AlertCircle class="h-4 w-4 shrink-0 text-amber-500" /><p>请先在<b>「语言依赖」</b>中安装所需的运行时。执行脚本时将自动注入该环境。</p>
                      </div>
                    </div>
                  </div>
                  <div class="grid grid-cols-1 sm:grid-cols-4 items-start gap-3">
                    <Label class="sm:text-right text-xs text-foreground/70 uppercase tracking-wider font-bold pt-2.5">语言环境</Label>
                    <div class="sm:col-span-3 space-y-2">
                      <div v-for="(clang, idx) in selectedLangs" :key="idx" class="flex gap-2 p-2 rounded-lg bg-muted/20 border border-muted-foreground/10 group/lang relative overflow-hidden">
                        <div class="absolute left-0 top-0 bottom-0 w-0.5 bg-primary/20 group-hover/lang:bg-primary transition-colors" />
                        <Popover>
                          <PopoverTrigger asChild><Button variant="ghost" class="justify-between flex-1 h-8 text-xs font-normal hover:bg-background/50"><div class="flex items-center gap-2 truncate"><div v-if="clang.name && getLangIcon(clang.name)" class="w-4 h-4 shrink-0 rounded-sm bg-white p-0.5 border shadow-sm"><img :src="getLangIcon(clang.name)" class="w-full h-full object-contain" /></div><span class="font-medium">{{ clang.name || "选择环境..." }}</span></div><ChevronsUpDown class="ml-1 h-3 w-3 opacity-40" /></Button></PopoverTrigger>
                          <PopoverContent class="p-0 w-[240px]" align="start">
                            <div class="p-2 border-b bg-muted/30"><div class="relative"><Search class="absolute left-2 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-muted-foreground" /><Input v-model="pluginSearch" placeholder="搜索已安装语言..." :class="cn('h-8 pl-8 bg-background border-muted-foreground/20', pluginSearch ? 'text-xs font-medium' : 'text-[10px]')" /></div></div>
                            <ScrollArea class="h-48 p-1">
                              <div v-if="loadingLangs" class="flex items-center justify-center py-6"><Loader2 class="h-5 w-5 animate-spin text-primary/50" /></div>
                              <button v-else v-for="p in filteredPlugins" :key="p" @click="updateLangName(idx, p)" class="w-full flex items-center px-3 py-2 text-xs rounded-md hover:bg-accent text-left transition-all mb-0.5"><span class="flex-1" :class="{ 'font-bold text-primary': clang.name === p }">{{ p }}</span><Check v-if="clang.name === p" class="h-3 w-3 text-primary" /></button>
                            </ScrollArea>
                          </PopoverContent>
                        </Popover>
                        <Popover>
                          <PopoverTrigger asChild :disabled="!clang.name"><Button variant="ghost" class="justify-between w-28 h-8 text-xs font-normal hover:bg-background/50" :disabled="!clang.name"><span class="truncate">{{ clang.version || "版本..." }}</span><ChevronsUpDown class="h-3 w-3 opacity-40 ml-1" /></Button></PopoverTrigger>
                          <PopoverContent class="p-0 w-[160px]" align="start">
                            <ScrollArea class="h-48 p-1"><button v-for="v in getFilteredVersions(clang.availableVersions)" :key="v" @click="clang.version = v" class="w-full flex items-center px-3 py-2 text-xs rounded-md hover:bg-accent font-mono mb-0.5"><span class="flex-1 truncate" :class="{ 'font-bold text-primary': clang.version === v }">{{ v }}</span><Check v-if="clang.version === v" class="h-3 w-3 text-primary" /></button></ScrollArea>
                          </PopoverContent>
                        </Popover>
                        <Button variant="ghost" size="icon" class="h-8 w-8 text-muted-foreground hover:text-destructive hover:bg-destructive/10 shrink-0" @click="removeLang(idx)"><X class="h-4 w-4" /></Button>
                      </div>
                      <Button variant="outline" size="sm" class="w-full h-9 text-xs border-dashed border-muted-foreground/30 text-muted-foreground hover:text-primary hover:border-primary/50 transition-all bg-muted/5 hover:bg-primary/5 rounded-xl" @click="addLang"><Plus class="h-4 w-4 mr-2" /> 添加运行时环境 (Mise)</Button>
                    </div>
                  </div>
                </template>
                <div class="grid grid-cols-1 sm:grid-cols-4 items-center gap-3">
                  <Label class="sm:text-right text-xs text-foreground/70 uppercase tracking-wider font-bold">执行命令</Label>
                  <div class="sm:col-span-3 relative"><Input v-model="form.command" placeholder="例如: python main.py --args" :class="cn('h-9 bg-muted/20 border-muted-foreground/15 transition-all focus:bg-background/50 pr-10', form.command ? 'font-mono text-sm tracking-tight font-medium' : 'text-[11px] font-normal')" /><Terminal class="absolute right-3 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-muted-foreground opacity-40 pointer-events-none" /></div>
                </div>
                <div class="grid grid-cols-1 sm:grid-cols-4 items-center gap-3">
                  <Label class="sm:text-right text-xs text-foreground/70 uppercase tracking-wider font-bold">工作目录</Label>
                  <div class="sm:col-span-3"><DirTreeSelect v-if="selectedAgentId === 'local'" v-model="currentWorkDir" class="h-9" /><Input v-else v-model="currentWorkDir" placeholder="任务运行路径（留空取 Agent 默认值）" :class="cn('h-9 bg-muted/20 border-muted-foreground/15 transition-all focus:bg-background/50', currentWorkDir ? 'font-mono text-sm tracking-tight font-medium' : 'text-[11px] font-normal')" /></div>
                </div>
                <div class="grid grid-cols-1 sm:grid-cols-4 items-center gap-3 pb-1">
                  <Label class="sm:text-right text-xs text-foreground/70 uppercase tracking-wider font-bold">变量注入</Label>
                  <div class="sm:col-span-3"><div class="flex items-center space-x-2 bg-muted/10 px-3 py-1.5 rounded-full border border-muted-foreground/10 w-fit"><Switch :model-value="allEnvsEnabled" @update:model-value="onAllEnvsChange" id="all-envs" class="scale-90" /><Label for="all-envs" class="text-[11px] font-medium cursor-pointer">全量注入</Label></div></div>
                </div>
                <div v-if="!allEnvsEnabled" class="grid grid-cols-1 sm:grid-cols-4 items-start gap-3">
                  <Label class="sm:text-right text-xs text-foreground/70 uppercase tracking-wider font-bold pt-2.5">按需包含</Label>
                  <div class="sm:col-span-3 space-y-2">
                    <Popover><PopoverTrigger as-child><Button variant="outline" class="w-full justify-between h-9 bg-muted/10 border-muted-foreground/15 text-xs rounded-xl"><span class="text-muted-foreground">选择关联的环境变量...</span><ChevronDown class="h-4 w-4 opacity-30" /></Button></PopoverTrigger>
                      <PopoverContent class="p-0 w-[400px]" align="start"><ScrollArea class="h-64 p-2"><div v-for="env in filteredEnvVars" :key="env.id" @click.stop="addEnv(env.id)" class="p-3 rounded-lg hover:bg-primary/5 cursor-pointer text-sm font-mono font-bold">{{ env.name }}</div></ScrollArea></PopoverContent>
                    </Popover>
                    <div v-if="selectedEnvs.length > 0" class="flex flex-wrap gap-2 p-3 rounded-xl bg-muted/10 border border-muted-foreground/10 min-h-12"><div v-for="env in selectedEnvs" :key="env?.id" class="flex items-center gap-1.5 px-2.5 py-1 rounded-full bg-background border border-muted-foreground/15 text-[11px] font-mono font-medium">{{ env?.name }}<X class="h-2.5 w-2.5 cursor-pointer" @click="removeEnv(env!.id)" /></div></div>
                  </div>
                </div>
              </div>
            </section>

            <!-- 调度策略 Section -->
            <section class="space-y-4">
              <div class="flex items-center gap-2 mb-2">
                <div class="h-4 w-1 bg-primary rounded-full shadow-sm shadow-primary/20" />
                <h3 class="text-sm font-bold text-foreground/90">调度策略</h3>
              </div>
              <div class="grid gap-5 pl-3 border-l border-muted">
                <template v-if="selectedTriggerType === TRIGGER_TYPE.CRON">
                  <div class="grid grid-cols-1 sm:grid-cols-4 items-center gap-3">
                    <Label class="sm:text-right text-xs text-foreground/70 uppercase tracking-wider font-bold">定时规则</Label>
                    <div class="sm:col-span-3">
                      <Input v-model="form.schedule" placeholder="秒 分 时 日 月 周 (必须 6 位)" :class="cn('h-9 bg-muted/20 border-muted-foreground/15 transition-all focus:ring-1 focus:ring-primary/40 focus:border-primary/40', form.schedule ? 'font-mono text-sm tracking-[0.1em] font-medium' : 'text-[11px] font-normal')" />
                      <div v-if="cronDescription" class="mt-2.5 p-2 px-3 rounded-xl bg-primary/5 border border-primary/10 text-[11px] text-primary/80 font-medium flex items-center gap-2.5"><Zap class="h-3 w-3 text-primary" />{{ cronDescription }}</div>
                      <div class="mt-3 flex flex-wrap gap-1.5"><button v-for="preset in cronPresets" :key="preset.value" class="px-2.5 py-1 text-[10px] rounded-lg bg-muted/30 border border-transparent hover:border-primary/30 hover:bg-primary/5 font-medium" @click.prevent="form.schedule = preset.value">{{ preset.label }}</button></div>
                    </div>
                  </div>
                </template>
                <div class="grid grid-cols-1 sm:grid-cols-4 items-center gap-3">
                  <Label class="sm:text-right text-xs text-foreground/70 uppercase tracking-wider font-bold">失败策略</Label>
                  <div class="sm:col-span-3 flex items-center gap-3">
                    <span class="text-[11px] text-muted-foreground">重试</span>
                    <Input :model-value="form.retry_count" @update:model-value="(v: string | number) => form.retry_count = Number(v)" type="number" :min="0" class="w-20 h-9 bg-muted/20 border-muted-foreground/15 text-center font-bold" />
                    <span class="text-[11px] text-muted-foreground">次</span>
                  </div>
                </div>
                <div class="grid grid-cols-1 sm:grid-cols-4 items-center gap-3">
                  <Label class="sm:text-right text-xs text-foreground/70 uppercase tracking-wider font-bold">运行策略</Label>
                  <div class="sm:col-span-3 flex items-center gap-3">
                    <Input :model-value="form.timeout" @update:model-value="(v: string | number) => form.timeout = Number(v)" type="number" :min="0" class="w-20 h-9 bg-muted/20 border-muted-foreground/15 text-center font-bold" />
                    <span class="text-[11px] text-muted-foreground">分钟超时</span>
                  </div>
                </div>
              </div>
            </section>
            <TaskNotificationConfig ref="notificationConfigRef" :task-id="isEdit ? task?.id : undefined" />
          </div>
        </ScrollArea>
        <div class="flex items-center justify-between px-6 py-4 bg-muted/20 border-t shrink-0 backdrop-blur-sm">
          <p class="text-[10px] text-muted-foreground/50 italic">最后编辑于: {{ isEdit ? (form.updated_at || '刚才') : '现在' }}</p>
          <div class="flex gap-3">
            <Button variant="ghost" size="sm" class="hover:bg-muted font-medium text-xs px-6" @click="emit('update:open', false)">取消</Button>
            <Button size="sm" class="px-8 font-semibold text-xs shadow-lg shadow-primary/20 bg-primary hover:bg-primary/90" @click="save">确定保存</Button>
          </div>
        </div>
      </div>
    </DialogContent>
  </Dialog>
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

