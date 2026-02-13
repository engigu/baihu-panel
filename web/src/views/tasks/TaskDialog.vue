<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'
import { ScrollArea } from '@/components/ui/scroll-area'
import DirTreeSelect from '@/components/DirTreeSelect.vue'
import { Plus, ChevronDown, X, Search, Check, ChevronsUpDown, Loader2, AlertCircle } from 'lucide-vue-next'
import { cn } from '@/lib/utils'
import { api, type Task, type EnvVar, type Agent, type MiseLanguage } from '@/api'
import { toast } from 'vue-sonner'

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
const cleanType = ref('none')
const cleanKeep = ref(30)
const allEnvVars = ref<EnvVar[]>([])
const allAgents = ref<Agent[]>([])
const selectedEnvIds = ref<number[]>([])
const selectedAgentId = ref<string>('local')
const envSearchQuery = ref('')
// 为每个执行位置保存独立的工作目录配置
const workDirCache = ref<Record<string, string>>({})
const concurrency = ref(0)
const concurrencyEnabled = ref(false)

// 监听 concurrencyEnabled 的变化，同步到 concurrency
watch(concurrencyEnabled, (val) => {
  concurrency.value = val ? 1 : 0
})

// 当前显示的工作目录（根据选择的执行位置）
const currentWorkDir = computed({
  get: () => workDirCache.value[selectedAgentId.value] || '',
  set: (val) => {
    workDirCache.value[selectedAgentId.value] = val
  }
})

const cleanConfig = computed(() => {
  if (!cleanType.value || cleanType.value === 'none' || cleanKeep.value <= 0) return ''
  return JSON.stringify({ type: cleanType.value, keep: cleanKeep.value })
})

const filteredEnvVars = computed(() => {
  return allEnvVars.value.filter(env => {
    const matchSearch = !envSearchQuery.value || env.name.toLowerCase().includes(envSearchQuery.value.toLowerCase())
    const notSelected = !selectedEnvIds.value.includes(env.id)
    return matchSearch && notSelected
  })
})

const selectedEnvs = computed(() => {
  return selectedEnvIds.value
    .map(id => allEnvVars.value.find(e => e.id === id))
    .filter((e): e is EnvVar => e !== undefined)
})

const onlineAgents = computed(() => {
  return allAgents.value.filter(a => a.enabled)
})

// 语言环境相关
const installedLangs = ref<MiseLanguage[]>([])
const loadingLangs = ref(false)
const availablePlugins = ref<string[]>([])
const pluginSearch = ref('')
const openPluginPopover = ref(false)

const availableVersions = ref<string[]>([])
const versionSearch = ref('')
const openVersionPopover = ref(false)

const filteredPlugins = computed(() => {
  if (!pluginSearch.value) return availablePlugins.value
  const s = pluginSearch.value.toLowerCase()
  return availablePlugins.value.filter(p => p.toLowerCase().includes(s))
})

const filteredVersions = computed(() => {
  if (!versionSearch.value) return availableVersions.value
  const s = versionSearch.value.toLowerCase()
  return availableVersions.value.filter(v => v.toLowerCase().includes(s))
})

async function fetchInstalledLangs() {
  loadingLangs.value = true
  try {
    installedLangs.value = await api.mise.list()
    const plugins = new Set<string>()
    installedLangs.value.forEach(l => plugins.add(l.plugin))
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
    return `https://cdn.jsdelivr.net/gh/devicons/devicon/icons/${mapping[name]}`
  }
  return ''
}

watch(() => form.value.language, (newVal) => {
  if (newVal) {
    availableVersions.value = installedLangs.value
      .filter(l => l.plugin === newVal)
      .map(l => l.version)
      .sort((a, b) => b.localeCompare(a, undefined, { numeric: true }))

    // 如果当前选中的版本不在可用列表中，且列表不为空，则自动选中第一个（通常是最新版）
    if (form.value.lang_version && !availableVersions.value.includes(form.value.lang_version)) {
      // 保持原样，可能还没加载完
    }
  } else {
    availableVersions.value = []
    form.value.lang_version = ''
  }
})

watch(() => props.open, async (val) => {
  if (val) {
    form.value = { ...props.task }
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
      } else {
        concurrency.value = 1
        concurrencyEnabled.value = true
      }
    } catch {
      concurrency.value = 1
      concurrencyEnabled.value = true
    }
    // 解析环境变量
    if (props.task?.envs) {
      selectedEnvIds.value = props.task.envs.split(',').map(s => parseInt(s.trim())).filter(n => !isNaN(n))
    } else {
      selectedEnvIds.value = []
    }
    // 解析 Agent 和工作目录
    const agentId = props.task?.agent_id ? String(props.task.agent_id) : 'local'
    selectedAgentId.value = agentId
    // 初始化工作目录缓存，将当前任务的工作目录保存到对应的执行位置
    workDirCache.value = {
      [agentId]: props.task?.work_dir || ''
    }
    envSearchQuery.value = ''
    // 加载数据
    await loadData()
    if (selectedAgentId.value === 'local') {
      await fetchInstalledLangs()
      // 如果已选择语言（编辑模式），手动触发一次版本更新
      if (form.value.language) {
        availableVersions.value = installedLangs.value
          .filter(l => l.plugin === form.value.language)
          .map(l => l.version)
          .sort((a, b) => b.localeCompare(a, undefined, { numeric: true }))
      }
    }
  }
})

