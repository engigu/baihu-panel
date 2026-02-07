<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import { Checkbox } from '@/components/ui/checkbox'
import DirTreeSelect from '@/components/DirTreeSelect.vue'
import { api, type Task, type RepoConfig, type Agent } from '@/api'
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

const concurrencyEnabled = computed({
  get: () => repoConfig.value.concurrency === 1,
  set: (val: boolean) => {
    repoConfig.value.concurrency = val ? 1 : 0
  }
})

const cleanConfig = computed(() => {
  if (!cleanType.value || cleanType.value === 'none' || cleanKeep.value <= 0) return ''
  return JSON.stringify({ type: cleanType.value, keep: cleanKeep.value })
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
      '$task_concurrency': repoConfig.value.concurrency !== undefined ? repoConfig.value.concurrency : 1
    }
    
    form.value.config = JSON.stringify(configToSave)
    form.value.command = `[${repoConfig.value.source_type}] ${repoConfig.value.source_url}`
    form.value.agent_id = selectedAgentId.value === 'local' ? null : Number(selectedAgentId.value)
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
    <DialogContent class="sm:max-w-[480px] max-h-[85vh] flex flex-col !gap-0 !p-0" :trap-focus="false" @open-auto-focus.prevent>
      <DialogHeader class="shrink-0 p-6 pb-0">
        <DialogTitle>{{ isEdit ? '编辑仓库同步' : '新建仓库同步' }}</DialogTitle>
      </DialogHeader>
      <div class="space-y-3 py-3 px-6 overflow-y-auto flex-1 custom-scrollbar">
        <div class="grid grid-cols-1 sm:grid-cols-4 items-center gap-2 sm:gap-3">
          <Label class="sm:text-right text-sm">任务名称</Label>
          <Input v-model="form.name" placeholder="我的仓库同步" class="sm:col-span-3 h-8 text-sm" />
        </div>
        <div class="grid grid-cols-1 sm:grid-cols-4 items-center gap-2 sm:gap-3">
          <Label class="sm:text-right text-sm">源类型</Label>
          <Select :model-value="repoConfig.source_type" @update:model-value="(v) => repoConfig.source_type = String(v || 'git')">
            <SelectTrigger class="sm:col-span-3 h-8 text-sm">
              <SelectValue placeholder="选择源类型" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="git">Git 仓库</SelectItem>
              <SelectItem value="url">URL 下载</SelectItem>
            </SelectContent>
          </Select>
        </div>
        <div class="grid grid-cols-1 sm:grid-cols-4 items-center gap-2 sm:gap-3">
          <Label class="sm:text-right text-sm">源地址</Label>
          <Input v-model="repoConfig.source_url" :placeholder="repoConfig.source_type === 'git' ? 'https://github.com/user/repo.git' : 'https://example.com/file.js'" class="sm:col-span-3 h-8 text-xs font-mono" />
        </div>
        <div class="grid grid-cols-1 sm:grid-cols-4 items-center gap-2 sm:gap-3">
          <Label class="sm:text-right text-sm">目标路径</Label>
          <div class="sm:col-span-3">
            <DirTreeSelect v-if="selectedAgentId === 'local'" :model-value="repoConfig.target_path || ''" @update:model-value="v => repoConfig.target_path = v" />
            <Input v-else v-model="repoConfig.target_path" placeholder="Agent 上的目标路径" class="h-8 text-sm" />
          </div>
        </div>
        <div class="grid grid-cols-1 sm:grid-cols-4 items-center gap-2 sm:gap-3">
          <Label class="sm:text-right text-sm">执行位置</Label>
          <div class="sm:col-span-3">
            <Select v-model="selectedAgentId" disabled>
              <SelectTrigger class="h-8 text-sm">
                <SelectValue placeholder="本地执行" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="local">本地执行</SelectItem>
              </SelectContent>
            </Select>
            <p class="text-xs text-muted-foreground mt-1">仓库同步任务暂时仅支持本地执行</p>
          </div>
        </div>
        <div v-if="repoConfig.source_type === 'git'" class="grid grid-cols-1 sm:grid-cols-4 items-center gap-2 sm:gap-3">
          <Label class="sm:text-right text-sm">分支</Label>
          <Input v-model="repoConfig.branch" placeholder="main (可选)" class="sm:col-span-3 h-8 text-sm" autocomplete="off" />
        </div>
        <div v-if="repoConfig.source_type === 'git'" class="grid grid-cols-1 sm:grid-cols-4 items-center gap-2 sm:gap-3">
          <Label class="sm:text-right text-sm">稀疏路径</Label>
          <Input v-model="repoConfig.sparse_path" placeholder="仅拉取指定目录或文件 (可选)" class="sm:col-span-3 h-8 text-sm" autocomplete="off" />
        </div>
        <div v-if="repoConfig.source_type === 'git' && repoConfig.sparse_path" class="grid grid-cols-1 sm:grid-cols-4 items-center gap-2 sm:gap-3">
          <Label class="sm:text-right text-sm">单文件</Label>
          <div class="sm:col-span-3 flex items-center gap-2">
            <Checkbox :checked="repoConfig.single_file" @update:checked="(v: boolean) => repoConfig.single_file = v" />
            <span class="text-xs text-muted-foreground">直接下载文件（适用于单个文件同步）</span>
          </div>
        </div>
        <div class="grid grid-cols-1 sm:grid-cols-4 items-center gap-2 sm:gap-3">
          <Label class="sm:text-right text-sm">代理</Label>
          <Select :model-value="repoConfig.proxy" @update:model-value="(v) => repoConfig.proxy = String(v || 'none')">
            <SelectTrigger class="sm:col-span-3 h-8 text-sm">
              <SelectValue placeholder="选择代理" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem v-for="opt in proxyOptions" :key="opt.value" :value="opt.value">{{ opt.label }}</SelectItem>
            </SelectContent>
          </Select>
        </div>
        <div v-if="repoConfig.proxy === 'custom'" class="grid grid-cols-1 sm:grid-cols-4 items-center gap-2 sm:gap-3">
          <Label class="sm:text-right text-sm">代理地址</Label>
          <Input v-model="repoConfig.proxy_url" placeholder="https://your-proxy.com/" class="sm:col-span-3 h-8 text-xs font-mono" autocomplete="off" />
        </div>
        <div class="grid grid-cols-1 sm:grid-cols-4 items-center gap-2 sm:gap-3">
          <Label class="sm:text-right text-sm">认证Token</Label>
          <Input v-model="repoConfig.auth_token" type="text" placeholder="可选，用于私有仓库" class="sm:col-span-3 h-8 text-sm" autocomplete="new-password" />
        </div>
        <div class="grid grid-cols-1 sm:grid-cols-4 items-center gap-2 sm:gap-3">
          <Label class="sm:text-right text-sm">并发控制</Label>
          <div class="sm:col-span-3 flex items-center gap-2">
            <Switch v-model:checked="concurrencyEnabled" />
            <span class="text-sm text-muted-foreground">允许并发</span>
            <span class="text-xs text-muted-foreground ml-2">(如果任务未执行完成，是否允许再次执行)</span>
          </div>
        </div>
        <div class="grid grid-cols-1 sm:grid-cols-4 items-center gap-2 sm:gap-3">
          <Label class="sm:text-right text-sm">定时规则</Label>
          <Input v-model="form.schedule" placeholder="0 0 0 * * *" class="sm:col-span-3 h-8 text-sm font-mono" />
        </div>
        <div class="grid grid-cols-1 sm:grid-cols-4 items-start gap-2 sm:gap-3">
          <span></span>
          <div class="sm:col-span-3">
            <p class="text-xs text-muted-foreground mb-1.5">格式: 秒 分 时 日 月 周</p>
            <div class="flex flex-wrap gap-1">
              <span
                v-for="preset in cronPresets"
                :key="preset.value"
                class="px-1.5 py-0.5 text-xs rounded bg-muted hover:bg-accent cursor-pointer transition-colors"
                @click="form.schedule = preset.value"
              >
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
              <Input v-if="cleanType && cleanType !== 'none'" v-model.number="cleanKeep" type="number" :placeholder="cleanType === 'day' ? '7' : '100'" class="w-20 h-9 text-sm" />
            </div>
          </div>
        </div>
      </div>
      <DialogFooter class="shrink-0 p-6 pt-3 border-t">
        <Button variant="outline" size="sm" @click="emit('update:open', false)">取消</Button>
        <Button size="sm" @click="save">保存</Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>
