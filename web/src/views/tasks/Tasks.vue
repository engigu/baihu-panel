<script setup lang="ts">
import { ref, onMounted, computed, watch, onUnmounted } from 'vue'
import { Button } from '@/components/ui/button'
import BaihuDialog from '@/components/ui/BaihuDialog.vue'
import { Input } from '@/components/ui/input'
import Pagination from '@/components/Pagination.vue'
import TaskDialog from './TaskDialog.vue'
import RepoDialog from './RepoDialog.vue'
import LogViewer from '@/views/history/LogViewer.vue'
import { Plus, Play, Pencil, Trash2, Search, ScrollText, GitBranch, Terminal, Server, Monitor, X, Loader2, RefreshCw, Wifi, WifiOff, Zap, ZapOff, Copy, Tag, ChevronDown, Pin, PinOff, MoreHorizontal } from 'lucide-vue-next'
import TagInput from '@/components/TagInput.vue'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
  DropdownMenuSeparator,
} from '@/components/ui/dropdown-menu'
import { api, type Agent, type Task, type TaskLog } from '@/api'
import { toast } from 'vue-sonner'
import { useSiteSettings } from '@/composables/useSiteSettings'
import { useRouter, useRoute } from 'vue-router'
import { TASK_TYPE, AGENT_STATUS, TRIGGER_TYPE, TASK_STATUS } from '@/constants'
import TextOverflow from '@/components/TextOverflow.vue'
import { getCronDescription } from '@/utils/cron'


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
const isStopping = ref(false)

async function runTask(id: string) {
  if (executingTaskId.value) return
  executingTaskId.value = id
  try {
    const res = await api.tasks.execute(id)
    toast.success('执行指令已发送')
    if (res.log_id) {
      // 开启日志查看器
      viewLogs(id)
    }
  } catch (error: any) {
    toast.error(error.message || '执行失败')
  } finally {
    executingTaskId.value = null
  }
}

async function handleStopTask() {
  if (!selectedLog.value || isStopping.value) return
  
  isStopping.value = true
  try {
    await api.tasks.stop(selectedLog.value.id)
    toast.success('停止指令已发送')
  } catch (error: any) {
    toast.error(error.message || '停止失败')
    // 出错时也尝试刷新，因为后端可能已经自动修正了“僵尸”状态
    loadTasks()
  } finally {
    isStopping.value = false
  }
}

async function toggleTask(task: Task, enabled: boolean) {
  try {
    await api.tasks.update(task.id, { ...task, enabled })
    toast.success(enabled ? '任务已启用' : '任务已禁用')
    loadTasks()
  } catch { toast.error('操作失败') }
}

async function togglePin(task: Task) {
  const newType = task.pin_type === 'top' ? 'none' : 'top'
  try {
    await api.tasks.update(task.id, { ...task, pin_type: newType })
    toast.success(newType === 'top' ? '任务已置顶' : '已取消置顶')
    loadTasks()
  } catch { toast.error('置顶操作失败') }
}

const showLogViewer = ref(false)
const selectedLog = ref<TaskLog | null>(null)
const logContent = ref('')
const logEmptyTitle = ref<string | undefined>(undefined)
const logEmptyDesc = ref<string | undefined>(undefined)
let logSocket: WebSocket | null = null
let durationTimer: ReturnType<typeof setInterval> | null = null

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

function cleanupDurationTimer() {
  if (durationTimer) {
    clearInterval(durationTimer)
    durationTimer = null
  }
}

watch(showLogViewer, (val) => {
  if (!val) {
    cleanupLogSocket()
    cleanupDurationTimer()
    logContent.value = ''
  }
})

onUnmounted(() => {
  cleanupLogSocket()
  cleanupDurationTimer()
})

import { decompressFromBase64 } from '@/utils/decompress'

const displayLogContent = computed(() => {
  if (!logContent.value) return ''
  return decompressFromBase64(logContent.value)
})

