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
import DirTreeSelect from '@/components/DirTreeSelect.vue'
import TaskLangConfig from './components/TaskLangConfig.vue'
import { Globe, GitBranch, Shield, Zap, Download, AlertCircle, Terminal } from 'lucide-vue-next'
import { api, type Task, type RepoConfig, type Agent } from '@/api'
import { toast } from 'vue-sonner'
import { cn } from '@/lib/utils'

import { parseBaihuCommand, parseQlCommand } from '@/utils/repo-parser'
import { copyToClipboard } from '@/utils/clipboard'
import TaskNotificationConfig from './components/TaskNotificationConfig.vue'
import TaskAdvancedConfig from './components/TaskAdvancedConfig.vue'
import TaskCronConfig from './components/TaskCronConfig.vue'
import TaskTagsConfig from './components/TaskTagsConfig.vue'

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
  commenttotask: 'false',
  concurrency: 1,
  repo_source: '',
  proxy: '',
  repo_dir_name: ''
})

const allAgents = ref<Agent[]>([])
const selectedAgentId = ref<string>('local')


const autoAddCron = computed({
  get: () => !!repoConfig.value.auto_add_cron,
  set: (val: boolean) => {
    repoConfig.value.auto_add_cron = val
  }
})

const pullQlConfig = computed({
  get: () => repoConfig.value.commenttotask === 'true',
  set: (val: boolean) => {
    repoConfig.value.commenttotask = val ? 'true' : 'false'
  }
})

// === 语言环境相关 ===
const selectedLangs = ref<{ name: string; version: string; availableVersions: string[] }[]>([])

const showQlImportDialog = ref(false)
const qlCommandInput = ref('')

const showBaihuImportDialog = ref(false)
const baihuCommandInput = ref('')

function exportBaihuCommand() {
  const parts = ['baihu', 'reposync']
  if (repoConfig.value.source_type) parts.push(`--source-type ${repoConfig.value.source_type}`)
  if (repoConfig.value.source_url) parts.push(`--source-url "${repoConfig.value.source_url}"`)
  if (repoConfig.value.target_path) parts.push(`--target-path "${repoConfig.value.target_path}"`)
  if (repoConfig.value.branch) parts.push(`--branch "${repoConfig.value.branch}"`)
  if (repoConfig.value.repo_dir_name) parts.push(`--repo-name "${repoConfig.value.repo_dir_name}"`)
  if (repoConfig.value.sparse_path) parts.push(`--path "${repoConfig.value.sparse_path}"`)
  if (repoConfig.value.single_file) parts.push(`--single-file`)
  if (repoConfig.value.proxy && repoConfig.value.proxy !== 'none') parts.push(`--proxy ${repoConfig.value.proxy}`)
  if (repoConfig.value.proxy_url) parts.push(`--proxy-url "${repoConfig.value.proxy_url}"`)
  if (repoConfig.value.auth_token) parts.push(`--auth-token "${repoConfig.value.auth_token}"`)
  if (repoConfig.value.whitelist_paths) parts.push(`--whitelist-paths "${repoConfig.value.whitelist_paths}"`)
  if (repoConfig.value.blacklist) parts.push(`--blacklist "${repoConfig.value.blacklist}"`)
  if (repoConfig.value.dependence) parts.push(`--dependence "${repoConfig.value.dependence}"`)
  if (repoConfig.value.extensions) parts.push(`--extensions "${repoConfig.value.extensions}"`)
  if (repoConfig.value.auto_add_cron) parts.push(`--commenttotask true`)
  
  if (form.value.pre_command) parts.push(`--pre-command "${form.value.pre_command}"`)
  if (form.value.post_command) parts.push(`--post-command "${form.value.post_command}"`)
  if (form.value.timeout !== undefined && form.value.timeout !== 30) parts.push(`--task-timeout ${form.value.timeout}`)
  
  if (selectedLangs.value.length > 0) {
    const langs = selectedLangs.value.filter(l => l.name).map(l => ({ name: l.name, version: l.version }))
    if (langs.length > 0) {
      parts.push(`--task-langs '${JSON.stringify(langs)}'`)
    }
  }

  const cmd = parts.join(' ')
  copyToClipboard(cmd).then((success) => {
    if (success) toast.success('baihu 指令已复制到剪贴板')
  })
}

function importFromBaihu() {
  baihuCommandInput.value = ''
  showBaihuImportDialog.value = true
}

function submitBaihuImport() {
  const result = parseBaihuCommand(baihuCommandInput.value)
  if (!result) {
    if (!baihuCommandInput.value.trim()) {
      showBaihuImportDialog.value = false
    } else {
      toast.error('未识别到有效的 reposync 参数')
    }
    return
  }

  // 应用解析结果
  repoConfig.value = { ...repoConfig.value, ...result.repoConfig }
  if (result.task.name) form.value.name = result.task.name
  if (result.task.timeout) form.value.timeout = result.task.timeout
  if (result.task.pre_command) form.value.pre_command = result.task.pre_command
  if (result.task.post_command) form.value.post_command = result.task.post_command
  
  if (result.task.languages) {
    selectedLangs.value = result.task.languages.map(l => ({
      name: l.name || '',
      version: l.version || '',
      availableVersions: []
    }))
  }

  // toast.success('命令解析成功，已自动填充表单')
  showBaihuImportDialog.value = false
}

