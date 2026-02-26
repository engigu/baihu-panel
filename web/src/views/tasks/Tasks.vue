<script setup lang="ts">
import { ref, onMounted, computed, watch } from 'vue'
import { Button } from '@/components/ui/button'
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle } from '@/components/ui/alert-dialog'
import { Input } from '@/components/ui/input'
import Pagination from '@/components/Pagination.vue'
import TaskDialog from './TaskDialog.vue'
import RepoDialog from './RepoDialog.vue'
import { Plus, Play, Pencil, Trash2, Search, ScrollText, GitBranch, Terminal, Server, Monitor, X, Loader2, Wifi, WifiOff, Zap, ZapOff, Copy, Tag } from 'lucide-vue-next'
import { api, type Agent, type Task } from '@/api'
import { toast } from 'vue-sonner'
import { useSiteSettings } from '@/composables/useSiteSettings'
import { useRouter, useRoute } from 'vue-router'
import { TASK_TYPE, AGENT_STATUS, TRIGGER_TYPE } from '@/constants'
import TextOverflow from '@/components/TextOverflow.vue'

const router = useRouter()
const route = useRoute()
const { pageSize } = useSiteSettings()

const tasks = ref<Task[]>([])
const agents = ref<Agent[]>([])
const showTaskDialog = ref(false)
const showRepoDialog = ref(false)
const editingTask = ref<Partial<Task>>({})
const isEdit = ref(false)
const showDeleteDialog = ref(false)
const deleteTaskId = ref<number | null>(null)

const filterName = ref('')
const filterTags = ref('')
const filterAgentId = ref<number | null>(null)
const currentPage = ref(1)
const total = ref(0)
let searchTimer: ReturnType<typeof setTimeout> | null = null

// 创建 agent 映射表
const agentMap = computed(() => {
  const map: Record<number, Agent> = {}
  agents.value.forEach(a => { map[a.id] = a })
  return map
})

// 当前筛选的 Agent 名称
const filterAgentName = computed(() => {
  if (!filterAgentId.value) return ''
  const agent = agentMap.value[filterAgentId.value]
  return agent ? agent.name : `Agent #${filterAgentId.value}`
})

// 获取任务执行位置名称
function getExecutorName(task: Task): string {
  if (!task.agent_id) return '本地'
  const agent = agentMap.value[task.agent_id]
  return agent ? agent.name : `Agent #${task.agent_id}`
}

// 获取任务执行位置状态
function getExecutorStatus(task: Task): 'local' | 'online' | 'offline' {
  if (!task.agent_id) return 'local'
  const agent = agentMap.value[task.agent_id]
  return agent?.status === AGENT_STATUS.ONLINE ? 'online' : 'offline'
}

async function loadTasks() {
  try {
    const res = await api.tasks.list({
      page: currentPage.value,
      page_size: pageSize.value,
      name: filterName.value || undefined,
      tags: filterTags.value || undefined,
      agent_id: filterAgentId.value || undefined
    })
    tasks.value = res.data
    total.value = res.total
  } catch { toast.error('加载任务失败') }
}

async function loadAgents() {
  try {
    agents.value = await api.agents.list()
  } catch { /* ignore */ }
}

function handleSearch() {
  if (searchTimer) clearTimeout(searchTimer)
  searchTimer = setTimeout(() => {
    currentPage.value = 1
    loadTasks()
  }, 300)
}

function handlePageChange(page: number) {
  currentPage.value = page
  loadTasks()
}

function clearAgentFilter() {
  filterAgentId.value = null
  router.replace({ query: {} })
  currentPage.value = 1
  loadTasks()
}

function openCreate() {
  editingTask.value = { name: '', command: '', type: TASK_TYPE.NORMAL, schedule: '0 * * * * *', timeout: 30, work_dir: '', enabled: true, clean_config: '', envs: '' }
  isEdit.value = false
  showTaskDialog.value = true
}

function openCreateRepo() {
  editingTask.value = { name: '', type: TASK_TYPE.REPO, schedule: '0 0 0 * * *', timeout: 30, enabled: true, clean_config: '', envs: '' }
  isEdit.value = false
  showRepoDialog.value = true
}

function openEdit(task: Task) {
  editingTask.value = { ...task }
  isEdit.value = true
  if (task.type === TASK_TYPE.REPO) {
    showRepoDialog.value = true
  } else {
    showTaskDialog.value = true
  }
}

