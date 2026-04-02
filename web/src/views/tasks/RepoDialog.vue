<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import { Checkbox } from '@/components/ui/checkbox'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'
import DirTreeSelect from '@/components/DirTreeSelect.vue'
import { X, Globe, GitBranch, Shield, Zap, Clock, Download, Plus, Search, Check, ChevronsUpDown, Loader2, AlertCircle } from 'lucide-vue-next'
import { api, type Task, type RepoConfig, type Agent, type MiseLanguage } from '@/api'
import { toast } from 'vue-sonner'
import { cn } from '@/lib/utils'
import { getCronDescription } from '@/utils/cron'
import TaskNotificationConfig from './components/TaskNotificationConfig.vue'

const notificationConfigRef = ref<InstanceType<typeof TaskNotificationConfig> | null>(null)

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

const proxyOptions = [
  { label: '不使用代理', value: 'none' },
  { label: 'ghproxy.com', value: 'ghproxy' },
  { label: 'mirror.ghproxy.com', value: 'mirror' },
  { label: '自定义代理', value: 'custom' },
]

const form = ref<Partial<Task>>({})
const repoConfig = ref<RepoConfig>({
  source_type: 'git',
  source_url: '',
  target_path: '',
  branch: '',
  sparse_path: '',
  single_file: false,
  proxy_url: '',
  auth_token: '',
  whitelist_paths: '',
  blacklist: '',
  dependence: '',
  extensions: '',
  auto_add_cron: false,
  concurrency: 1,
  repo_source: '',
  proxy: ''
})
const cleanType = ref('none')
const cleanKeep = ref(30)
const allAgents = ref<Agent[]>([])
const selectedAgentId = ref<string>('local')
const tagInput = ref('')

const autoAddCron = computed({
  get: () => !!repoConfig.value.auto_add_cron,
  set: (val: boolean) => {
    repoConfig.value.auto_add_cron = val
  }
})

// === 语言环境相关 ===
const installedLangs = ref<MiseLanguage[]>([])
const loadingLangs = ref(false)
const selectedLangs = ref<{ name: string; version: string; availableVersions: string[] }[]>([])
const availablePlugins = ref<string[]>([])
const pluginSearch = ref('')
const versionSearch = ref('')

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

const showQlImportDialog = ref(false)
const qlCommandInput = ref('')

function importFromQl() {
  qlCommandInput.value = ''
  showQlImportDialog.value = true
}

function submitQlImport() {
  const s = qlCommandInput.value.trim()
  if (!s) {
    showQlImportDialog.value = false
    return
  }
  if (!s.startsWith('ql repo')) {
    toast.error('无效的指令：必须以 ql repo 开头')
    return
  }

  // Parse arguments handling quotes
  const args: string[] = []
  const regex = /[^\s"']+|"([^"]*)"|'([^']*)'/g
  let match
  while ((match = regex.exec(s)) !== null) {
    args.push(match[1] || match[2] || match[0])
  }
  
  if (args[2]) {
    repoConfig.value.source_url = args[2]
    repoConfig.value.source_type = 'git'
    // form task name
    let name = '同步 '
    try {
      const urlPaths = args[2]?.split('/')
      if (urlPaths && urlPaths.length > 0) {
        name += urlPaths[urlPaths.length - 1]?.replace('.git', '') || ''
      } else {
        name += '未命名仓库'
      }
    } catch {
      name += '未命名仓库'
    }
    form.value.name = name
  }
  
  if (args[3]) repoConfig.value.whitelist_paths = args[3]
  if (args[4]) repoConfig.value.blacklist = args[4]
  if (args[5]) repoConfig.value.dependence = args[5]
  if (args[6]) repoConfig.value.branch = args[6]
  if (args[7]) repoConfig.value.extensions = args[7]
  
  repoConfig.value.auto_add_cron = true
  repoConfig.value.repo_source = 'ql'
  toast.success('指令解析成功，已开启自动添加任务，请继续完善其他设置')
  showQlImportDialog.value = false
}

const cronDescription = computed(() => {
  if (!form.value.schedule) return ''
  return getCronDescription(form.value.schedule, (navigator as any).language)
})

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


const concurrencyEnabled = computed({
  get: () => repoConfig.value.concurrency === 1,
  set: (val: boolean) => {
    repoConfig.value.concurrency = val ? 1 : 0
  }
})

function onConcurrencyChange(val: boolean) {
  concurrencyEnabled.value = val
}

const isSingleFile = computed({
  get: () => !!repoConfig.value.single_file,
  set: (val: boolean) => {
    repoConfig.value.single_file = val
  }
})

const cleanConfig = computed(() => {
  if (!cleanType.value || cleanType.value === 'none' || cleanKeep.value <= 0) return ''
  return JSON.stringify({ type: cleanType.value, keep: cleanKeep.value })
})

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
    // 解析仓库配置
    // 解析仓库配置
    const defaultConfig: RepoConfig = {
      source_type: 'git',
      source_url: '',
      target_path: '',
      branch: '',
      sparse_path: '',
      single_file: false,
      proxy: 'none',
      proxy_url: '',
      auth_token: '',
      whitelist_paths: '',
      blacklist: '',
      dependence: '',
      extensions: '',
      auto_add_cron: false,
      concurrency: 1,
      repo_source: ''
    }
    const configStr = props.task?.config
    if (configStr) {
      try {
        const parsed = JSON.parse(configStr)
        // 兼容旧字段: 优先使用 $task_concurrency, 若无则默认 1
        let concurrency = 1
        if (parsed['$task_concurrency'] !== undefined) {
          concurrency = parsed['$task_concurrency'] === 1 ? 1 : 0
        }
        repoConfig.value = { ...defaultConfig, ...parsed, concurrency }
      } catch {
        repoConfig.value = defaultConfig
      }
    } else {
      repoConfig.value = defaultConfig
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
    
    // 仓库任务暂时仅支持本地执行
    selectedAgentId.value = 'local'
    // 加载 Agent 列表
    await loadAgents()
    if (selectedAgentId.value === 'local') {
      await fetchInstalledLangs()
      selectedLangs.value.forEach((lang: { name: string; version: string; availableVersions: string[] }) => {
        updateAvailableVersions(lang)
      })
    }
    // 加载通知配置
    await notificationConfigRef.value?.loadConfig(props.isEdit ? props.task?.id : undefined)
  }
})