async function viewLogs(taskId: string) {
  try {
    const res = await api.logs.list({ task_id: taskId, page: 1, page_size: 1 })
    if (res.data && res.data.length > 0) {
      const latestLog = res.data[0]
      if (!latestLog) return
      selectedLog.value = latestLog
      logContent.value = ''
      logEmptyTitle.value = undefined
      logEmptyDesc.value = undefined
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

      // 启动状态轮询
      cleanupDurationTimer()
      const updateLogStatus = async () => {
        try {
          const res = await api.logs.get(latestLog.id)
          if (res && selectedLog.value && selectedLog.value.id === latestLog.id) {
            selectedLog.value.duration = res.duration
            selectedLog.value.status = res.status
            selectedLog.value.end_time = res.end_time
            
            // 如果任务结束，停止轮询
            if (res.status !== TASK_STATUS.RUNNING) {
              cleanupDurationTimer()
              loadTasks() // 同时刷新列表状态
            }
          }
        } catch { /* ignore */ }
      }
      durationTimer = setInterval(updateLogStatus, 3000)
    } else {
      // 如果没有日志，构造一个基础的任务信息对象用于展示弹窗
      const task = tasks.value.find(t => t.id === taskId)
      selectedLog.value = {
        id: '',
        task_id: taskId,
        task_name: task?.name || '未知任务',
        command: task?.command || '',
        status: 'UNEXECUTED',
        duration: 0,
        start_time: '-',
        end_time: '-',
      } as TaskLog
      logContent.value = ''
      logEmptyTitle.value = '该任务暂无执行记录'
      logEmptyDesc.value = '此任务尚未被触发执行，目前没有任何运行日志产生。'
      showLogViewer.value = true
    }
  } catch {
    toast.error('获取日志失败')
  }
}

// 视图管理
const taskViews = ref<any[]>([])
const newViewName = ref('')
const isSavingView = ref(false)

async function loadViewsFromSettings() {
  try {
    const res = await api.settings.getSection('task_qviews')
    const val = res['task_views']
    if (val) {
      taskViews.value = JSON.parse(val)
    }
  } catch (e) {
    console.error('Failed to load views', e)
  }
}

async function saveView() {
  if (!newViewName.value.trim()) {
    toast.error('请输入视图名称')
    return
  }
  
  const newView = {
    name: newViewName.value.trim(),
    query: {
      name: filterName.value,
      tags: filterTags.value,
      agent_id: filterAgentId.value,
      type: filterType.value
    }
  }
  
  const updatedViews = [...taskViews.value, newView]
  isSavingView.value = true
  try {
    await api.settings.setSection('task_qviews', {
      'task_views': JSON.stringify(updatedViews)
    })
    taskViews.value = updatedViews
    newViewName.value = ''
    toast.success('视图已保存')
  } catch (e) {
    toast.error('保存失败')
  } finally {
    isSavingView.value = false
  }
}

function applyView(view: any) {
  filterName.value = view.query.name || ''
  filterTags.value = view.query.tags || ''
  filterAgentId.value = view.query.agent_id || null
  filterType.value = view.query.type || TASK_TYPE.NORMAL
  handleSearch()
}