function duplicateTask(task: Task) {
  const newTask = { ...task }
  delete (newTask as any).id
  delete (newTask as any).last_run
  delete (newTask as any).next_run
  newTask.name = newTask.name + ' - 副本'
  editingTask.value = newTask
  isEdit.value = false
  if (task.type === TASK_TYPE.REPO) {
    showRepoDialog.value = true
  } else {
    showTaskDialog.value = true
  }
}

function confirmDelete(id: number) {
  deleteTaskId.value = id
  showDeleteDialog.value = true
}

async function deleteTask() {
  if (!deleteTaskId.value) return
  try {
    await api.tasks.delete(deleteTaskId.value)
    toast.success('任务已删除')
    loadTasks()
  } catch { toast.error('删除失败') }
  showDeleteDialog.value = false
  deleteTaskId.value = null
}

const executingTaskId = ref<number | null>(null)

async function runTask(id: number) {
  executingTaskId.value = id
  toast.message('正在执行...', { id: 'executing' })
  try {
    const res = await api.tasks.execute(id)
    if (res.Success === false) {
      throw new Error(res.Error || '执行失败')
    }
    toast.success('触发成功', { id: 'executing' })
  } catch (error: any) {
    toast.error(error?.message || '执行失败', { id: 'executing' })
  } finally {
    executingTaskId.value = null
  }
}

async function toggleTask(task: Task, enabled: boolean) {
  try {
    await api.tasks.update(task.id, { ...task, enabled })
    toast.success(enabled ? '任务已启用' : '任务已禁用')
    loadTasks()
  } catch { toast.error('操作失败') }
}

function viewLogs(taskId: number) {
  router.push({ path: '/history', query: { task_id: String(taskId) } })
}

function getTaskTypeTitle(type: string) {
  return type === TASK_TYPE.REPO ? '仓库同步' : '普通任务'
}

onMounted(async () => {
  // 先加载 agents，再处理 URL 参数
  await loadAgents()

  // 从 URL 参数读取 agent_id
  const agentIdParam = route.query.agent_id
  if (agentIdParam) {
    filterAgentId.value = Number(agentIdParam)
  }

  loadTasks()
})

// 监听路由参数变化
watch(() => route.query.agent_id, (newVal) => {
  filterAgentId.value = newVal ? Number(newVal) : null
  currentPage.value = 1
  loadTasks()
})
</script>

