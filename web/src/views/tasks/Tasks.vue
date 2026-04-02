<script setup lang="ts">
import { ref, onMounted, computed, watch, onUnmounted } from 'vue'
import { Button } from '@/components/ui/button'
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle } from '@/components/ui/alert-dialog'
import { Input } from '@/components/ui/input'
import Pagination from '@/components/Pagination.vue'
import TaskDialog from './TaskDialog.vue'
import RepoDialog from './RepoDialog.vue'
import LogViewer from '@/views/history/LogViewer.vue'
import { Plus, Play, Pencil, Trash2, Search, ScrollText, GitBranch, Terminal, Server, Monitor, X, Loader2, RefreshCw, Wifi, WifiOff, Zap, ZapOff, Copy, Tag } from 'lucide-vue-next'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { api, type Agent, type Task } from '@/api'
import { toast } from 'vue-sonner'
import { useSiteSettings } from '@/composables/useSiteSettings'
import { useRouter, useRoute } from 'vue-router'
import { TASK_TYPE, AGENT_STATUS, TRIGGER_TYPE, TASK_STATUS } from '@/constants'
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
const deleteTaskId = ref<string | null>(null)

const filterName = ref('')
const filterTags = ref('')
const filterType = ref<string>(TASK_TYPE.NORMAL)
const filterAgentId = ref<string | null>(null)
const currentPage = ref(1)
const total = ref(0)
const loading = ref(false)
let searchTimer: ReturnType<typeof setTimeout> | null = null