async function deleteView(index: number) {
  const updatedViews = taskViews.value.filter((_, i) => i !== index)
  try {
    await api.settings.setSection('task_qviews', {
      'task_views': JSON.stringify(updatedViews)
    })
    taskViews.value = updatedViews
    toast.success('视图已删除')
  } catch (e) {
    toast.error('删除失败')
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
  loadViewsFromSettings()
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
    <div class="flex flex-col xl:flex-row xl:items-center justify-between gap-4">
      <div class="flex flex-col shrink-0">
        <Popover>
          <PopoverTrigger as-child>
            <div class="flex items-center gap-2 cursor-pointer group w-fit">
              <h2 class="text-xl sm:text-2xl font-bold tracking-tight">{{ filterType === TASK_TYPE.REPO ? '仓库同步' : '定时任务' }}</h2>
              <div class="flex items-center gap-1 px-1.5 py-0.5 rounded-md bg-muted/50 group-hover:bg-primary/10 transition-colors border border-transparent group-hover:border-primary/20">
                <span class="text-[10px] font-bold text-muted-foreground group-hover:text-primary uppercase tracking-wider">视图</span>
                <ChevronDown class="h-3.5 w-3.5 text-muted-foreground group-hover:text-primary transition-colors" />
              </div>
            </div>
          </PopoverTrigger>
          <PopoverContent class="w-64 p-3 shadow-xl border-muted-foreground/10" align="start" :side-offset="8">
            <div class="space-y-4">
              <div>
                <div class="flex items-center justify-between mb-2 px-1">
                  <h4 class="text-sm font-semibold">我的视图</h4>
                </div>
                <div v-if="taskViews.length === 0" class="text-xs text-muted-foreground px-1 py-4 text-center border-2 border-dashed rounded-md bg-muted/20">
                  暂无保存的视图
                </div>
                <div class="flex flex-wrap gap-2 pr-1 max-h-[200px] overflow-y-auto custom-scrollbar">
                  <div v-for="(view, index) in taskViews" :key="index" 
                    class="flex items-center gap-1.5 pl-2.5 pr-1.5 py-1 bg-primary/5 text-primary rounded-full text-[12px] font-medium border border-primary/10 hover:bg-primary/10 transition-all cursor-pointer group"
                    @click="applyView(view)">
                    <span class="max-w-[120px] truncate">{{ view.name }}</span>
                    <button type="button" class="p-0.5 rounded-full hover:bg-destructive/10 hover:text-destructive transition-colors"
                      @click.stop="deleteView(index)">
                      <X class="h-3 w-3" />
                    </button>
                  </div>
                </div>
              </div>
              
              <div class="pt-3 border-t space-y-2.5">
                <h4 class="text-xs font-semibold px-1 text-muted-foreground uppercase tracking-wider">保存当前过滤为新视图</h4>
                <div class="flex gap-2">
                  <Input v-model="newViewName" placeholder="视图名称..." class="h-9 text-xs bg-muted/30 focus:bg-background" @keydown.enter="saveView" />
                  <Button size="sm" class="h-9 px-3" @click="saveView" :disabled="isSavingView">
                    <Plus v-if="!isSavingView" class="h-4 w-4" />
                    <Loader2 v-else class="h-4 w-4 animate-spin" />
                  </Button>
                </div>
              </div>
            </div>
          </PopoverContent>
        </Popover>
        <p class="text-muted-foreground text-xs mt-0.5 ml-0.5">管理和调度自动化执行任务</p>
      </div>

      <div class="flex flex-row items-center flex-wrap gap-2 w-full xl:w-auto xl:ml-auto xl:justify-end">
        <!-- 搜索与标签 -->
        <div class="flex flex-row items-center gap-2 w-full sm:flex-1 xl:flex-none xl:w-auto text-sm">
          <div class="relative flex-1 xl:flex-none xl:w-[240px] group">
            <Search class="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground group-focus-within:text-primary transition-colors" />
            <Input v-model="filterName" placeholder="搜索任务..." class="h-9 pl-9 w-full bg-muted/20 border-muted-foreground/10 focus:bg-background text-sm"
              @input="handleSearch" />
          </div>
          <TagInput v-model="filterTags" placeholder="搜索标签..." :icon="Tag" multiple
            class="h-9 flex-1 xl:flex-none xl:w-[180px] bg-muted/20 border-muted-foreground/10 focus:bg-background text-sm"
            @enter="handleSearch" @update:modelValue="handleSearch" />
        </div>

        <div class="flex items-center gap-2 w-full sm:w-auto sm:justify-end">
          <!-- 移动端类型切换 -->
          <div class="xl:hidden flex-1 shrink-0">
             <Select v-model="filterType" @update:model-value="(_v: any) => handleTypeChange()">
               <SelectTrigger class="h-9 w-full text-sm bg-muted/20 border-muted-foreground/10">
                 <SelectValue />
               </SelectTrigger>
               <SelectContent>
                 <SelectItem :value="TASK_TYPE.NORMAL">定时任务</SelectItem>
                 <SelectItem :value="TASK_TYPE.REPO">仓库同步</SelectItem>
               </SelectContent>
             </Select>
          </div>

          <div v-if="filterAgentId"
            class="hidden xl:flex items-center gap-1 px-2 py-1 bg-primary/10 text-primary rounded-md text-sm shrink-0">
            <Server class="h-3.5 w-3.5" />
            <span>{{ filterAgentName }}</span>
            <X class="h-3.5 w-3.5 cursor-pointer hover:text-destructive" @click="clearAgentFilter" />
          </div>

          <Button variant="outline" size="icon" class="h-9 w-9 shrink-0" @click="loadTasks" :disabled="loading" title="刷新">
            <RefreshCw class="h-4 w-4" :class="{ 'animate-spin': loading }" />
          </Button>

          <div class="flex items-center gap-2 shrink-0 ml-auto sm:ml-0">
            <Button variant="outline" class="shrink-0 px-2 xl:px-3 h-9 shadow-sm text-destructive border-destructive/20 hover:bg-destructive/10" @click="confirmBatchDelete" title="批量删除">
              <Trash2 class="h-4 w-4 xl:mr-2" /> <span class="hidden xl:inline">批量删除</span>
            </Button>
            <Button v-if="filterType === TASK_TYPE.NORMAL" @click="openCreate" class="shrink-0 px-2 xl:px-3 h-9 shadow-sm font-medium" title="新建任务">
              <Plus class="h-4 w-4 xl:mr-2" /> <span class="hidden xl:inline">新建任务</span>
            </Button>
            <Button v-else-if="filterType === TASK_TYPE.REPO" @click="openCreateRepo" class="shrink-0 px-2 xl:px-3 h-9 shadow-sm font-medium" title="同步仓库">
              <GitBranch class="h-4 w-4 xl:mr-2" /> <span class="hidden xl:inline">同步仓库</span>
            </Button>

            <!-- 桌面端类型切换 -->
            <Tabs :model-value="filterType" @update:model-value="(v: string | number) => { filterType = String(v); handleTypeChange() }" class="shrink-0 hidden xl:block">
            <TabsList class="h-9 p-1 bg-muted/30 border">
                  <TabsTrigger :value="TASK_TYPE.NORMAL" class="px-4 h-7 text-sm">定时任务</TabsTrigger>
                  <TabsTrigger :value="TASK_TYPE.REPO" class="px-4 h-7 text-sm">仓库同步</TabsTrigger>
               </TabsList>
            </Tabs>
          </div>
        </div>
        <!-- 移动端/平板 agent 过滤标签 -->
        <div v-if="filterAgentId"
          class="flex xl:hidden items-center gap-1 px-2 py-1 bg-primary/10 text-primary rounded-md text-sm w-fit mt-1">
          <Server class="h-3.5 w-3.5" />
          <span>{{ filterAgentName }}</span>
          <X class="h-3.5 w-3.5 cursor-pointer hover:text-destructive" @click="clearAgentFilter" />
        </div>
      </div>
    </div>

    <div class="rounded-lg border bg-card overflow-hidden">
      <!-- ========== 1. 大屏布局 (Large >= 1280px) ========== -->
      <div class="hidden xl:block">
        <!-- 表头 -->
        <div class="flex items-center gap-4 px-4 py-1.5 border-b bg-muted/20 text-xs text-muted-foreground font-medium">
          <span class="w-12 shrink-0 pl-1">序号</span>
          <span class="w-8 shrink-0 text-center">类型</span>
          <span class="w-56 shrink-0">名称</span>
          <span class="w-32 shrink-0">执行位置</span>
          <span class="flex-1 min-w-0 flex items-center gap-1.5 line-clamp-1">
            <GitBranch v-if="filterType === TASK_TYPE.REPO" class="h-3.5 w-3.5 opacity-50" />
            <Terminal v-else class="h-3.5 w-3.5 opacity-50" />
            {{ filterType === TASK_TYPE.REPO ? '仓库地址' : '命令内容' }}
          </span>
          <span class="w-28 shrink-0">{{ filterType === TASK_TYPE.REPO ? '同步周期' : '定时规则' }}</span>
          <span class="w-40 shrink-0">执行时间</span>
          <span class="w-8 shrink-0 text-center">状态</span>
          <span class="w-24 shrink-0 text-center">操作</span>
        </div>
        <!-- 列表 -->
        <div class="divide-y text-sm">
          <div v-for="(task, index) in tasks" :key="`large-${task.id}`"
            class="flex items-center gap-2 px-4 py-1.5 hover:bg-muted/30 transition-colors">
            <div v-if="task.running_status === 'running'" class="h-2 w-2 rounded-full bg-amber-500 animate-pulse shadow-[0_0_8px_rgba(245,158,11,0.5)] shrink-0" title="运行中" />
            <div v-else class="h-1.5 w-1.5 rounded-full bg-muted-foreground/20 shrink-0" />
            <div class="w-12 shrink-0 text-muted-foreground tabular-nums">#{{ total - (currentPage - 1) * pageSize - index }}</div>
            <span class="w-8 shrink-0 flex justify-center" :title="getTaskTypeTitle(task.type || 'task')">
              <div class="relative">
                <GitBranch v-if="task.type === TASK_TYPE.REPO" class="h-4 w-4 text-primary" />
                <Terminal v-else class="h-4 w-4 text-primary" />
              </div>
            </span>
            <div class="w-56 shrink-0 flex flex-col justify-center gap-0.5 overflow-hidden">
              <div class="flex items-center gap-1.5 overflow-hidden">
                <span class="font-medium truncate cursor-help" :title="task.name">{{ task.name }}</span>
                <Pin v-if="task.pin_type === 'top'" class="h-3 w-3 text-primary fill-primary shrink-0 rotate-45" />
              </div>
              <div v-if="task.tags" class="flex items-center gap-1 overflow-hidden">
                <span v-for="tag in task.tags.split(',').filter(Boolean).slice(0, 3)" :key="tag"
                  class="truncate text-[10px] leading-none px-1 py-0.5 bg-secondary text-secondary-foreground rounded border">{{ tag }}</span>
              </div>
            </div>
            <span class="w-32 shrink-0 flex items-center gap-1 text-xs" :title="getExecutorName(task)">
              <Monitor v-if="!task.agent_id" class="h-3 w-3 text-muted-foreground" />
              <template v-else>
                <Wifi v-if="getExecutorStatus(task) === 'online'" class="h-3 w-3 text-green-500" />
                <WifiOff v-else class="h-3 w-3 text-muted-foreground" />
              </template>
              <span class="truncate">{{ getExecutorName(task) }}</span>
            </span>
            <code class="flex-1 min-w-0 text-muted-foreground truncate text-xs bg-muted/40 px-2 py-1 rounded">
              <TextOverflow :text="task.command" :title="task.type === TASK_TYPE.REPO ? '仓库地址' : '执行命令'" class="truncate" />
            </code>
            <div class="w-28 shrink-0 flex flex-col items-start justify-center gap-1 overflow-hidden">
              <span v-if="task.trigger_type === TRIGGER_TYPE.BAIHU_STARTUP" class="text-[10px] leading-tight bg-blue-500/10 text-blue-500 px-2 py-1 rounded-md">服务启动时</span>
              <code v-else-if="task.schedule" class="text-muted-foreground text-xs bg-muted/40 px-1.5 py-0.5 rounded truncate font-mono tabular-nums">{{ task.schedule }}</code>
            </div>
            <div class="w-40 shrink-0 flex flex-col justify-center gap-0.5 text-[11px] text-muted-foreground tabular-nums">
              <span class="truncate">上: {{ task.last_run || '-' }}</span>
              <span class="truncate">下: {{ task.next_run || '-' }}</span>
            </div>
            <span class="w-8 flex justify-center shrink-0 cursor-pointer group" @click="toggleTask(task, !task.enabled)">
              <div v-if="task.enabled" class="h-6 w-6 rounded-md bg-green-500/10 flex items-center justify-center group-hover:bg-green-500/20">
                <Zap class="h-3.5 w-3.5 text-green-500 fill-green-500" />
              </div>
              <div v-else class="h-6 w-6 rounded-md bg-muted flex items-center justify-center group-hover:bg-muted/80">
                <ZapOff class="h-3.5 w-3.5 text-muted-foreground" />
              </div>
            </span>
            <span class="w-24 shrink-0 flex justify-center">
              <Button variant="ghost" size="icon" class="h-6 w-6" @click="runTask(task.id)" :disabled="executingTaskId === task.id">
                <Loader2 v-if="executingTaskId === task.id" class="h-3 w-3 animate-spin" />
                <Play v-else class="h-3 w-3" />
              </Button>
              <Button variant="ghost" size="icon" class="h-6 w-6" @click="viewLogs(task.id)"><ScrollText class="h-3 w-3" /></Button>
              <Button variant="ghost" size="icon" class="h-6 w-6" @click="openEdit(task)"><Pencil class="h-3 w-3" /></Button>
              
              <DropdownMenu>
                <DropdownMenuTrigger as-child>
                  <Button variant="ghost" size="icon" class="h-6 w-6"><MoreHorizontal class="h-3 w-3" /></Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="end" class="w-32">
                  <DropdownMenuItem @click="togglePin(task)">
                    <Pin v-if="task.pin_type !== 'top'" class="h-3.5 w-3.5 mr-2" />
                    <PinOff v-else class="h-3.5 w-3.5 mr-2 text-primary" />
                    <span>{{ task.pin_type === 'top' ? '取消置顶' : '置顶任务' }}</span>
                  </DropdownMenuItem>
                  <DropdownMenuItem @click="duplicateTask(task)">
                    <Copy class="h-3.5 w-3.5 mr-2" />
                    <span>复制任务</span>
                  </DropdownMenuItem>
                  <DropdownMenuSeparator />
                  <DropdownMenuItem class="text-destructive focus:text-destructive" @click="confirmDelete(task.id)">
                    <Trash2 class="h-3.5 w-3.5 mr-2" />
                    <span>删除任务</span>
                  </DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>
            </span>
          </div>
        </div>
      </div>

      <!-- ========== 2. 中屏布局 (Medium 640px - 1280px) ========== -->
      <div class="hidden sm:block xl:hidden">
        <!-- 表头 -->
        <div class="flex items-center gap-4 px-4 py-1.5 border-b bg-muted/20 text-xs text-muted-foreground font-medium">
          <span class="w-12 shrink-0 pl-1">序号</span>
          <span class="w-48 shrink-0">任务信息</span>
          <span class="flex-1 min-w-0">
            {{ filterType === TASK_TYPE.REPO ? '仓库地址' : '命令内容' }}
          </span>
          <span class="w-8 shrink-0 text-center">状态</span>
          <span class="w-24 shrink-0 text-center">操作</span>
        </div>
        <!-- 列表 -->
        <div class="divide-y text-sm">
          <div v-for="(task, index) in tasks" :key="`medium-${task.id}`"
            class="flex items-center gap-2 px-4 py-2.5 hover:bg-muted/30 transition-colors">
            <div v-if="task.running_status === 'running'" class="h-1.5 w-1.5 rounded-full bg-amber-500 animate-pulse shadow-[0_0_8px_rgba(245,158,11,0.5)] shrink-0" />
            <div v-else class="h-1 w-1 rounded-full bg-muted-foreground/20 shrink-0" />
            <div class="w-12 shrink-0 text-muted-foreground tabular-nums text-xs">#{{ total - (currentPage - 1) * pageSize - index }}</div>
            <div class="w-48 shrink-0 flex items-center gap-2 overflow-hidden">
              <span class="shrink-0" :title="getTaskTypeTitle(task.type || 'task')">
                <div class="relative">
                  <GitBranch v-if="task.type === TASK_TYPE.REPO" class="h-3.5 w-3.5 text-primary" />
                  <Terminal v-else class="h-3.5 w-3.5 text-primary" />
                </div>
              </span>
              <div class="flex flex-col min-w-0">
                <div class="flex items-center gap-1.5 overflow-hidden">
                  <span class="font-medium truncate">{{ task.name }}</span>
                  <Pin v-if="task.pin_type === 'top'" class="h-3 w-3 text-primary fill-primary shrink-0 rotate-45" />
                </div>
                <span v-if="task.schedule" class="text-[10px] text-muted-foreground font-mono truncate">{{ task.schedule }}</span>
              </div>
            </div>
            <code class="flex-1 min-w-0 text-[11px] text-muted-foreground bg-muted/20 px-2 py-1 rounded truncate">
              {{ task.command }}
            </code>
            <span class="w-8 flex justify-center shrink-0 cursor-pointer group" @click="toggleTask(task, !task.enabled)">
              <div v-if="task.enabled" class="h-6 w-6 rounded-md bg-green-500/5 flex items-center justify-center">
                <Zap class="h-3.5 w-3.5 text-green-500" />
              </div>
              <div v-else class="h-6 w-6 rounded-md bg-muted flex items-center justify-center">
                <ZapOff class="h-3.5 w-3.5 text-muted-foreground" />
              </div>
            </span>
            <div class="w-24 shrink-0 flex justify-center">
              <Button variant="ghost" size="icon" class="h-6 w-6" @click="runTask(task.id)" :disabled="executingTaskId === task.id">
                <Loader2 v-if="executingTaskId === task.id" class="h-3 w-3 animate-spin" />
                <Play v-else class="h-3 w-3" />
              </Button>
              <Button variant="ghost" size="icon" class="h-6 w-6" @click="viewLogs(task.id)"><ScrollText class="h-3 w-3" /></Button>
              <Button variant="ghost" size="icon" class="h-6 w-6" @click="openEdit(task)"><Pencil class="h-3 w-3" /></Button>
              
              <DropdownMenu>
                <DropdownMenuTrigger as-child>
                  <Button variant="ghost" size="icon" class="h-6 w-6"><MoreHorizontal class="h-3 w-3" /></Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="end" class="w-32">
                  <DropdownMenuItem @click="togglePin(task)">
                    <Pin v-if="task.pin_type !== 'top'" class="h-3.5 w-3.5 mr-2" />
                    <PinOff v-else class="h-3.5 w-3.5 mr-2 text-primary" />
                    <span>{{ task.pin_type === 'top' ? '取消置顶' : '置顶任务' }}</span>
                  </DropdownMenuItem>
                  <DropdownMenuItem @click="duplicateTask(task)">
                    <Copy class="h-3.5 w-3.5 mr-2" />
                    <span>复制任务</span>
                  </DropdownMenuItem>
                  <DropdownMenuSeparator />
                  <DropdownMenuItem class="text-destructive focus:text-destructive" @click="confirmDelete(task.id)">
                    <Trash2 class="h-3.5 w-3.5 mr-2" />
                    <span>删除任务</span>
                  </DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>
            </div>
          </div>
        </div>
      </div>

      <!-- ========== 3. 小屏布局 (Small < 640px) ========== -->
      <div class="divide-y sm:hidden">
        <div v-if="tasks.length === 0" class="text-sm text-muted-foreground text-center py-8">暂无任务</div>
        <div v-for="(task, index) in tasks" :key="`small-${task.id}`" class="p-3 hover:bg-muted/50 transition-colors">
          <div class="flex items-start justify-between mb-3 border-b border-border/40 pb-2">
            <div class="flex items-center gap-2 flex-1 min-w-0 pr-2">
              <div v-if="task.running_status === 'running'" class="h-1.5 w-1.5 rounded-full bg-amber-500 animate-pulse shadow-[0_0_8px_rgba(245,158,11,0.5)] shrink-0" />
              <div v-else class="h-1 w-1 rounded-full bg-muted-foreground/20 shrink-0" />
              <span class="text-xs text-muted-foreground tabular-nums flex-shrink-0">#{{ total - (currentPage - 1) * pageSize - index }}</span>
              <span class="shrink-0">
                <div class="relative">
                  <GitBranch v-if="task.type === TASK_TYPE.REPO" class="h-3.5 w-3.5 text-primary" />
                  <Terminal v-else class="h-3.5 w-3.5 text-primary" />
                </div>
              </span>
              <div class="flex items-center gap-1.5 min-w-0 flex-1">
                <span class="font-bold text-sm truncate">{{ task.name }}</span>
                <Pin v-if="task.pin_type === 'top'" class="h-3 w-3 text-primary fill-primary shrink-0 rotate-45" />
              </div>
            </div>
            <span @click="toggleTask(task, !task.enabled)" class="cursor-pointer">
              <div v-if="task.enabled" class="h-6 w-6 rounded-md bg-green-500/10 flex items-center justify-center"><Zap class="h-3.5 w-3.5 text-green-500 fill-green-500" /></div>
              <div v-else class="h-6 w-6 rounded-md bg-muted flex items-center justify-center"><ZapOff class="h-3.5 w-3.5 text-muted-foreground" /></div>
            </span>
          </div>
          <div class="space-y-1.5 text-xs text-muted-foreground mb-3 px-1">
            <div class="flex items-center gap-3">
              <span class="w-10 shrink-0 font-medium opacity-70">{{ task.type === TASK_TYPE.REPO ? '周期:' : '定时:' }}</span>
              <span v-if="task.trigger_type === TRIGGER_TYPE.BAIHU_STARTUP" class="text-[10px] leading-tight bg-blue-500/10 text-blue-500 px-1.5 py-0.5 rounded font-medium">服务启动时</span>
              <div v-else-if="task.schedule" class="flex items-center gap-1.5 flex-1 min-w-0">
                <span class="text-xs text-foreground bg-muted/40 px-1.5 py-0.5 rounded shrink-0">{{ task.schedule }}</span>
                <div class="flex items-center gap-1 text-[10px] text-muted-foreground/60 min-w-0 flex-1">
                  <Zap class="h-2.5 w-2.5 fill-current opacity-50 shrink-0" />
                  <TextOverflow :text="getCronDescription(task.schedule)" class="truncate" />
                </div>
              </div>
            </div>
            <div class="flex items-start gap-3">
              <span class="w-10 shrink-0 font-medium mt-0.5 opacity-70">{{ task.type === TASK_TYPE.REPO ? '地址:' : '命令:' }}</span>
              <div class="flex-1 min-w-0 overflow-hidden text-foreground"><TextOverflow :text="task.command" class="truncate opacity-80" /></div>
            </div>
            <div v-if="task.remark" class="flex items-start gap-3">
              <span class="w-10 shrink-0 font-medium mt-0.5 opacity-70">备注:</span>
              <span class="flex-1 text-[11px] truncate">{{ task.remark }}</span>
            </div>
          </div>
          <div class="grid grid-cols-4 items-center pt-2 mt-2 border-t border-border/40 -mx-1">
            <Button variant="ghost" class="h-9 px-0 text-xs gap-1.5 hover:bg-primary/5 rounded-none" @click="runTask(task.id)" :disabled="executingTaskId === task.id">
              <Loader2 v-if="executingTaskId === task.id" class="h-3.5 w-3.5 animate-spin" />
              <Play v-else class="h-3.5 w-3.5" />{{ task.type === TASK_TYPE.REPO ? '同步' : '执行' }}
            </Button>
            <Button variant="ghost" class="h-9 px-0 text-xs gap-1.5 hover:bg-primary/5 rounded-none border-l border-border/10" @click="viewLogs(task.id)">
              <ScrollText class="h-3.5 w-3.5" />日志
            </Button>
            <Button variant="ghost" class="h-9 px-0 text-xs gap-1.5 hover:bg-primary/5 rounded-none border-l border-border/10" @click="openEdit(task)">
              <Pencil class="h-3.5 w-3.5" />编辑
            </Button>
            
            <DropdownMenu>
              <DropdownMenuTrigger as-child>
                <Button variant="ghost" class="h-9 px-0 text-xs gap-1.5 hover:bg-primary/5 rounded-none border-l border-border/10 w-full">
                  <MoreHorizontal class="h-3.5 w-3.5" />更多
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end" class="w-40">
                <DropdownMenuItem @click="togglePin(task)">
                  <Pin v-if="task.pin_type !== 'top'" class="h-4 w-4 mr-2" />
                  <PinOff v-else class="h-4 w-4 mr-2 text-primary" />
                  <span>{{ task.pin_type === 'top' ? '取消置顶' : '置顶任务' }}</span>
                </DropdownMenuItem>
                <DropdownMenuItem @click="duplicateTask(task)">
                  <Copy class="h-4 w-4 mr-2" />
                  <span>复制任务</span>
                </DropdownMenuItem>
                <DropdownMenuSeparator />
                <DropdownMenuItem class="text-destructive focus:text-destructive" @click="confirmDelete(task.id)">
                  <Trash2 class="h-4 w-4 mr-2" />
                  <span>删除任务</span>
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
        </div>
      </div>
    </div>

    <!-- 分页 -->
    <Pagination :total="total" :page="currentPage" @update:page="handlePageChange" />

    <!-- 普通任务弹窗 -->
    <TaskDialog v-model:open="showTaskDialog" :task="editingTask" :is-edit="isEdit" @saved="loadTasks" />

    <!-- 仓库同步弹窗 -->
    <RepoDialog v-model:open="showRepoDialog" :task="editingTask" :is-edit="isEdit" @saved="loadTasks" />

    <LogViewer v-model:open="showLogViewer"
      title="最新日志"
      variant="full"
      :log="selectedLog"
      :content="displayLogContent"
      :is-stopping="isStopping"
      :empty-title="logEmptyTitle"
      :empty-description="logEmptyDesc"
      @stop="handleStopTask" />


    <!-- 删除确认 (批量) -->
    <BaihuDialog v-model:open="showBatchDeleteDialog" title="确认批量删除">
      <div class="text-sm text-muted-foreground leading-relaxed">
        将会删除当前所有过滤条件下匹配的 <b class="text-foreground text-lg px-1">{{ total }}</b> 个任务。
        <p class="mt-2 text-destructive font-medium">⚠️ 操作不可撤销，请谨慎操作。</p>
      </div>
      <template #footer>
        <Button variant="ghost" @click="showBatchDeleteDialog = false">取消</Button>
        <Button variant="destructive" class="shadow-lg shadow-destructive/20" @click="batchDeleteTasks">确认批量删除</Button>
      </template>
    </BaihuDialog>

    <!-- 删除确认 (单个) -->
    <BaihuDialog v-model:open="showDeleteDialog" title="确认删除任务">
      <div class="text-sm text-muted-foreground leading-relaxed">
        确定要删除任务 <b class="text-foreground">{{ tasks.find(t => t.id === deleteTaskId)?.name }}</b> 吗？
        <p class="mt-2 text-destructive font-medium">⚠️ 此操作无法撤销。</p>
      </div>
      <template #footer>
        <Button variant="ghost" @click="showDeleteDialog = false">取消</Button>
        <Button variant="destructive" class="shadow-lg shadow-destructive/20" @click="deleteTask">确认删除</Button>
      </template>
    </BaihuDialog>
  </div>
</template>
