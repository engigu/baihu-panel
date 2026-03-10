<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import { Checkbox } from '@/components/ui/checkbox'
import { ScrollArea } from '@/components/ui/scroll-area'
import DirTreeSelect from '@/components/DirTreeSelect.vue'
import { X, Globe, GitBranch, Shield, Zap, Clock } from 'lucide-vue-next'
import { api, type Task, type RepoConfig, type Agent } from '@/api'
import { toast } from 'vue-sonner'
import { cn } from '@/lib/utils'

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
  concurrency: 1,
  proxy: ''
})
const cleanType = ref('none')
const cleanKeep = ref(30)
const allAgents = ref<Agent[]>([])
const selectedAgentId = ref<string>('local')
const tagInput = ref('')

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
  form.value.tags = currentTags.filter(t => t !== tagToRemove).join(',')
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

watch(() => props.open, async (val) => {
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
      concurrency: 1
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
    // 仓库任务暂时仅支持本地执行
    selectedAgentId.value = 'local'
    // 加载 Agent 列表
    await loadAgents()
  }
})

async function loadAgents() {
  try {
    allAgents.value = await api.agents.list()
  } catch { /* ignore */ }
}

async function save() {
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

    form.value.config = JSON.stringify(configToSave)
    form.value.command = `[${repoConfig.value.source_type}] ${repoConfig.value.source_url}`
    form.value.agent_id = selectedAgentId.value === 'local' ? null : selectedAgentId.value
    if (props.isEdit && form.value.id) {
      await api.tasks.update(form.value.id, form.value)
      toast.success('同步任务已更新')
    } else {
      await api.tasks.create(form.value)
      toast.success('同步任务已创建')
    }
    emit('update:open', false)
    emit('saved')
  } catch { toast.error('保存失败') }
}
</script>