async function loadAgents() {
  try {
    allAgents.value = await api.agents.list()
  } catch { /* ignore */ }
}

async function save() {
  if (repoConfig.value.auto_add_cron) {
    if (selectedLangs.value.length === 0 || !selectedLangs.value[0]?.name) {
      toast.error('您开启了“自动添加任务”，请先至少添加并选择一个运行语言环境和版本')
      return
    }
  }

  try {
    form.value.clean_config = cleanConfig.value
    form.value.type = 'repo'
    // 确保 concurrency 字段被正确保存到 config 中
    // 注意：我们将 concurrency 存储在 config 的 $task_concurrency 字段中
    // 同时也保留在 repoConfig 对象中以便回显
    const configToSave: any = {
      ...repoConfig.value,
      '$task_concurrency': concurrencyEnabled.value ? 1 : 0
    }

    // 保存语言环境
    form.value.languages = selectedLangs.value.map((l: { name: string; version: string }) => ({
      name: l.name,
      version: l.version
    }))

    form.value.config = JSON.stringify(configToSave)
    form.value.command = `[${repoConfig.value.source_type}] ${repoConfig.value.source_url}`
    form.value.agent_id = selectedAgentId.value === 'local' ? null : selectedAgentId.value
    if (props.isEdit && form.value.id) {
      await api.tasks.update(form.value.id, form.value)
      await notificationConfigRef.value?.saveConfig(form.value.id)
      toast.success('同步任务已更新')
    } else {
      const task = await api.tasks.create(form.value)
      await notificationConfigRef.value?.saveConfig(task.id)
      toast.success('同步任务已创建')
    }
    emit('update:open', false)
    emit('saved')
  } catch { toast.error('保存失败') }
}
</script>