// 创建 agent 映射表
const agentMap = computed(() => {
  const map: Record<string, Agent> = {}
  agents.value.forEach((a: Agent) => { map[a.id] = a })
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
  loading.value = true
  try {
    const res = await api.tasks.list({
      page: currentPage.value,
      page_size: pageSize.value,
      name: filterName.value || undefined,
      tags: filterTags.value || undefined,
      type: filterType.value === 'all' ? undefined : filterType.value,
      agent_id: filterAgentId.value || undefined
    })
    tasks.value = res.data
    total.value = res.total
  } catch { toast.error('加载任务失败') } finally {
    loading.value = false
  }
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

function handleTypeChange() {
  currentPage.value = 1
  loadTasks()
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
  editingTask.value = { name: '', remark: '', command: '', type: TASK_TYPE.NORMAL, schedule: '0 * * * * *', timeout: 30, work_dir: '', enabled: true, clean_config: '', envs: '', random_range: 0 }
  isEdit.value = false
  showTaskDialog.value = true
}

function openCreateRepo() {
  editingTask.value = { name: '', remark: '', type: TASK_TYPE.REPO, schedule: '0 0 0 * * *', timeout: 30, enabled: true, clean_config: '', envs: '', random_range: 0 }
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

const showBatchDeleteDialog = ref(false)

function confirmDelete(id: string) {
  deleteTaskId.value = id
  showDeleteDialog.value = true
}

function confirmBatchDelete() {
  if (total.value === 0) return
  showBatchDeleteDialog.value = true
}

async function batchDeleteTasks() {
  try {
    const res = await api.tasks.batchDeleteByQuery({
      name: filterName.value || undefined,
      tags: filterTags.value || undefined,
      type: filterType.value === 'all' ? undefined : filterType.value,
      agent_id: filterAgentId.value || undefined
    })
    toast.success(`成功删除 ${res.count} 个任务`)
    loadTasks()
  } catch {
    toast.error('批量删除失败')
  }
  showBatchDeleteDialog.value = false
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

const executingTaskId = ref<string | null>(null)

async function runTask(id: string) {
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

const showLogViewer = ref(false)
const selectedLogId = ref<string | undefined>()
const latestLogStatus = ref('')
const latestLogTitle = ref('')
const logContent = ref('')
let logSocket: WebSocket | null = null

function cleanupLogSocket() {
  if (logSocket) {
    logSocket.onopen = null
    logSocket.onmessage = null
    logSocket.onerror = null
    logSocket.onclose = null
    logSocket.close()
    logSocket = null
  }
}

watch(showLogViewer, (val) => {
  if (!val) {
    cleanupLogSocket()
    logContent.value = ''
  }
})

onUnmounted(() => {
  cleanupLogSocket()
})

import { decompressFromBase64 } from '@/utils/decompress'

const displayLogContent = computed(() => {
  if (!logContent.value) return '无输出'
  return decompressFromBase64(logContent.value)
})

async function viewLogs(taskId: string) {
  try {
    const res = await api.logs.list({ task_id: taskId, page: 1, page_size: 1 })
    if (res.data && res.data.length > 0) {
      const latestLog = res.data[0]
      if (!latestLog) return
      latestLogTitle.value = latestLog.task_name || ''
      latestLogStatus.value = latestLog.status || ''
      selectedLogId.value = latestLog.id
      logContent.value = ''
      showLogViewer.value = true

      if (latestLog.status !== TASK_STATUS.RUNNING) {
        try {
          const detail = await api.logs.get(latestLog.id)
          logContent.value = detail.output
        } catch {
          toast.error('加载日志详情失败')
        }
        return
      }

      // Connect WebSocket to load log content for running tasks
      cleanupLogSocket()
      const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
      const host = window.location.host
      const baseUrl = (window as any).__BASE_URL__ || ''
      const apiVersion = (window as any).__API_VERSION__ || '/api/v1'
      const wsUrl = `${protocol}//${host}${baseUrl}${apiVersion}/logs/ws?log_id=${latestLog.id}`

      logSocket = new WebSocket(wsUrl)
      logSocket.onmessage = (event) => {
        logContent.value += event.data
      }
    } else {
      toast.info('该任务暂无执行日志')
    }
  } catch {
    toast.error('获取日志失败')
  }
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
    filterAgentId.value = String(agentIdParam)
  }

  loadTasks()
})

// 监听路由参数变化
watch(() => route.query.agent_id, (newVal: any) => {
  filterAgentId.value = newVal ? String(newVal) : null
  currentPage.value = 1
  loadTasks()
})
</script>

<template>
  <div class="space-y-6">
    <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
      <div class="flex flex-col">
        <h2 class="text-xl sm:text-2xl font-bold tracking-tight">定时任务</h2>
        <p class="text-muted-foreground text-sm">管理和调度自动化执行任务</p>
      </div>

      <div class="flex flex-col sm:flex-row gap-2.5 w-full md:w-auto">
        <!-- 搜索与标签 -->
        <div class="flex items-center gap-2 w-full sm:w-auto">
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
        </div>

        <div class="flex items-center gap-3 w-full sm:w-auto">
          <!-- 移动端类型切换 -->
          <div class="md:hidden flex-1 shrink-0">
             <Select v-model="filterType" @update:model-value="(_v: any) => handleTypeChange()">
               <SelectTrigger class="h-9 w-full text-sm">
                 <SelectValue />
               </SelectTrigger>
               <SelectContent>
                 <SelectItem :value="TASK_TYPE.NORMAL">定时任务</SelectItem>
                 <SelectItem :value="TASK_TYPE.REPO">仓库同步</SelectItem>
               </SelectContent>
             </Select>
          </div>

          <div v-if="filterAgentId"
            class="hidden sm:flex items-center gap-1 px-2 py-1 bg-primary/10 text-primary rounded-md text-sm shrink-0">
            <Server class="h-3.5 w-3.5" />
            <span>{{ filterAgentName }}</span>
            <X class="h-3.5 w-3.5 cursor-pointer hover:text-destructive" @click="clearAgentFilter" />
          </div>

          <Button variant="outline" size="icon" class="h-9 w-9 shrink-0" @click="loadTasks" :disabled="loading" title="刷新">
            <RefreshCw class="h-4 w-4" :class="{ 'animate-spin': loading }" />
          </Button>

          <div class="flex items-center gap-2 shrink-0 justify-end">
            <!-- 动态新增按钮 -->
            <Button variant="outline" class="shrink-0 px-3 h-9 shadow-sm text-destructive border-destructive/20 hover:bg-destructive/10" @click="confirmBatchDelete">
              <Trash2 class="h-4 w-4 sm:mr-2" /> <span class="hidden sm:inline">批量删除</span>
            </Button>
            <Button v-if="filterType === TASK_TYPE.NORMAL" @click="openCreate" class="shrink-0 px-3 h-9 shadow-sm">
              <Plus class="h-4 w-4 sm:mr-2" /> <span class="hidden sm:inline">新建任务</span>
            </Button>
            <Button v-else-if="filterType === TASK_TYPE.REPO" @click="openCreateRepo" class="shrink-0 px-3 h-9 shadow-sm">
              <GitBranch class="h-4 w-4 sm:mr-2" /> <span class="hidden sm:inline">同步仓库</span>
            </Button>
          </div>

          <!-- 桌面端类型切换移到后面 -->
          <Tabs :model-value="filterType" @update:model-value="(v: string | number) => { filterType = String(v); handleTypeChange() }" class="shrink-0 hidden md:block">
             <TabsList class="h-9 p-1 bg-muted/30 border">
                <TabsTrigger :value="TASK_TYPE.NORMAL" class="px-4 h-7 text-[13px]">定时任务</TabsTrigger>
                <TabsTrigger :value="TASK_TYPE.REPO" class="px-4 h-7 text-[13px]">仓库同步</TabsTrigger>
             </TabsList>
          </Tabs>
        </div>
        <!-- 移动端 agent 过滤标签 -->
        <div v-if="filterAgentId"
          class="flex sm:hidden items-center gap-1 px-2 py-1 bg-primary/10 text-primary rounded-md text-sm w-fit mt-1">
          <Server class="h-3.5 w-3.5" />
          <span>{{ filterAgentName }}</span>
          <X class="h-3.5 w-3.5 cursor-pointer hover:text-destructive" @click="clearAgentFilter" />
        </div>
      </div>
    </div>

    <div class="rounded-lg border bg-card">
      <!-- 大屏表头 -->
      <div
        class="hidden sm:flex items-center gap-4 px-4 py-1.5 border-b bg-muted/20 text-xs text-muted-foreground font-medium">
        <span class="w-12 shrink-0 pl-1">序号</span>
        <span class="w-8 shrink-0 text-center">类型</span>
        <span class="w-40 md:w-48 lg:w-56 shrink-0">名称</span>
        <span class="w-32 shrink-0 hidden md:block">执行位置</span>
        <span class="w-8 shrink-0 text-center">状态</span>
        <span class="flex-1 min-w-0 flex items-center gap-1.5">
          <Terminal class="h-3.5 w-3.5 opacity-50" />命令/地址
        </span>
        <span class="w-28 shrink-0 hidden md:block">定时规则</span>
        <span class="w-40 shrink-0 hidden lg:block">执行时间</span>
        <span class="w-36 shrink-0 text-center">操作</span>
      </div>

      <div class="divide-y">
        <div v-if="tasks.length === 0" class="text-sm text-muted-foreground text-center py-8">
          暂无任务
        </div>

        <!-- 小屏卡片布局 -->
        <div v-for="(task, index) in tasks" :key="`mobile-${task.id}`"
          class="sm:hidden p-3 hover:bg-muted/50 transition-colors">
          <!-- 标题行 -->
          <div class="flex items-start justify-between mb-3 border-b border-border/40 pb-2">
            <div class="flex items-center gap-2 flex-1 min-w-0 pr-2">
              <span class="text-xs text-muted-foreground tabular-nums flex-shrink-0">#{{ total - (currentPage - 1) * pageSize - index }}</span>
              <span class="shrink-0" :title="getTaskTypeTitle(task.type || 'task')">
                <GitBranch v-if="task.type === TASK_TYPE.REPO" class="h-3.5 w-3.5 text-primary" />
                <Terminal v-else class="h-3.5 w-3.5 text-primary" />
              </span>
              <span class="font-bold text-sm truncate" :title="task.name">{{ task.name }}</span>
            </div>
            <div class="flex items-center gap-2 shrink-0">
              <span class="cursor-pointer group"
                @click="toggleTask(task, !task.enabled)"
                :title="task.enabled ? '点击禁用' : '点击启用'">
                <div v-if="task.enabled"
                  class="h-6 w-6 rounded-md bg-green-500/10 flex items-center justify-center group-hover:bg-green-500/20 transition-colors">
                  <Zap class="h-3.5 w-3.5 text-green-500 fill-green-500" />
                </div>
                <div v-else
                  class="h-6 w-6 rounded-md bg-muted flex items-center justify-center group-hover:bg-muted/80 transition-colors">
                  <ZapOff class="h-3.5 w-3.5 text-muted-foreground" />
                </div>
              </span>
            </div>
          </div>

          <!-- 详情信息列表 -->
          <div class="space-y-1.5 text-xs text-muted-foreground mb-3 px-1">
            <div class="flex items-center gap-3">
              <span class="w-10 shrink-0 font-medium opacity-70">定时:</span>
              <span v-if="task.trigger_type === TRIGGER_TYPE.BAIHU_STARTUP"
                class="text-[10px] leading-tight bg-blue-500/10 text-blue-500 px-1.5 py-0.5 rounded font-medium">服务启动时</span>
              <span v-else-if="task.schedule" class="text-xs text-foreground bg-muted/40 px-1.5 py-0.5 rounded">{{ task.schedule }}</span>
              <span v-else class="text-muted-foreground opacity-50">-</span>
            </div>
            <div class="flex items-start gap-3">
              <span class="w-10 shrink-0 font-medium mt-0.5 opacity-70">命令:</span>
              <div class="flex-1 min-w-0 overflow-hidden text-foreground">
                <TextOverflow :text="task.command" :title="task.type === TASK_TYPE.REPO ? '同步地址' : '执行命令'" class="truncate opacity-80" />
              </div>
            </div>
            <div v-if="task.tags" class="flex items-start gap-3">
              <span class="w-10 shrink-0 font-medium mt-0.5 opacity-70">标签:</span>
              <div class="flex flex-wrap gap-1 flex-1">
                <span v-for="tag in task.tags.split(',').filter(Boolean)" :key="tag"
                  class="text-[9px] leading-none px-1.5 py-0.5 bg-primary/10 text-primary/80 rounded-full border border-primary/20 font-semibold">
                  {{ tag }}
                </span>
              </div>
            </div>
            <div v-if="task.remark" class="flex items-start gap-3">
              <span class="w-10 shrink-0 font-medium mt-0.5 opacity-70">备注:</span>
              <span class="flex-1 text-[11px] truncate">{{ task.remark }}</span>
            </div>
          </div>

          <!-- 操作按钮行 -->
          <div class="flex items-center justify-end gap-0.5 pt-2 border-t border-border/40">
            <Button variant="ghost" class="h-7 px-1.5 text-[10px] gap-1" @click="runTask(task.id)" title="执行" :disabled="executingTaskId === task.id">
              <Loader2 v-if="executingTaskId === task.id" class="h-3 w-3 animate-spin" />
              <Play v-else class="h-3 w-3" />执行
            </Button>
            <Button variant="ghost" class="h-7 px-1.5 text-[10px] gap-1" @click="viewLogs(task.id)" title="日志">
              <ScrollText class="h-3 w-3" />日志
            </Button>
            <Button variant="ghost" class="h-7 px-1.5 text-[10px] gap-1" @click="openEdit(task)" title="编辑">
              <Pencil class="h-3 w-3" />编辑
            </Button>
            <Button variant="ghost" class="h-7 px-1.5 text-[10px] gap-1" @click="duplicateTask(task)" title="克隆">
              <Copy class="h-3 w-3" />克隆
            </Button>
            <Button variant="ghost" class="h-7 px-1.5 text-[10px] gap-1 text-destructive" @click="confirmDelete(task.id)" title="删除">
              <Trash2 class="h-3 w-3" />删除
            </Button>
          </div>
        </div>

        <!-- 大屏行布局 -->
        <div v-for="(task, index) in tasks" :key="`desktop-${task.id}`"
          class="hidden sm:flex items-center gap-4 px-4 py-1.5 hover:bg-muted/30 transition-colors">
          <div class="w-12 shrink-0 pl-1">
            <span class="text-muted-foreground text-sm">#{{ total - (currentPage - 1) * pageSize - index }}</span>
          </div>
          <span class="w-8 shrink-0 flex justify-center" :title="getTaskTypeTitle(task.type || 'task')">
            <GitBranch v-if="task.type === TASK_TYPE.REPO" class="h-4 w-4 text-primary" />
            <Terminal v-else class="h-4 w-4 text-primary" />
          </span>
          <div class="w-40 md:w-48 lg:w-56 shrink-0 flex flex-col justify-center gap-0.5 overflow-hidden">
            <span class="font-medium truncate text-sm cursor-help" :title="task.name">{{ task.name }}</span>
            <div v-if="task.tags" class="flex items-center gap-1 overflow-hidden" :title="task.tags">
              <span v-for="tag in task.tags.split(',').filter(Boolean).slice(0, 3)" :key="tag"
                class="truncate text-[10px] leading-none px-1 py-0.5 bg-secondary text-secondary-foreground rounded border">
                {{ tag }}
              </span>
              <span v-if="task.tags.split(',').filter(Boolean).length > 3" class="text-[10px] text-muted-foreground">...</span>
            </div>
          </div>
          <span class="w-32 shrink-0 hidden md:flex items-center gap-1 text-xs" :title="getExecutorName(task)">
            <Monitor v-if="!task.agent_id" class="h-3 w-3 text-muted-foreground" />
            <template v-else>
              <Wifi v-if="getExecutorStatus(task) === 'online'" class="h-3 w-3 text-green-500" />
              <WifiOff v-else class="h-3 w-3 text-muted-foreground" />
            </template>
            <span class="truncate">{{ getExecutorName(task) }}</span>
          </span>
          <span class="w-8 flex justify-center shrink-0 cursor-pointer group"
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
          <code class="flex-1 min-w-0 text-muted-foreground truncate text-xs bg-muted/40 px-2 py-1 rounded overflow-hidden">
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
          <span class="w-36 shrink-0 flex justify-center gap-1">
            <Button variant="ghost" size="icon" class="h-7 w-7" @click="runTask(task.id)" title="执行"
              :disabled="executingTaskId === task.id">
              <Loader2 v-if="executingTaskId === task.id" class="h-3.5 w-3.5 animate-spin" />
              <Play v-else class="h-3.5 w-3.5" />
            </Button>
            <Button variant="ghost" size="icon" class="h-7 w-7" @click="viewLogs(task.id)" title="日志">
              <ScrollText class="h-3.5 w-3.5" />
            </Button>
            <Button variant="ghost" size="icon" class="h-7 w-7" @click="openEdit(task)" title="编辑">
              <Pencil class="h-3.5 w-3.5" />
            </Button>
            <Button variant="ghost" size="icon" class="h-7 w-7" @click="duplicateTask(task)" title="克隆">
              <Copy class="h-3.5 w-3.5" />
            </Button>
            <Button variant="ghost" size="icon" class="h-7 w-7 text-destructive"
              @click="confirmDelete(task.id)" title="删除">
              <Trash2 class="h-3.5 w-3.5" />
            </Button>
          </span>
        </div>
      </div>
      <!-- 分页 -->
      <Pagination :total="total" :page="currentPage" @update:page="handlePageChange" />
    </div>

    <!-- 普通任务弹窗 -->
    <TaskDialog v-model:open="showTaskDialog" :task="editingTask" :is-edit="isEdit" @saved="loadTasks" />

    <!-- 仓库同步弹窗 -->
    <RepoDialog v-model:open="showRepoDialog" :task="editingTask" :is-edit="isEdit" @saved="loadTasks" />

    <!-- 最新日志全屏查看 -->
    <LogViewer v-model:open="showLogViewer" :title="`最新日志 - ${latestLogTitle}`"
      :content="displayLogContent || '无输出'" :status="latestLogStatus" />

    <!-- 删除确认 (批量) -->
    <AlertDialog v-model:open="showBatchDeleteDialog">
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>确认批量删除</AlertDialogTitle>
          <AlertDialogDescription>
            将会删除当前所有过滤条件下匹配的 <b>{{ total }}</b> 个任务。操作不可撤销。
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>取消</AlertDialogCancel>
          <AlertDialogAction class="bg-destructive text-white hover:bg-destructive/90" @click="batchDeleteTasks">
            确认删除
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>

    <!-- 删除确认 (单个) -->
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