async function loadData() {
  try {
    const [envs, agents] = await Promise.all([
      api.env.all(),
      api.agents.list()
    ])
    allEnvVars.value = envs
    allAgents.value = agents
  } catch { /* ignore */ }
}

function addEnv(id: number) {
  if (!selectedEnvIds.value.includes(id)) {
    selectedEnvIds.value.push(id)
  }
  envSearchQuery.value = ''
}

function removeEnv(id: number) {
  selectedEnvIds.value = selectedEnvIds.value.filter(envId => envId !== id)
}

async function save() {
  try {
    form.value.clean_config = cleanConfig.value
    form.value.envs = selectedEnvIds.value.join(',')
    form.value.type = 'task'
    form.value.agent_id = selectedAgentId.value === 'local' ? null : Number(selectedAgentId.value)

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
    config['$task_concurrency'] = concurrency.value

    // 重新序列化配置
    form.value.config = JSON.stringify(config)

    // 保存当前选择的执行位置对应的工作目录
    form.value.work_dir = currentWorkDir.value

    if (props.isEdit && form.value.id) {
      await api.tasks.update(form.value.id, form.value)
      toast.success('任务已更新')
    } else {
      await api.tasks.create(form.value)
      toast.success('任务已创建')
    }
    emit('update:open', false)
    emit('saved')
  } catch (error) {
    toast.error('保存失败')
  }
}
</script>