<template>
  <Dialog :open="open" @update:open="emit('update:open', $event)">
    <DialogContent class="max-w-[95vw] sm:max-w-[700px] xl:max-w-[950px] p-0 overflow-hidden border-none bg-background shadow-2xl transition-all duration-300" style="text-rendering: optimizeLegibility;" @openAutoFocus.prevent>
      <div class="flex flex-col max-h-[85vh]">
        <DialogHeader class="px-6 pr-12 pt-6 pb-2 shrink-0">
          <div class="flex items-center justify-between">
            <DialogTitle class="text-xl font-bold">
              {{ isEdit ? '编辑仓库同步' : '新建仓库同步' }}
            </DialogTitle>
            <Button v-if="!isEdit" variant="outline" size="sm" @click="importFromQl" class="h-8 gap-1.5 bg-primary/5 hover:bg-primary/10 border-primary/20 hover:border-primary/40 text-primary">
              <Download class="w-3.5 h-3.5" />
              青龙格式导入
            </Button>
          </div>
        </DialogHeader>

        <ScrollArea class="flex-1 min-h-0 px-6">
          <div class="space-y-8 py-4 pb-8">
            <!-- 基本信息 Section -->
            <section class="space-y-4">
              <div class="flex items-center gap-2 mb-1">
                <div class="h-4 w-1 bg-primary rounded-full" />
                <h3 class="text-sm font-bold text-foreground">基本信息</h3>
              </div>

              <div class="grid gap-4 pl-3 border-l border-muted">
                <div class="grid grid-cols-1 sm:grid-cols-4 items-center gap-3">
                  <Label class="sm:text-right text-xs text-foreground/70 uppercase tracking-wider font-bold">任务名称</Label>
                  <Input v-model="form.name" placeholder="输入同步任务名称" class="sm:col-span-3 h-9 bg-muted/30 border-muted-foreground/20 focus:bg-background transition-all" />
                </div>
                <div class="grid grid-cols-1 sm:grid-cols-4 items-center gap-3">
                  <Label class="sm:text-right text-xs text-foreground/70 uppercase tracking-wider font-bold">任务备注</Label>
                  <Input v-model="form.remark" placeholder="输入同步任务备注" class="sm:col-span-3 h-9 bg-muted/30 border-muted-foreground/20 focus:bg-background transition-all" />
                </div>

                <div class="grid grid-cols-1 sm:grid-cols-4 items-start gap-3">
                  <Label class="sm:text-right text-xs text-muted-foreground uppercase tracking-wider pt-2.5">任务标签</Label>
                  <div class="sm:col-span-3 space-y-2">
                    <div class="flex gap-2">
                      <div class="relative flex-1">
                        <Input v-model="tagInput" placeholder="输入标签按回车..." class="h-9 bg-muted/30 border-muted-foreground/20 pr-12" @keydown.enter.prevent="addTag" />
                        <Button type="button" variant="ghost" size="sm" class="absolute right-1 top-1 h-7 px-2 text-xs hover:bg-primary/10 hover:text-primary transition-colors" @click="addTag">
                          添加
                        </Button>
                      </div>
                    </div>
                    <div v-if="form.tags" class="flex flex-wrap gap-1.5 pt-1">
                      <span v-for="tag in form.tags.split(',').filter(Boolean)" :key="tag" 
                        class="flex items-center gap-1.5 bg-primary/5 text-primary px-2.5 py-1 rounded-full text-[11px] font-medium border border-primary/10 group transition-all hover:bg-primary/10">
                        {{ tag }}
                        <button type="button" class="text-primary/40 hover:text-destructive transition-colors shrink-0" @click.prevent="removeTag(tag)">
                          <X class="h-3 w-3" />
                        </button>
                      </span>
                    </div>
                  </div>
                </div>
              </div>
            </section>

            <!-- 仓库配置 Section -->
            <section class="space-y-4">
              <div class="flex items-center gap-2 mb-1">
                <div class="h-4 w-1 bg-primary rounded-full" />
                <h3 class="text-sm font-bold text-foreground">核心配置</h3>
              </div>

              <div class="grid gap-4 pl-3 border-l border-muted">
                <div class="grid grid-cols-1 sm:grid-cols-4 items-center gap-3">
                  <Label class="sm:text-right text-xs text-foreground/70 uppercase tracking-wider font-bold">源类型</Label>
                  <div class="sm:col-span-3">
                    <Select :model-value="repoConfig.source_type" @update:model-value="(v: any) => repoConfig.source_type = String(v || 'git')">
                      <SelectTrigger class="h-9 bg-muted/30 border-muted-foreground/20">
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="git">
                          <div class="flex items-center gap-2">
                            <GitBranch class="h-3.5 w-3.5" />
                            <span>Git 仓库 (Repository)</span>
                          </div>
                        </SelectItem>
                        <SelectItem value="url">
                          <div class="flex items-center gap-2">
                            <Globe class="h-3.5 w-3.5" />
                            <span>URL 下载 (Direct Link)</span>
                          </div>
                        </SelectItem>
                      </SelectContent>
                    </Select>
                  </div>
                </div>

                <div class="grid grid-cols-1 sm:grid-cols-4 items-center gap-3">
                  <Label class="sm:text-right text-xs text-foreground/70 uppercase tracking-wider font-bold">源地址</Label>
                  <div class="sm:col-span-3 relative">
                    <Input v-model="repoConfig.source_url"
                      :placeholder="repoConfig.source_type === 'git' ? 'https://github.com/user/repo.git' : 'https://example.com/file.js'"
                      class="h-9 font-mono text-[13px] bg-muted/30 border-muted-foreground/20 focus:bg-background pr-10 transition-all" 
                      autocomplete="off" />
                    <Globe class="absolute right-3 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-muted-foreground opacity-40" />
                  </div>
                </div>

                <div class="grid grid-cols-1 sm:grid-cols-4 items-center gap-3">
                  <Label class="sm:text-right text-xs text-foreground/70 uppercase tracking-wider font-bold">目标路径</Label>
                  <div class="sm:col-span-3">
                    <DirTreeSelect v-if="selectedAgentId === 'local'" :model-value="repoConfig.target_path || ''"
                      @update:model-value="v => repoConfig.target_path = v" class="h-9" />
                    <Input v-else v-model="repoConfig.target_path" placeholder="Agent 上的目标路径" class="h-9 bg-muted/30 border-muted-foreground/20" />
                  </div>
                </div>
                <div v-if="repoConfig.source_type === 'git'" class="grid grid-cols-1 sm:grid-cols-4 items-center gap-3">
                  <Label class="sm:text-right text-xs text-foreground/70 uppercase tracking-wider font-medium">分支</Label>
                  <Input v-model="repoConfig.branch" placeholder="main (默认)" class="sm:col-span-3 h-9 bg-muted/30 border-muted-foreground/20 focus:bg-background transition-all" autocomplete="off" />
                </div>

                <div v-if="repoConfig.source_type === 'git'" class="grid grid-cols-1 sm:grid-cols-4 items-center gap-3">
                  <Label class="sm:text-right text-xs text-foreground/70 uppercase tracking-wider font-medium">稀疏路径</Label>
                  <Input v-model="repoConfig.sparse_path" placeholder="指定目录或文件 (可选)" class="sm:col-span-3 h-9 bg-muted/30 border-muted-foreground/20 focus:bg-background transition-all" autocomplete="off" />
                </div>

                <div v-if="repoConfig.source_type === 'git' && repoConfig.sparse_path" class="grid grid-cols-1 sm:grid-cols-4 items-center gap-3">
                  <Label class="sm:text-right text-xs text-foreground/70 uppercase tracking-wider font-medium">下载模式</Label>
                  <div class="sm:col-span-3">
                    <div class="flex items-center space-x-2 bg-muted/20 px-3 py-1.5 rounded-full border border-muted-foreground/10 w-fit">
                      <Checkbox id="single-file-sync" v-model="isSingleFile" class="scale-90" />
                      <Label for="single-file-sync" class="text-[11px] font-medium cursor-pointer">作为单文件直接下载</Label>
                    </div>
                  </div>
                </div>
              </div>
            </section>

            <!-- 访问策略 Section -->
            <section class="space-y-4">
              <div class="flex items-center gap-2 mb-1">
                <div class="h-4 w-1 bg-primary rounded-full" />
                <h3 class="text-sm font-bold text-foreground">访问控制</h3>
              </div>

              <div class="grid gap-4 pl-3 border-l border-muted">
                <div class="grid grid-cols-1 sm:grid-cols-4 items-center gap-3">
                  <Label class="sm:text-right text-xs text-foreground/70 uppercase tracking-wider font-medium">代理配置</Label>
                  <div class="sm:col-span-3">
                    <Select :model-value="repoConfig.proxy" @update:model-value="(v: any) => repoConfig.proxy = String(v || 'none')">
                      <SelectTrigger class="h-9 bg-muted/30 border-muted-foreground/20">
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem v-for="opt in proxyOptions" :key="opt.value" :value="opt.value">{{ opt.label }}</SelectItem>
                      </SelectContent>
                    </Select>
                  </div>
                </div>

                <div v-if="repoConfig.proxy === 'custom'" class="grid grid-cols-1 sm:grid-cols-4 items-center gap-3">
                  <Label class="sm:text-right text-xs text-foreground/70 uppercase tracking-wider font-medium">代理地址</Label>
                  <Input v-model="repoConfig.proxy_url" placeholder="https://your-proxy.com" class="sm:col-span-3 h-9 bg-muted/30 font-mono text-xs border-muted-foreground/20 focus:bg-background transition-all" autocomplete="off" />
                </div>

                <div class="grid grid-cols-1 sm:grid-cols-4 items-center gap-3">
                  <Label class="sm:text-right text-xs text-foreground/70 uppercase tracking-wider font-medium">身份认证</Label>
                  <div class="sm:col-span-3 relative">
                    <Input v-model="repoConfig.auth_token" type="password" placeholder="推荐使用 Token 替代密码" class="h-9 bg-muted/30 border-muted-foreground/20 pr-10 text-xs focus:bg-background transition-all" autocomplete="new-password" />
                    <Shield class="absolute right-3 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-muted-foreground opacity-40" />
                  </div>
                </div>
              </div>
            </section>

            <!-- 脚本过滤 Section -->
            <section class="space-y-4">
              <div class="flex items-center gap-2 mb-1">
                <div class="h-4 w-1 bg-primary rounded-full" />
                <h3 class="text-sm font-bold text-foreground">脚本过滤</h3>
              </div>

              <div class="grid gap-4 pl-3 border-l border-muted">
                <div class="grid grid-cols-1 sm:grid-cols-4 items-center gap-3">
                  <Label class="sm:text-right text-xs text-foreground/70 uppercase tracking-wider font-medium">白名单</Label>
                  <div class="sm:col-span-3 relative">
                    <Input v-model="repoConfig.whitelist_paths" placeholder="保活路径或脚本关键词 (如: logs/ | jd_ )" class="h-9 bg-muted/30 border-muted-foreground/20 focus:bg-background transition-all" autocomplete="off" />
                    <p class="text-[10px] text-muted-foreground mt-1 px-1 leading-relaxed">请输入脚本筛选白名单关键词或保活路径（支持 *），多个关键词或路径使用竖线(|)或逗号(,)分割</p>
                  </div>
                </div>

                <div class="grid grid-cols-1 sm:grid-cols-4 items-center gap-3">
                  <Label class="sm:text-right text-xs text-foreground/70 uppercase tracking-wider font-medium">脚本黑名单</Label>
                  <div class="sm:col-span-3 relative">
                    <Input v-model="repoConfig.blacklist" placeholder="黑名单关键词 (如: help)" class="h-9 bg-muted/30 border-muted-foreground/20 focus:bg-background transition-all" autocomplete="off" />
                    <p class="text-[10px] text-muted-foreground mt-1 px-1">脚本筛选黑名单关键词，多个关键词竖线(|)分割</p>
                  </div>
                </div>

                <div class="grid grid-cols-1 sm:grid-cols-4 items-center gap-3">
                  <Label class="sm:text-right text-xs text-foreground/70 uppercase tracking-wider font-medium">依赖文件</Label>
                  <div class="sm:col-span-3 relative">
                    <Input v-model="repoConfig.dependence" placeholder="依赖文件关键词 (如: ccav | notify)" class="h-9 bg-muted/30 border-muted-foreground/20 focus:bg-background transition-all" autocomplete="off" />
                    <p class="text-[10px] text-muted-foreground mt-1 px-1">脚本依赖文件关键词，多个关键词竖线(|)分割</p>
                  </div>
                </div>

                <div class="grid grid-cols-1 sm:grid-cols-4 items-center gap-3">
                  <Label class="sm:text-right text-xs text-foreground/70 uppercase tracking-wider font-medium">文件后缀</Label>
                  <div class="sm:col-span-3 relative">
                    <Input v-model="repoConfig.extensions" placeholder="文件后缀 (如: js | py | sh)" class="h-9 bg-muted/30 border-muted-foreground/20 focus:bg-background transition-all" autocomplete="off" />
                    <p class="text-[10px] text-muted-foreground mt-1 px-1">脚本文件后缀，多个后缀竖线(|)分割</p>
                  </div>
                </div>
              </div>
            </section>

            <!-- 运行环境 Section -->
            <section v-if="selectedAgentId === 'local'" class="space-y-4">
              <div class="flex items-center gap-2 mb-1">
                <div class="h-4 w-1 bg-primary rounded-full" />
                <h3 class="text-sm font-bold text-foreground">运行环境</h3>
              </div>

              <div class="grid gap-4 pl-3 border-l border-muted">
                <div class="grid grid-cols-1 sm:grid-cols-4 items-start gap-3 mt-2">
                  <Label class="sm:text-right text-xs text-muted-foreground uppercase tracking-wider pt-2.5">语言环境</Label>
                  <div class="sm:col-span-3 space-y-2">
                    <div class="flex items-start gap-2.5 p-3 rounded-xl bg-amber-500/5 border border-amber-500/10 text-amber-600 dark:text-amber-400 text-[11px] leading-relaxed mb-2">
                      <AlertCircle class="h-4 w-4 shrink-0 text-amber-500 mt-0.5" />
                      <p>同步后生成的任务将自动继承此运行环境。如果不指定语言版本，某些依赖特定语言的脚本（如 js, py）将无法顺利解析和运行！</p>
                    </div>

                    <div v-for="(clang, idx) in selectedLangs" :key="idx" 
                      class="flex gap-2 p-2 rounded-lg bg-muted/20 border border-muted-foreground/10 group/lang relative overflow-hidden">
                      <div class="absolute left-0 top-0 bottom-0 w-0.5 bg-primary/20 group-hover/lang:bg-primary transition-colors" />
                      <Popover>
                        <PopoverTrigger asChild>
                          <Button variant="ghost" role="combobox" class="justify-between flex-1 h-8 text-xs font-normal hover:bg-background/50">
                            <div class="flex items-center gap-2 truncate">
                              <div v-if="clang.name && getLangIcon(clang.name)" class="w-4 h-4 shrink-0 rounded-sm bg-white p-0.5 border shadow-sm">
                                <img :src="getLangIcon(clang.name)" class="w-full h-full object-contain" />
                              </div>
                              <span class="font-medium">{{ clang.name || "选择插件..." }}</span>
                            </div>
                            <ChevronsUpDown class="ml-1 h-3 w-3 opacity-40" />
                          </Button>
                        </PopoverTrigger>
                        <PopoverContent class="p-0 w-[240px]" align="start">
                          <div class="p-2 border-b bg-muted/30">
                            <div class="relative">
                              <Search class="absolute left-2 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-muted-foreground" />
                              <Input v-model="pluginSearch" placeholder="搜索已安装语言..." class="h-8 pl-8 text-xs bg-background" />
                            </div>
                          </div>
                          <ScrollArea class="h-48 p-1">
                            <div v-if="loadingLangs" class="flex items-center justify-center py-6">
                              <Loader2 class="h-5 w-5 animate-spin text-primary/50" />
                            </div>
                            <div v-else-if="filteredPlugins.length === 0" class="py-6 text-center text-xs text-muted-foreground">
                              未找到匹配项
                            </div>
                            <button v-else v-for="p in filteredPlugins" :key="p" @click="updateLangName(idx, p)"
                              class="w-full flex items-center px-3 py-2 text-xs rounded-md hover:bg-accent text-left transition-all group/item mb-0.5">
                              <div class="mr-3 h-5 w-5 shrink-0 flex items-center justify-center transition-transform group-hover/item:scale-110">
                                <img v-if="getLangIcon(p)" :src="getLangIcon(p)" class="w-full h-full object-contain p-0.5 bg-white rounded border" />
                                <div v-else class="w-full h-full flex items-center justify-center bg-primary/10 rounded-sm text-[8px] font-bold border">
                                  {{ p.substring(0, 2) }}
                                </div>
                              </div>
                              <span class="flex-1" :class="{ 'font-bold text-primary': clang.name === p }">{{ p }}</span>
                              <Check v-if="clang.name === p" class="h-3 w-3 text-primary" />
                            </button>
                          </ScrollArea>
                        </PopoverContent>
                      </Popover>

                      <Popover>
                        <PopoverTrigger asChild :disabled="!clang.name">
                          <Button variant="ghost" role="combobox" class="justify-between w-28 h-8 text-xs font-normal hover:bg-background/50" :disabled="!clang.name">
                            <span class="truncate">{{ clang.version || "版本..." }}</span>
                            <ChevronsUpDown class="h-3 w-3 opacity-40 ml-1" />
                          </Button>
                        </PopoverTrigger>
                        <PopoverContent class="p-0 w-[160px]" align="start">
                          <div class="p-2 border-b bg-muted/30">
                            <div class="relative">
                              <Search class="absolute left-2 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-muted-foreground" />
                              <Input v-model="versionSearch" placeholder="搜索版本..." class="h-8 pl-8 text-xs bg-background" />
                            </div>
                          </div>
                          <ScrollArea class="h-48 p-1">
                            <div v-if="getFilteredVersions(clang.availableVersions).length === 0" class="py-6 text-center text-xs text-muted-foreground">
                              无可用版本
                            </div>
                            <button v-else v-for="v in getFilteredVersions(clang.availableVersions)" :key="v" @click="clang.version = v"
                              class="w-full flex items-center px-3 py-2 text-xs rounded-md hover:bg-accent text-left mb-0.5 font-mono">
                              <span class="flex-1 truncate" :class="{ 'font-bold text-primary': clang.version === v }">{{ v }}</span>
                              <Check v-if="clang.version === v" class="h-3 w-3 text-primary" />
                            </button>
                          </ScrollArea>
                        </PopoverContent>
                      </Popover>

                      <Button variant="ghost" size="icon" class="h-8 w-8 text-muted-foreground hover:text-destructive hover:bg-destructive/10 shrink-0"
                        @click="removeLang(idx)">
                        <X class="h-4 w-4" />
                      </Button>
                    </div>

                    <Button variant="outline" size="sm" class="w-full h-9 text-xs border-dashed border-muted-foreground/30 text-muted-foreground hover:text-primary hover:border-primary/50 transition-all bg-muted/10 hover:bg-primary/5"
                      @click="addLang">
                      <Plus class="h-4 w-4 mr-2" /> 必须添加运行语言和版本
                    </Button>
                  </div>
                </div>
              </div>
            </section>

            <!-- 调度策略 Section -->
            <section class="space-y-4">
              <div class="flex items-center gap-2 mb-1">
                <div class="h-4 w-1 bg-primary rounded-full shadow-sm shadow-primary/20" />
                <h3 class="text-sm font-bold text-foreground/90">调度策略</h3>
              </div>

              <div class="grid gap-5 pl-3 border-l border-muted">
                <div class="grid grid-cols-1 sm:grid-cols-4 items-center gap-3">
                  <Label class="sm:text-right text-xs text-foreground/70 uppercase tracking-wider font-bold">定时规则</Label>
                  <div class="sm:col-span-3">
                    <Input v-model="form.schedule" placeholder="* * * * * *" class="h-9 font-mono text-[13px] bg-muted/30 border-muted-foreground/20 focus:ring-1 focus:ring-primary/50" />
                    <div v-if="cronDescription" class="mt-2.5 p-2 rounded-lg bg-primary/5 border border-primary/10 text-[11px] text-primary font-medium flex items-center gap-2 animate-in fade-in slide-in-from-top-1 duration-300">
                      <Zap class="h-3 w-3" />
                      {{ cronDescription }}
                    </div>
                    <div class="mt-2.5 space-y-2">
                       <div class="flex items-center gap-1.5 text-[10px] text-muted-foreground/70 uppercase font-bold tracking-tighter">
                          <Clock class="h-3 w-3" /> 格式指导: 秒 分 时 日 月 周
                        </div>
                      <div class="flex flex-wrap gap-1.5">
                        <button v-for="preset in cronPresets" :key="preset.value"
                          class="px-2 py-1 text-[10px] rounded-md bg-muted/50 border border-muted-foreground/10 hover:border-primary/50 hover:bg-primary/5 hover:text-primary transition-all font-medium"
                          @click.prevent="form.schedule = preset.value">
                          {{ preset.label }}
                        </button>
                      </div>
                    </div>
                  </div>
                </div>

                <div class="grid grid-cols-1 sm:grid-cols-4 items-center gap-3">
                  <Label class="sm:text-right text-xs text-foreground/70 uppercase tracking-wider font-bold">随机延迟</Label>
                  <div class="sm:col-span-3 flex items-center gap-4">
                    <div class="flex items-center gap-2">
                      <Input :model-value="form.random_range" @update:model-value="(v: string | number) => form.random_range = Number(v || 0)" type="number" :min="0" class="w-20 h-9 bg-muted/30 text-center" />
                      <span class="text-xs font-semibold text-muted-foreground">秒</span>
                    </div>
                    <div class="flex-1 text-[11px] text-muted-foreground leading-snug p-2 rounded-lg bg-blue-500/5 border border-blue-500/10 italic">
                      避免高频并发，在基准时间点后延迟 0~{{ form.random_range || 0 }}s
                    </div>
                  </div>
                </div>

                <div class="grid grid-cols-1 sm:grid-cols-4 items-start gap-3">
                  <Label class="sm:text-right text-xs text-foreground/70 uppercase tracking-wider font-bold">运行策略</Label>
                  <div class="sm:col-span-3 space-y-4">
                    
                    <div class="p-3 rounded-xl bg-muted/20 border border-muted-foreground/10 space-y-2.5">
                      <div class="flex items-center justify-between">
                        <div class="flex items-center gap-2 text-xs font-semibold">
                          <Zap :class="cn('h-3.5 w-3.5', autoAddCron ? 'text-primary' : 'text-muted-foreground')" /> 
                          自动添加任务
                        </div>
                        <Switch :model-value="autoAddCron" @update:model-value="(v: boolean) => autoAddCron = v" />
                      </div>
                      <p class="text-[11px] text-muted-foreground leading-relaxed">
                        {{ autoAddCron ? '同步完成后将尝试自动分析脚本并注册定时任务。' : '仅拉取脚本，不自动注册成面板任务。' }}
                      </p>
                    </div>

                    <div class="flex items-center gap-4">
                      <div class="flex items-center gap-2">
                         <Input :model-value="form.timeout" @update:model-value="(v: string | number) => form.timeout = Number(v || 0)" type="number" :min="0" class="w-20 h-9 bg-muted/30 text-center" />
                         <span class="text-[11px] font-semibold text-muted-foreground">分钟超时</span>
                      </div>
                      <div class="flex items-center gap-2 pl-4 border-l">
                        <Select :model-value="cleanType" @update:model-value="(v: any) => cleanType = String(v || 'none')">
                          <SelectTrigger class="w-28 h-9 text-xs bg-muted/10">
                            <SelectValue />
                          </SelectTrigger>
                          <SelectContent>
                            <SelectItem value="none">保留日志</SelectItem>
                            <SelectItem value="day">按天清理</SelectItem>
                            <SelectItem value="count">按条清理</SelectItem>
                          </SelectContent>
                        </Select>
                        <Input v-if="cleanType && cleanType !== 'none'" :model-value="cleanKeep" @update:model-value="v => cleanKeep = Number(v || 30)" type="number" class="w-16 h-9 bg-muted/30 text-center text-xs" />
                      </div>
                    </div>

                    <div class="p-3 rounded-xl bg-muted/20 border border-muted-foreground/10 space-y-2.5">
                      <div class="flex items-center justify-between">
                        <div class="flex items-center gap-2 text-xs font-semibold">
                          <Zap :class="cn('h-3.5 w-3.5', concurrencyEnabled ? 'text-primary' : 'text-muted-foreground')" /> 
                          并发控制
                        </div>
                        <Switch :model-value="concurrencyEnabled" @update:model-value="onConcurrencyChange" />
                      </div>
                      <p class="text-[11px] text-muted-foreground leading-relaxed">
                        {{ concurrencyEnabled ? '允许同时开启多个同步副本。' : '当前同步未结束时，新触发将被静默忽略。' }}
                      </p>
                    </div>
                  </div>
                </div>
              </div>
            </section>

            <!-- 通知配置 -->
            <TaskNotificationConfig ref="notificationConfigRef" :task-id="isEdit ? task?.id : undefined" />
          </div>
        </ScrollArea>

        <div class="flex items-center justify-between px-6 py-4 bg-muted/30 border-t shrink-0">
          <p class="text-[10px] text-muted-foreground">最后编辑于: {{ isEdit ? (form.updated_at || '刚才') : '现在' }}</p>
          <div class="flex gap-3">
            <Button variant="ghost" size="sm" class="hover:bg-muted font-medium text-xs px-6" @click="emit('update:open', false)">取消</Button>
            <Button size="sm" class="px-8 font-semibold text-xs shadow-lg shadow-primary/20 transition-all hover:scale-105 active:scale-95 bg-primary hover:bg-primary/90" @click="save">
              确定保存
            </Button>
          </div>
        </div>
      </div>
    </DialogContent>
  </Dialog>

  <!-- 青龙导入提示对话框 -->
  <Dialog :open="showQlImportDialog" @update:open="v => showQlImportDialog = v">
    <DialogContent class="sm:max-w-[425px] p-0 border-none bg-background shadow-2xl">
      <DialogHeader class="px-6 pt-6 pb-2">
        <DialogTitle class="text-xl font-bold bg-clip-text text-transparent bg-gradient-to-r from-foreground to-foreground/70">
          请输入青龙面板的 ql repo 指令
        </DialogTitle>
      </DialogHeader>
      
      <div class="px-6 py-4 space-y-4 text-sm text-muted-foreground leading-relaxed">
        <p>例如：</p>
        <div class="p-2 rounded-md bg-muted/50 font-mono text-xs select-all text-primary/80 break-all border border-muted-foreground/10">
          ql repo "https://github.com/a/b.git" "jd_|jx_" "activity" "^jd[^_]" "main" "js|py"
        </div>
        <div class="relative mt-2">
          <Input v-model="qlCommandInput" placeholder="在此处粘贴完整指令，如 ql repo ..." class="h-10 pr-10 focus:ring-primary/20 bg-muted/20" @keydown.enter.prevent="submitQlImport" />
        </div>
      </div>
      
      <DialogFooter class="px-6 pb-6 pt-2">
        <Button variant="outline" size="sm" @click="showQlImportDialog = false" class="border-border/40 hover:bg-muted/30">
          取消
        </Button>
        <Button size="sm" @click="submitQlImport" class="shadow-sm">
          确定 <Download class="h-3 w-3 ml-1.5" />
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>

<style scoped>
/* 仅针对任务编辑页面的字体渲染优化 */
:deep(*) {
  -webkit-font-smoothing: auto !important;
  -moz-osx-font-smoothing: auto !important;
  letter-spacing: 0 !important;
}

:deep(label), :deep(h3), :deep(input) {
  text-rendering: optimizeLegibility;
}
</style>
<style scoped>
:deep(*) {
  text-rendering: optimizeLegibility;
}
:deep(label) {
  text-rendering: optimizeLegibility;
  letter-spacing: 0.01em;
}
</style>