<template>
  <Dialog :open="open" @update:open="emit('update:open', $event)">
    <DialogContent class="sm:max-w-[560px] p-0 overflow-hidden border-none bg-background/95 backdrop-blur-xl shadow-2xl" @openAutoFocus.prevent>
      <div class="absolute inset-0 bg-gradient-to-br from-primary/5 via-transparent to-primary/5 pointer-events-none" />

      <div class="flex flex-col max-h-[85vh]">
        <DialogHeader class="px-6 pt-6 pb-2 shrink-0">
          <DialogTitle class="text-xl font-bold bg-clip-text text-transparent bg-gradient-to-r from-foreground to-foreground/70">
            {{ isEdit ? '编辑仓库同步' : '新建仓库同步' }}
          </DialogTitle>
        </DialogHeader>

        <ScrollArea class="flex-1 min-h-0 px-6">
          <div class="space-y-8 py-4 pb-8">
            <!-- 基本信息 Section -->
            <section class="space-y-4">
              <div class="flex items-center gap-2 mb-1">
                <div class="h-4 w-1 bg-primary rounded-full" />
                <h3 class="text-sm font-semibold text-foreground/80">基本信息</h3>
              </div>

              <div class="grid gap-4 pl-3 border-l border-muted">
                <div class="grid grid-cols-1 sm:grid-cols-4 items-center gap-3">
                  <Label class="sm:text-right text-xs text-muted-foreground uppercase tracking-wider">任务名称</Label>
                  <Input v-model="form.name" placeholder="输入同步任务名称" class="sm:col-span-3 h-9 bg-muted/30 border-muted-foreground/20 focus:bg-background transition-all" />
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
                <h3 class="text-sm font-semibold text-foreground/80">核心配置</h3>
              </div>

              <div class="grid gap-4 pl-3 border-l border-muted">
                <div class="grid grid-cols-1 sm:grid-cols-4 items-center gap-3">
                  <Label class="sm:text-right text-xs text-muted-foreground uppercase tracking-wider">源类型</Label>
                  <div class="sm:col-span-3">
                    <Select :model-value="repoConfig.source_type" @update:model-value="(v) => repoConfig.source_type = String(v || 'git')">
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
                  <Label class="sm:text-right text-xs text-muted-foreground uppercase tracking-wider font-semibold">源地址</Label>
                  <div class="sm:col-span-3 relative">
                    <Input v-model="repoConfig.source_url"
                      :placeholder="repoConfig.source_type === 'git' ? 'https://github.com/user/repo.git' : 'https://example.com/file.js'"
                      class="h-9 font-mono text-[13px] bg-muted/30 border-muted-foreground/20 focus:bg-background pr-10 transition-all" 
                      autocomplete="off" />
                    <Globe class="absolute right-3 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-muted-foreground opacity-40" />
                  </div>
                </div>

                <div class="grid grid-cols-1 sm:grid-cols-4 items-center gap-3">
                  <Label class="sm:text-right text-xs text-muted-foreground uppercase tracking-wider">目标路径</Label>
                  <div class="sm:col-span-3">
                    <DirTreeSelect v-if="selectedAgentId === 'local'" :model-value="repoConfig.target_path || ''"
                      @update:model-value="v => repoConfig.target_path = v" class="h-9" />
                    <Input v-else v-model="repoConfig.target_path" placeholder="Agent 上的目标路径" class="h-9 bg-muted/30 border-muted-foreground/20" />
                  </div>
                </div>

                <div v-if="repoConfig.source_type === 'git'" class="grid grid-cols-1 sm:grid-cols-4 items-center gap-3">
                  <Label class="sm:text-right text-xs text-muted-foreground uppercase tracking-wider">分支</Label>
                  <Input v-model="repoConfig.branch" placeholder="main (默认)" class="sm:col-span-3 h-9 bg-muted/30 border-muted-foreground/20 focus:bg-background transition-all" autocomplete="off" />
                </div>

                <div v-if="repoConfig.source_type === 'git'" class="grid grid-cols-1 sm:grid-cols-4 items-center gap-3">
                  <Label class="sm:text-right text-xs text-muted-foreground uppercase tracking-wider">稀疏路径</Label>
                  <Input v-model="repoConfig.sparse_path" placeholder="指定目录或文件 (可选)" class="sm:col-span-3 h-9 bg-muted/30 border-muted-foreground/20 focus:bg-background transition-all" autocomplete="off" />
                </div>

                <div v-if="repoConfig.source_type === 'git' && repoConfig.sparse_path" class="grid grid-cols-1 sm:grid-cols-4 items-center gap-3">
                  <Label class="sm:text-right text-xs text-muted-foreground uppercase tracking-wider">下载模式</Label>
                  <div class="sm:col-span-3">
                    <div class="flex items-center space-x-2 bg-muted/20 px-3 py-1.5 rounded-full border border-muted-foreground/10 w-fit">
                      <Checkbox id="single-file-sync" v-model:checked="isSingleFile" class="scale-90" />
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
                <h3 class="text-sm font-semibold text-foreground/80">访问控制</h3>
              </div>

              <div class="grid gap-4 pl-3 border-l border-muted">
                <div class="grid grid-cols-1 sm:grid-cols-4 items-center gap-3">
                  <Label class="sm:text-right text-xs text-muted-foreground uppercase tracking-wider">代理配置</Label>
                  <div class="sm:col-span-3">
                    <Select :model-value="repoConfig.proxy" @update:model-value="(v) => repoConfig.proxy = String(v || 'none')">
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
                  <Label class="sm:text-right text-xs text-muted-foreground uppercase tracking-wider">代理地址</Label>
                  <Input v-model="repoConfig.proxy_url" placeholder="https://your-proxy.com" class="sm:col-span-3 h-9 bg-muted/30 font-mono text-xs border-muted-foreground/20 focus:bg-background transition-all" autocomplete="off" />
                </div>

                <div class="grid grid-cols-1 sm:grid-cols-4 items-center gap-3">
                  <Label class="sm:text-right text-xs text-muted-foreground uppercase tracking-wider">身份认证</Label>
                  <div class="sm:col-span-3 relative">
                    <Input v-model="repoConfig.auth_token" type="password" placeholder="推荐使用 Token 替代密码" class="h-9 bg-muted/30 border-muted-foreground/20 pr-10 text-xs focus:bg-background transition-all" autocomplete="new-password" />
                    <Shield class="absolute right-3 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-muted-foreground opacity-40" />
                  </div>
                </div>
              </div>
            </section>

            <!-- 调度策略 Section -->
            <section class="space-y-4">
              <div class="flex items-center gap-2 mb-1">
                <div class="h-4 w-1 bg-primary rounded-full" />
                <h3 class="text-sm font-semibold text-foreground/80">调度策略</h3>
              </div>

              <div class="grid gap-5 pl-3 border-l border-muted">
                <div class="grid grid-cols-1 sm:grid-cols-4 items-center gap-3">
                  <Label class="sm:text-right text-xs text-muted-foreground uppercase tracking-wider font-semibold">定时规则</Label>
                  <div class="sm:col-span-3">
                    <Input v-model="form.schedule" placeholder="* * * * * *" class="h-9 font-mono text-[13px] bg-muted/30 border-muted-foreground/20 focus:ring-1 focus:ring-primary/50" />
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
                  <Label class="sm:text-right text-xs text-muted-foreground uppercase tracking-wider">随机延迟</Label>
                  <div class="sm:col-span-3 flex items-center gap-4">
                    <div class="flex items-center gap-2">
                      <Input :model-value="form.random_range" @update:model-value="v => form.random_range = Number(v || 0)" type="number" :min="0" class="w-20 h-9 bg-muted/30 text-center" />
                      <span class="text-xs font-semibold text-muted-foreground">秒</span>
                    </div>
                    <div class="flex-1 text-[11px] text-muted-foreground leading-snug p-2 rounded-lg bg-blue-500/5 border border-blue-500/10 italic">
                      避免高频并发，在基准时间点后延迟 0~{{ form.random_range || 0 }}s
                    </div>
                  </div>
                </div>

                <div class="grid grid-cols-1 sm:grid-cols-4 items-start gap-3">
                  <Label class="sm:text-right text-xs text-muted-foreground uppercase tracking-wider">运行策略</Label>
                  <div class="sm:col-span-3 space-y-4">
                    <div class="flex items-center gap-4">
                      <div class="flex items-center gap-2">
                         <Input :model-value="form.timeout" @update:model-value="v => form.timeout = Number(v || 0)" type="number" :min="0" class="w-20 h-9 bg-muted/30 text-center" />
                         <span class="text-[11px] font-semibold text-muted-foreground">分钟超时</span>
                      </div>
                      <div class="flex items-center gap-2 pl-4 border-l">
                        <Select :model-value="cleanType" @update:model-value="(v) => cleanType = String(v || 'none')">
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
</template>