function importFromQl() {
  qlCommandInput.value = ''
  showQlImportDialog.value = true
}

function submitQlImport() {
  const result = parseQlCommand(qlCommandInput.value)
  if (!result) {
    if (!qlCommandInput.value.trim()) {
      showQlImportDialog.value = false
    } else {
      toast.error('无效的指令：必须以 ql repo 开头')
    }
    return
  }

  // 应用解析结果
  repoConfig.value = { ...repoConfig.value, ...result.repoConfig }
  if (result.task.name) form.value.name = result.task.name

  toast.success('指令解析成功，已开启自动添加任务，请继续完善其他设置')
  showQlImportDialog.value = false
}





const isSingleFile = computed({
  get: () => !!repoConfig.value.single_file,
  set: (val: boolean) => {
    repoConfig.value.single_file = val
  }
})

watch(() => props.open, async (val: boolean) => {
  if (val) {
    form.value = {
      retry_count: props.task?.retry_count ?? 0,
      retry_interval: props.task?.retry_interval ?? 0,
      random_range: props.task?.random_range ?? 0,
      timeout: props.task?.timeout ?? 30,
      pin_type: props.task?.pin_type ?? 'none',
      pre_command: props.task?.pre_command ?? '',
      post_command: props.task?.post_command ?? '',
      ...props.task
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
      commenttotask: 'false',
      concurrency: 1,
      repo_source: '',
      repo_dir_name: ''
    }
    const configStr = props.task?.config
    if (configStr) {
      try {
        const parsed = JSON.parse(configStr)
        repoConfig.value = { ...defaultConfig, ...parsed }
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
    let existingConfig = {}
    if (form.value.config) {
      try { existingConfig = JSON.parse(form.value.config) } catch {}
    }
    const configToSave: any = {
      ...existingConfig,
      ...repoConfig.value
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
  } catch (error: any) {
    toast.error('保存失败', {
      description: error.response?.data?.error || error.response?.data?.message || error.message || '未知错误'
    })
  }
}
</script>

<template>
  <Dialog :open="open" @update:open="emit('update:open', $event)">
    <DialogContent class="max-w-[95vw] sm:max-w-[700px] xl:max-w-[950px] p-0 overflow-hidden border-none bg-background shadow-2xl transition-all duration-300" style="text-rendering: optimizeLegibility;" @openAutoFocus.prevent>
      <div class="flex flex-col max-h-[85vh]">
        <DialogHeader class="px-5 sm:px-6 pr-20 pt-6 pb-2 shrink-0">
          <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4 sm:gap-2">
            <DialogTitle class="text-xl font-bold whitespace-nowrap">
              {{ isEdit ? '编辑仓库同步' : '新建仓库同步' }}
            </DialogTitle>
            <div class="flex flex-wrap items-center justify-end self-end sm:self-auto gap-2 -mr-14 sm:mr-4">
              <Button v-if="isEdit" variant="outline" size="sm" @click="exportBaihuCommand" title="复制导出 baihu 指令" class="h-8 gap-1.5 bg-primary/5 hover:bg-primary/10 border-primary/20 hover:border-primary/40 text-primary px-3">
                <Terminal class="w-3.5 h-3.5" />
                <span class="text-xs">复制指令</span>
              </Button>
              <template v-else>
              <Button variant="outline" size="sm" @click="importFromBaihu" class="flex-1 sm:flex-initial h-8 gap-1.5 bg-primary/5 hover:bg-primary/10 border-primary/20 hover:border-primary/40 text-primary px-3">
                <Terminal class="w-3.5 h-3.5" />
                <span class="text-xs">Baihu 命令导入</span>
              </Button>
              <Button variant="outline" size="sm" @click="importFromQl" class="flex-1 sm:flex-initial h-8 gap-1.5 bg-muted/50 hover:bg-muted border-muted-foreground/20 text-muted-foreground px-3">
                <Download class="w-3.5 h-3.5" />
                <span class="text-xs">Qinlong格式导入</span>
              </Button>
              </template>
            </div>
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

                <TaskTagsConfig v-model="form.tags" />
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
                  <Label class="sm:text-right text-xs text-foreground/70 uppercase tracking-wider font-medium">目录名定制</Label>
                  <div class="sm:col-span-3 relative">
                    <Input v-model="repoConfig.repo_dir_name" placeholder="留空则默认为 username_reponame 拼接" class="h-9 bg-muted/30 border-muted-foreground/20 focus:bg-background transition-all" autocomplete="off" />
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

                <div class="grid grid-cols-1 sm:grid-cols-4 items-center gap-3 mt-4">
                  <Label class="sm:text-right text-xs text-foreground/70 uppercase tracking-wider font-bold">前置脚本</Label>
                  <div class="sm:col-span-3 relative"><Input v-model="form.pre_command" placeholder="同步前运行的指令 (可选)" :class="cn('h-9 bg-muted/20 border-muted-foreground/15 transition-all focus:bg-background/50 pr-10', form.pre_command ? 'font-mono text-sm tracking-tight font-medium' : 'text-[11px] font-normal')" /><Zap class="absolute right-3 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-muted-foreground opacity-40 pointer-events-none" /></div>
                </div>
                <div class="grid grid-cols-1 sm:grid-cols-4 items-center gap-3">
                  <Label class="sm:text-right text-xs text-foreground/70 uppercase tracking-wider font-bold">后置脚本</Label>
                  <div class="sm:col-span-3 relative"><Input v-model="form.post_command" placeholder="同步后运行的指令 (可选)" :class="cn('h-9 bg-muted/20 border-muted-foreground/15 transition-all focus:bg-background/50 pr-10', form.post_command ? 'font-mono text-sm tracking-tight font-medium' : 'text-[11px] font-normal')" /><Zap class="absolute right-3 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-muted-foreground opacity-40 pointer-events-none" /></div>
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

                    <TaskLangConfig v-model="selectedLangs" />
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
                <TaskCronConfig v-model="form.schedule" />

                <TaskAdvancedConfig v-model="form" :show-retry="false">
                  <template #run-strategy-prepend>
                    <div class="p-3 rounded-xl bg-muted/20 border border-muted-foreground/10 space-y-2.5">
                      <div class="flex items-center justify-between">
                        <div class="flex items-center gap-2 text-xs font-semibold">
                          <Zap :class="cn('h-3.5 w-3.5', autoAddCron ? 'text-primary' : 'text-muted-foreground')" /> 
                          自动添加任务并解析元数据
                        </div>
                        <Switch :model-value="autoAddCron" @update:model-value="(v: boolean) => { autoAddCron = v; pullQlConfig = v }" />
                      </div>
                      <p class="text-[11px] text-muted-foreground leading-relaxed italic">
                        {{ autoAddCron ? '同步后将自动识别脚本中的 new Env("xxx") 和 cron 信息并注册任务。' : '仅拉取脚本，不自动注册任务。' }}
                      </p>
                    </div>
                  </template>
                </TaskAdvancedConfig>
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

  <!-- Baihu 导入提示对话框 -->
  <Dialog :open="showBaihuImportDialog" @update:open="v => showBaihuImportDialog = v">
    <DialogContent class="sm:max-w-[550px] p-0 border-none bg-background shadow-xl overflow-hidden">
      <DialogHeader class="px-6 pt-6 pb-2">
        <DialogTitle class="text-lg font-bold flex items-center gap-2">
          <Terminal class="w-4 h-4 text-primary" />
          命令行快速导入
        </DialogTitle>
      </DialogHeader>
      
      <div class="px-6 py-4 space-y-5">
        <div class="p-3 rounded-lg bg-primary/5 border border-primary/10">
          <p class="text-xs text-primary/80 leading-relaxed">
            粘贴包含 <code class="px-1 py-0.5 rounded bg-primary/10 font-mono">reposync</code> 及其参数的命令，系统将自动填充表单。
          </p>
        </div>

        <div class="space-y-2">
          <div class="flex items-center justify-between">
            <Label class="text-[11px] font-medium text-muted-foreground uppercase tracking-wider">示例命令</Label>
            <button class="text-[10px] text-primary hover:underline font-medium" @click="baihuCommandInput = 'baihu reposync --source-url \'https://github.com/example/repo.git\' --branch \'main\' --blacklist \'test|dev\' --pre-command \'npm install\' --post-command \'echo done\''">填入示例</button>
          </div>
          <div class="p-3 rounded-lg bg-muted/40 font-mono text-[11px] text-muted-foreground/70 border border-muted/20 leading-relaxed break-all">
            baihu reposync --source-url 'https://...' --branch 'main' --blacklist '...' --pre-command '...' --post-command '...'
          </div>
        </div>

        <div class="relative group">
          <textarea 
            v-model="baihuCommandInput" 
            placeholder="在此处粘贴完整指令，如 baihu reposync --source-url ..." 
            class="w-full min-h-[140px] p-4 rounded-lg bg-muted/30 border border-muted/30 focus:border-primary/40 focus:ring-1 focus:ring-primary/20 transition-all text-sm resize-none outline-none"
            @keydown.enter.ctrl.prevent="submitBaihuImport"
          />
          <div class="absolute bottom-3 right-3 text-[10px] text-muted-foreground/40 font-medium">
            CTRL + ENTER 快速确认
          </div>
        </div>
      </div>
      
      <DialogFooter class="px-6 pb-6 pt-2 flex gap-2">
        <Button variant="ghost" size="sm" @click="showBaihuImportDialog = false" class="flex-1 h-9 rounded-md font-medium text-xs">
          取消
        </Button>
        <Button size="sm" @click="submitBaihuImport" class="flex-1 h-9 rounded-md font-bold text-xs shadow-sm bg-primary hover:bg-primary/90">
          确认解析并填充
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