<template>
  <div class="space-y-6">
    <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
      <div>
        <h2 class="text-xl sm:text-2xl font-bold tracking-tight">定时任务</h2>
        <p class="text-muted-foreground text-sm">管理和调度自动化任务</p>
      </div>
      <div class="flex items-center gap-2">
        <div class="relative flex-1 sm:flex-none">
          <Search class="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
          <Input v-model="filterName" placeholder="搜索任务..." class="h-9 pl-9 w-full sm:w-40 text-sm"
            @input="handleSearch" />
        </div>
        <div class="relative flex-1 sm:flex-none">
          <Tag class="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
          <Input v-model="filterTags" placeholder="搜索标签..." class="h-9 pl-9 w-full sm:w-32 text-sm"
            @input="handleSearch" />
        </div>
        <div v-if="filterAgentId"
          class="flex items-center gap-1 px-2 py-1 bg-primary/10 text-primary rounded-md text-sm">
          <Server class="h-3.5 w-3.5" />
          <span>{{ filterAgentName }}</span>
          <X class="h-3.5 w-3.5 cursor-pointer hover:text-destructive" @click="clearAgentFilter" />
        </div>
        <Button variant="outline" @click="openCreateRepo" class="shrink-0">
          <GitBranch class="h-4 w-4 sm:mr-2" /> <span class="hidden sm:inline">仓库同步</span>
        </Button>
        <Button @click="openCreate" class="shrink-0">
          <Plus class="h-4 w-4 sm:mr-2" /> <span class="hidden sm:inline">新建任务</span>
        </Button>
      </div>
    </div>

    <div class="rounded-lg border bg-card overflow-x-auto">
      <!-- 表头 -->
      <div
        class="flex flex-wrap sm:flex-nowrap items-center gap-x-2 gap-y-1.5 sm:gap-4 px-3 sm:px-4 py-1.5 border-b bg-muted/20 text-xs sm:text-sm text-muted-foreground font-medium min-w-0 sm:min-w-[1000px]">
        <span class="w-10 sm:w-12 shrink-0 max-sm:order-1">ID</span>
        <span class="w-8 shrink-0 text-center max-sm:order-2">类型</span>
        <span class="flex-1 min-w-0 sm:flex-none sm:w-40 md:w-48 lg:w-56 shrink-0 max-sm:order-3">名称</span>
        <span class="w-24 sm:w-32 shrink-0 hidden md:block">执行位置</span>
        <span class="flex-1 min-w-[120px] max-sm:order-6 block sm:block max-sm:mt-1">命令/地址</span>
        <span class="w-28 shrink-0 hidden md:block">定时规则</span>
        <span class="w-40 shrink-0 hidden lg:block">执行时间</span>
        <span class="w-8 shrink-0 text-center max-sm:order-4 max-sm:ml-auto">状态</span>
        <span class="w-28 sm:w-32 shrink-0 text-right sm:text-center max-sm:order-7 max-sm:mt-1">操作</span>
        <div class="w-full hidden max-sm:block max-sm:order-5"></div>
      </div>
      <!-- 列表 -->
      <div class="divide-y min-w-0 sm:min-w-[1000px]">
        <div v-if="tasks.length === 0" class="text-sm text-muted-foreground text-center py-8">
          暂无任务
        </div>
        <div v-for="task in tasks" :key="task.id"
          class="flex flex-wrap sm:flex-nowrap items-center gap-x-2 gap-y-1.5 sm:gap-4 px-3 sm:px-4 py-1.5 hover:bg-muted/30 transition-colors">
          <span class="w-10 sm:w-12 shrink-0 text-muted-foreground text-xs sm:text-sm max-sm:order-1">#{{ task.id
          }}</span>
          <span class="w-8 shrink-0 flex justify-center max-sm:order-2" :title="getTaskTypeTitle(task.type || 'task')">
            <GitBranch v-if="task.type === TASK_TYPE.REPO" class="h-3.5 w-3.5 sm:h-4 sm:w-4 text-primary" />
            <Terminal v-else class="h-3.5 w-3.5 sm:h-4 sm:w-4 text-primary" />
          </span>
          <div
            class="flex-1 min-w-0 sm:flex-none sm:w-40 md:w-48 lg:w-56 shrink-0 flex flex-col justify-center gap-0.5 overflow-hidden max-sm:order-3">
            <span class="font-medium truncate text-xs sm:text-sm cursor-help block w-full" :title="task.name">{{
              task.name }}</span>
            <div v-if="task.tags" class="flex items-center gap-1 overflow-hidden" :title="task.tags">
              <span v-for="tag in task.tags.split(',').filter(Boolean).slice(0, 3)" :key="tag"
                class="truncate text-[10px] leading-none px-1 py-0.5 bg-secondary text-secondary-foreground rounded border">
                {{ tag }}
              </span>
              <span v-if="task.tags.split(',').filter(Boolean).length > 3"
                class="text-[10px] text-muted-foreground">...</span>
            </div>
          </div>
          <span class="w-24 sm:w-32 shrink-0 hidden md:flex items-center gap-1 text-xs" :title="getExecutorName(task)">
            <Monitor v-if="!task.agent_id" class="h-3 w-3 text-muted-foreground" />
            <template v-else>
              <Wifi v-if="getExecutorStatus(task) === 'online'" class="h-3 w-3 text-green-500" />
              <WifiOff v-else class="h-3 w-3 text-muted-foreground" />
            </template>
            <span class="truncate">{{ getExecutorName(task) }}</span>
          </span>

          <code
            class="flex-1 min-w-[120px] text-muted-foreground truncate text-xs bg-muted/40 px-2 py-1 rounded block sm:block max-sm:order-6 overflow-hidden max-sm:mt-1">
  <TextOverflow :text="task.command" :title="task.type === TASK_TYPE.REPO ? '同步地址' : '执行命令'" class="truncate" />
