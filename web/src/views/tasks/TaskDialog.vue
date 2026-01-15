<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'
import { Badge } from '@/components/ui/badge'
import DirTreeSelect from '@/components/DirTreeSelect.vue'
import { Plus, ChevronDown, X } from 'lucide-vue-next'
import { api, type Task, type EnvVar, type Agent } from '@/api'
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
  const idx = selectedEnvIds.value.indexOf(id)
  if (idx !== -1) selectedEnvIds.value.splice(idx, 1)
}

async function save() {
  try {
    form.value.clean_config = cleanConfig.value
    form.value.envs = selectedEnvIds.value.join(',')
    form.value.type = 'task'
    form.value.agent_id = selectedAgentId.value === 'local' ? null : Number(selectedAgentId.value)
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
  } catch { toast.error('保存失败') }
}
</script>

<template>
  <Dialog :open="open" @update:open="emit('update:open', $event)">
    <DialogContent class="sm:max-w-[480px]" @open-auto-focus.prevent>
      <DialogHeader>
        <DialogTitle>{{ isEdit ? '编辑任务' : '新建任务' }}</DialogTitle>
      </DialogHeader>
      <div class="space-y-3 py-3">
        <div class="grid grid-cols-1 sm:grid-cols-4 items-center gap-2 sm:gap-3">
          <Label class="sm:text-right text-sm">任务名称</Label>
          <Input v-model="form.name" placeholder="我的任务" class="sm:col-span-3 h-8 text-sm" />
        </div>
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
        <div class="grid grid-cols-1 sm:grid-cols-4 items-center gap-2 sm:gap-3">
          <Label class="sm:text-right text-sm">定时规则</Label>
          <Input v-model="form.schedule" placeholder="0 * * * * *" class="sm:col-span-3 h-8 text-sm font-mono" />
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
                  <div
                    v-for="env in filteredEnvVars"
                    :key="env.id"
                    class="flex items-center gap-2 px-2 py-1 rounded hover:bg-muted cursor-pointer text-xs"
                    @click="addEnv(env.id)"
                  >
                    <Plus class="h-3 w-3 text-muted-foreground" />
                    <span class="truncate">{{ env.name }}</span>
                  </div>
                </div>
              </PopoverContent>
            </Popover>
            <div v-if="selectedEnvs.length > 0" class="flex flex-wrap gap-1">
              <Badge v-for="env in selectedEnvs" :key="env.id" variant="secondary" class="gap-0.5 pr-0.5 text-xs h-5">
                {{ env.name }}
                <X class="h-2.5 w-2.5 cursor-pointer hover:text-destructive" @click.stop="removeEnv(env.id)" />
              </Badge>
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