<template>
  <Dialog :open="open" @update:open="emit('update:open', $event)">
    <DialogContent class="sm:max-w-[480px]" @openAutoFocus.prevent>
      <DialogHeader>
        <DialogTitle>{{ isEdit ? '编辑任务' : '新建任务' }}</DialogTitle>
      </DialogHeader>
      <div class="space-y-3 py-3">
        <div class="grid grid-cols-1 sm:grid-cols-4 items-center gap-2 sm:gap-3">
          <Label class="sm:text-right text-sm">任务名称</Label>
          <Input v-model="form.name" placeholder="我的任务" class="sm:col-span-3 h-8 text-sm" />
        </div>

        <div class="grid grid-cols-1 sm:grid-cols-4 items-center gap-2 sm:gap-3">
          <Label class="sm:text-right text-sm">执行位置</Label>
          <div class="sm:col-span-3">
            <Select v-model="selectedAgentId">
              <SelectTrigger class="h-8 text-sm">
                <SelectValue placeholder="选择执行位置" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="local">本地执行</SelectItem>
                <SelectItem v-for="agent in onlineAgents" :key="agent.id" :value="String(agent.id)">
                  {{ agent.name }} ({{ agent.status === 'online' ? '在线' : '离线' }})
                </SelectItem>
              </SelectContent>
            </Select>
          </div>
        </div>

        <!-- 本地任务语言版本配置 -->
        <template v-if="selectedAgentId === 'local'">
          <div class="grid grid-cols-1 sm:grid-cols-4 items-start gap-2 sm:gap-3">
            <span></span>
            <div class="sm:col-span-3">
              <div
                class="flex items-start gap-2 p-2 rounded-md bg-amber-500/10 border border-amber-500/20 text-amber-600 dark:text-amber-500 text-[11px] leading-relaxed">
                <AlertCircle class="h-3.5 w-3.5 mt-0.5 shrink-0" />
                <p>请先在<b>「语言依赖」</b>中安装所需的运行时。任务执行时将使用该环境，确保所有依赖已正确配置（如果是执行 <b>bash</b> 脚本，可随便选择一个环境即可）。</p>
              </div>
            </div>
          </div>
          <div class="grid grid-cols-1 sm:grid-cols-4 items-center gap-2 sm:gap-3">
            <Label class="sm:text-right text-sm font-medium">语言环境</Label>
            <div class="sm:col-span-3 flex gap-2">
              <Popover v-model:open="openPluginPopover">
                <PopoverTrigger asChild>
                  <Button variant="outline" role="combobox" :aria-expanded="openPluginPopover"
                    class="justify-between flex-1 h-8 text-sm font-normal">
                    <div class="flex items-center gap-2 truncate">
                      <div v-if="form.language && getLangIcon(form.language)"
                        class="w-4 h-4 shrink-0 rounded-sm bg-white p-0.5 border">
                        <img :src="getLangIcon(form.language)" class="w-full h-full object-contain" />
                      </div>
                      <span>{{ form.language || "选择环境..." }}</span>
                    </div>
                    <ChevronsUpDown class="ml-2 h-3.5 w-3.5 shrink-0 opacity-50" />
                  </Button>
                </PopoverTrigger>
                <PopoverContent class="p-0 w-[240px]" align="start">
                  <div class="p-2 border-b">
                    <div class="relative">
                      <Search class="absolute left-2 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-muted-foreground" />
                      <Input v-model="pluginSearch" placeholder="搜索已安装语言..." class="h-7 pl-8 text-xs" />
                    </div>
                  </div>
                  <ScrollArea class="h-48">
                    <div class="p-1">
                      <div v-if="loadingLangs" class="flex items-center justify-center py-4">
                        <Loader2 class="h-4 w-4 animate-spin text-muted-foreground" />
                      </div>
                      <div v-else-if="filteredPlugins.length === 0"
                        class="py-4 text-center text-xs text-muted-foreground">
                        未找到已安装语言
                      </div>
                      <button v-else v-for="p in filteredPlugins" :key="p"
                        @click="() => { form.language = p; openPluginPopover = false }"
                        class="w-full flex items-center px-2 py-1.5 text-xs rounded-sm hover:bg-muted text-left transition-colors group">
                        <div class="mr-2 h-4 w-4 shrink-0 flex items-center justify-center relative">
                          <div v-if="getLangIcon(p)"
                            class="w-full h-full rounded-sm bg-white overflow-hidden p-0.5 border">
                            <img :src="getLangIcon(p)" class="w-full h-full object-contain" />
                          </div>
                          <div v-else
                            class="w-full h-full flex items-center justify-center bg-primary/10 rounded-sm text-[8px] font-bold uppercase border">
                            {{ p.substring(0, 2) }}
                          </div>
                          <Check v-if="form.language === p"
                            class="absolute -right-2 -top-1 h-3 w-3 text-primary bg-background rounded-full border shadow-sm" />
                        </div>
                        <span :class="{ 'font-bold text-primary': form.language === p }">{{ p }}</span>
                      </button>
                    </div>
                  </ScrollArea>
                </PopoverContent>
              </Popover>

              <Popover v-model:open="openVersionPopover">
                <PopoverTrigger asChild :disabled="!form.language">
                  <Button variant="outline" role="combobox" :aria-expanded="openVersionPopover"
                    class="justify-between w-32 h-8 text-sm font-normal" :disabled="!form.language">
                    <span class="truncate">{{ form.lang_version || "选择版本..." }}</span>
                    <div class="flex items-center">
                      <ChevronsUpDown class="h-3.5 w-3.5 shrink-0 opacity-50" />
                    </div>
                  </Button>
                </PopoverTrigger>
                <PopoverContent class="p-0 w-[140px]" align="start">
                  <div class="p-2 border-b">
                    <div class="relative">
                      <Search class="absolute left-2 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-muted-foreground" />
                      <Input v-model="versionSearch" placeholder="搜索版本..." class="h-7 pl-8 text-xs" />
                    </div>
                  </div>
                  <ScrollArea class="h-48">
                    <div class="p-1">
                      <div v-if="filteredVersions.length === 0" class="py-4 text-center text-xs text-muted-foreground">
                        无可用版本
                      </div>
                      <button v-else v-for="v in filteredVersions" :key="v"
                        @click="() => { form.lang_version = v; openVersionPopover = false }"
                        class="w-full flex items-center px-2 py-1.5 text-xs rounded-sm hover:bg-muted text-left transition-colors">
                        <Check :class="cn('mr-2 h-3 w-3', form.lang_version === v ? 'opacity-100' : 'opacity-0')" />
                        <span class="truncate">{{ v }}</span>
                      </button>
                    </div>
                  </ScrollArea>
                </PopoverContent>
              </Popover>
            </div>
          </div>
        </template>
        <div class="grid grid-cols-1 sm:grid-cols-4 items-center gap-2 sm:gap-3">
          <Label class="sm:text-right text-sm">执行命令</Label>
          <Input v-model="form.command" placeholder="node script.js" class="sm:col-span-3 h-8 text-sm font-mono" />
        </div>
        <div class="grid grid-cols-1 sm:grid-cols-4 items-center gap-2 sm:gap-3">
          <Label class="sm:text-right text-sm">工作目录</Label>
          <div class="sm:col-span-3">
            <DirTreeSelect v-if="selectedAgentId === 'local'" v-model="currentWorkDir" />
            <Input v-else v-model="currentWorkDir" placeholder="工作目录（可选）" class="h-8 text-sm" />
          </div>
        </div>
        <div class="grid grid-cols-1 sm:grid-cols-4 items-center gap-2 sm:gap-3">
          <Label class="sm:text-right text-sm">定时规则</Label>
          <Input v-model="form.schedule" placeholder="0 * * * * *" class="sm:col-span-3 h-8 text-sm font-mono" />
        </div>
        <div class="grid grid-cols-1 sm:grid-cols-4 items-start gap-2 sm:gap-3">
          <span></span>
          <div class="sm:col-span-3">
            <p class="text-xs text-muted-foreground mb-1.5">格式: 秒 分 时 日 月 周</p>
            <div class="flex flex-wrap gap-1">
              <span v-for="preset in cronPresets" :key="preset.value"
                class="px-1.5 py-0.5 text-xs rounded bg-muted hover:bg-accent cursor-pointer transition-colors"
                @click="form.schedule = preset.value">
                {{ preset.label }}
              </span>
            </div>
          </div>
        </div>
        <div class="grid grid-cols-1 sm:grid-cols-4 items-center gap-2 sm:gap-3">
          <Label class="sm:text-right text-sm">超时/清理</Label>
          <div class="sm:col-span-3 flex flex-wrap items-center gap-2">
            <div class="flex items-center gap-1.5">
              <Input v-model.number="form.timeout" type="number" placeholder="30" class="w-20 h-9 text-sm" />
              <span class="text-sm text-muted-foreground whitespace-nowrap">分钟</span>
            </div>
            <div class="flex items-center gap-1.5">
              <Select :model-value="cleanType" @update:model-value="(v) => cleanType = String(v || 'none')">
                <SelectTrigger class="w-24 h-9 text-sm">
                  <SelectValue placeholder="不清理" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="none">不清理</SelectItem>
                  <SelectItem value="day">按天数</SelectItem>
                  <SelectItem value="count">按条数</SelectItem>
                </SelectContent>
              </Select>
              <Input v-if="cleanType && cleanType !== 'none'" v-model.number="cleanKeep" type="number"
                :placeholder="cleanType === 'day' ? '7' : '100'" class="w-20 h-9 text-sm" />
            </div>
          </div>
        </div>
        <div class="grid grid-cols-1 sm:grid-cols-4 items-start gap-2 sm:gap-3">
          <Label class="sm:text-right text-sm pt-2">并发控制</Label>
          <div class="sm:col-span-3 space-y-1.5">
            <div class="flex items-center gap-2">
              <Switch v-model="concurrencyEnabled" />
              <span class="text-sm text-muted-foreground">允许并发</span>
            </div>
            <p class="text-xs text-muted-foreground">如果任务未执行完成，是否允许再次执行</p>
          </div>
        </div>
        <div class="grid grid-cols-1 sm:grid-cols-4 items-start gap-2 sm:gap-3">
          <Label class="sm:text-right text-sm pt-1.5">环境变量</Label>
          <div class="sm:col-span-3 space-y-1.5">
            <Popover>
              <PopoverTrigger as-child>
                <Button variant="outline" class="w-full justify-between font-normal h-8 text-sm">
                  <span class="text-muted-foreground text-xs">搜索并添加环境变量...</span>
                  <ChevronDown class="h-3.5 w-3.5 shrink-0 opacity-50" />
                </Button>
              </PopoverTrigger>
              <PopoverContent class="w-[260px] p-2" align="start">
                <Input v-model="envSearchQuery" placeholder="搜索环境变量..." class="mb-2 h-7 text-sm" />
                <div v-if="filteredEnvVars.length === 0" class="text-xs text-muted-foreground text-center py-2">
                  {{ allEnvVars.length === 0 ? '暂无环境变量' : '无匹配结果' }}
                </div>
                <div v-else class="max-h-[140px] overflow-y-auto space-y-0.5">
                  <div v-for="env in filteredEnvVars" :key="env.id"
                    class="flex items-center gap-2 px-2 py-1 rounded hover:bg-muted cursor-pointer text-xs"
                    @click="addEnv(env.id)">
                    <Plus class="h-3 w-3 text-muted-foreground" />
                    <span class="truncate">{{ env.name }}</span>
                  </div>
                </div>
              </PopoverContent>
            </Popover>
            <div v-if="selectedEnvs.length > 0" class="flex flex-wrap gap-1">
              <div v-for="env in selectedEnvs" :key="env.id"
                class="inline-flex items-center gap-0.5 px-2 py-0.5 text-xs h-5 rounded-full bg-secondary text-secondary-foreground">
                <span>{{ env.name }}</span>
                <X class="h-2.5 w-2.5 cursor-pointer hover:text-destructive" @click="removeEnv(env.id)" />
              </div>
            </div>
          </div>
        </div>
      </div>
      <DialogFooter>
        <Button variant="outline" size="sm" @click="emit('update:open', false)">取消</Button>
        <Button size="sm" @click="save">保存</Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>