</code>
          <div class="w-28 shrink-0 hidden md:flex flex-col items-start justify-center gap-1 overflow-hidden">
            <span v-if="task.trigger_type === TRIGGER_TYPE.BAIHU_STARTUP"
              class="text-[10px] leading-tight bg-blue-500/10 text-blue-500 px-2 py-1 rounded-md whitespace-nowrap font-medium">服务启动时</span>
            <code v-else-if="task.schedule"
              class="text-muted-foreground text-xs bg-muted/40 px-1.5 py-0.5 rounded truncate max-w-full"
              :title="task.schedule">{{ task.schedule }}</code>
          </div>
          <div class="w-40 shrink-0 hidden lg:flex flex-col justify-center gap-0.5">
            <span class="text-[11px] text-muted-foreground truncate" :title="'上次执行: ' + (task.last_run || '-')">
              上: {{ task.last_run || '-' }}
            </span>
            <span class="text-[11px] text-muted-foreground truncate" :title="'下次执行: ' + (task.next_run || '-')">
              下: {{ task.next_run || '-' }}
            </span>
          </div>
          <span class="w-8 flex justify-center shrink-0 cursor-pointer group max-sm:order-4 max-sm:ml-auto"
            @click="toggleTask(task, !task.enabled)" :title="task.enabled ? '点击禁用' : '点击启用'">
            <div v-if="task.enabled"
              class="h-6 w-6 rounded-md bg-green-500/10 flex items-center justify-center group-hover:bg-green-500/20 transition-colors">
              <Zap class="h-3.5 w-3.5 text-green-500 fill-green-500" />
            </div>
            <div v-else
              class="h-6 w-6 rounded-md bg-muted flex items-center justify-center group-hover:bg-muted/80 transition-colors">
              <ZapOff class="h-3.5 w-3.5 text-muted-foreground" />
            </div>
          </span>
          <span
            class="w-28 sm:w-32 shrink-0 flex justify-end sm:justify-center gap-0.5 sm:gap-1 max-sm:order-7 max-sm:mt-1">
            <Button variant="ghost" size="icon" class="h-6 w-6 sm:h-7 sm:w-7" @click="runTask(task.id)" title="执行"
              :disabled="executingTaskId === task.id">
              <Loader2 v-if="executingTaskId === task.id" class="h-3 w-3 sm:h-3.5 sm:w-3.5 animate-spin" />
              <Play v-else class="h-3 w-3 sm:h-3.5 sm:w-3.5" />
            </Button>
            <Button variant="ghost" size="icon" class="h-6 w-6 sm:h-7 sm:w-7" @click="viewLogs(task.id)" title="日志">
              <ScrollText class="h-3 w-3 sm:h-3.5 sm:w-3.5" />
            </Button>
            <Button variant="ghost" size="icon" class="h-6 w-6 sm:h-7 sm:w-7" @click="openEdit(task)" title="编辑">
              <Pencil class="h-3 w-3 sm:h-3.5 sm:w-3.5" />
            </Button>
            <Button variant="ghost" size="icon" class="h-6 w-6 sm:h-7 sm:w-7" @click="duplicateTask(task)" title="克隆">
              <Copy class="h-3 w-3 sm:h-3.5 sm:w-3.5" />
            </Button>
            <Button variant="ghost" size="icon" class="h-6 w-6 sm:h-7 sm:w-7 text-destructive"
              @click="confirmDelete(task.id)" title="删除">
              <Trash2 class="h-3 w-3 sm:h-3.5 sm:w-3.5" />
            </Button>
          </span>
          <div class="w-full hidden max-sm:block max-sm:order-5"></div>
        </div>
      </div>
      <!-- 分页 -->
      <Pagination :total="total" :page="currentPage" @update:page="handlePageChange" />
    </div>

    <!-- 普通任务弹窗 -->
    <TaskDialog v-model:open="showTaskDialog" :task="editingTask" :is-edit="isEdit" @saved="loadTasks" />

    <!-- 仓库同步弹窗 -->
    <RepoDialog v-model:open="showRepoDialog" :task="editingTask" :is-edit="isEdit" @saved="loadTasks" />

    <!-- 删除确认 -->
    <AlertDialog v-model:open="showDeleteDialog">
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>确认删除</AlertDialogTitle>
          <AlertDialogDescription>确定要删除此任务吗？此操作无法撤销。</AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>取消</AlertDialogCancel>
          <AlertDialogAction class="bg-destructive text-white hover:bg-destructive/90" @click="deleteTask">删除
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  </div>
</template>
