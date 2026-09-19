<script setup lang="ts">
import { ref, onMounted, computed, watch, onUnmounted } from 'vue'
import { Dialog, DialogContent } from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import Pagination from '@/components/Pagination.vue'
import TaskDialog from './TaskDialog.vue'
import RepoDialog from './RepoDialog.vue'
import ApplyDialog from '@/views/apps/ApplyDialog.vue'
import LogViewer from '@/views/history/LogViewer.vue'
import XTerminal from '@/components/XTerminal.vue'
import TaskHeader from './components/TaskHeader.vue'
import TaskList from './components/TaskList.vue'
import TaskDeleteDialog from './components/dialogs/TaskDeleteDialog.vue'
import TaskExportDialog from './components/dialogs/TaskExportDialog.vue'
import { X } from 'lucide-vue-next'
import { api, type Agent, type Task, type TaskLog } from '@/api'
import { toast } from 'vue-sonner'
import { useSiteSettings } from '@/composables/useSiteSettings'
import { useEventBus } from '@/composables/useEventBus'
import { useRouter, useRoute } from 'vue-router'
import { TASK_TYPE, TASK_STATUS } from '@/constants'
import { generateBaihuCommand } from '@/utils/repo-parser'
import { decompressFromBase64 } from '@/utils/decompress'

export interface TaskView {
  name: string
  query: Record<string, any>
  isDefault?: boolean
}

const router = useRouter()
const route = useRoute()
const { pageSize } = useSiteSettings()

// 核心列表与 Agent 数据
const tasks = ref<Task[]>([])
const agents = ref<Agent[]>([])
const loading = ref(false)
const currentPage = ref(1)
const total = ref(0)

// 排序与筛选
const sortBy = ref('created_at')
const order = ref('desc')
const filterName = ref('')
const filterTags = ref('')
const filterType = ref<string>(TASK_TYPE.NORMAL)
const filterSourceId = ref('')
const filterAgentId = ref<string | null>(null)
const filterEnabled = ref<string | undefined>(undefined)
let searchTimer: ReturnType<typeof setTimeout> | null = null

// 自定义视图列表
const taskViews = ref<TaskView[]>([])

// 弹窗状态
const showTaskDialog = ref(false)
const showRepoDialog = ref(false)
const showApplyDialog = ref(false)
const showDeleteDialog = ref(false)
const showBatchDeleteDialog = ref(false)
const showBatchUpdateDialog = ref(false)
const showExportDialog = ref(false)
const showTerminalDialog = ref(false)
const showLogViewer = ref(false)

const editingTask = ref<Partial<Task>>({})
const editingMarketApp = ref<any>(null)
const isEdit = ref(false)
const deleteTaskId = ref<string | null>(null)
const exportCommandText = ref('')
const terminalCmd = ref('')
const executingTaskId = ref<string | null>(null)
const isStopping = ref(false)

// 日志 SSE 变量
const selectedLog = ref<TaskLog | null>(null)
const logContent = ref('')
const logEmptyTitle = ref<string | undefined>(undefined)
const logEmptyDesc = ref<string | undefined>(undefined)
let logSource: EventSource | null = null
let durationTimer: ReturnType<typeof setInterval> | null = null
let logBuffer: string[] = []
let logFlushInterval: ReturnType<typeof setInterval> | null = null

// Agent 映射表
const agentMap = computed(() => {
  const map: Record<string, Agent> = {}
  agents.value.forEach((a: Agent) => { map[a.id] = a })
  return map
})

const filterAgentName = computed(() => {
  if (!filterAgentId.value) return ''
  const agent = agentMap.value[filterAgentId.value]
  return agent ? agent.name : `Agent #${filterAgentId.value}`
})

const displayLogContent = computed(() => {
  if (!logContent.value) return ''
  return decompressFromBase64(logContent.value)
})

async function loadTasks() {
  loading.value = true
  try {
    const res = await api.tasks.list({
      page: currentPage.value,
      page_size: pageSize.value,
      name: filterName.value || undefined,
      tags: filterTags.value || undefined,
      type: filterType.value === 'all' ? undefined : filterType.value,
      source_id: filterSourceId.value || undefined,
      agent_id: filterAgentId.value || undefined,
      enabled: filterEnabled.value,
      sort_by: sortBy.value || undefined,
      order: order.value || undefined
    })
    tasks.value = res.data
    total.value = res.total
  } catch {
    toast.error('加载任务失败')
  } finally {
    loading.value = false
  }
}

function handlePageChange(page: number) {
  currentPage.value = page
  loadTasks()
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

function handleTypeChange(clearSource = true) {
  if (clearSource) {
    filterSourceId.value = ''
  }
  currentPage.value = 1

  const currentQueryType = (route.query.type as string) || ''
  const targetType = (filterType.value === TASK_TYPE.NORMAL || !filterType.value) ? '' : filterType.value

  if (currentQueryType !== targetType) {
    const query = { ...route.query }
    if (targetType) {
      query.type = targetType
    } else {
      delete query.type
    }
    router.replace({ query })
  } else {
    loadTasks()
  }
}

function toggleSort(field: string) {
  if (sortBy.value === field) {
    order.value = order.value === 'asc' ? 'desc' : 'asc'
  } else {
    sortBy.value = field
    order.value = 'desc'
  }
  currentPage.value = 1
  loadTasks()
}

function clearAgentFilter() {
  filterAgentId.value = null
  router.replace({ query: {} })
  currentPage.value = 1
  loadTasks()
}

function openCreate() {
  showBatchUpdateDialog.value = false
  editingTask.value = { name: '', remark: '', command: '', type: TASK_TYPE.NORMAL, schedule: '0 * * * * *', timeout: 30, work_dir: '', enabled: true, clean_config: '', envs: '', random_range: 0 }
  isEdit.value = false
  showTaskDialog.value = true
}

function openCreateRepo() {
  showBatchUpdateDialog.value = false
  editingTask.value = { name: '', remark: '', type: TASK_TYPE.REPO, schedule: '0 0 0 * * *', timeout: 30, enabled: true, clean_config: '', envs: '', random_range: 0 }
  isEdit.value = false
  showRepoDialog.value = true
}

function openEdit(task: Task) {
  showBatchUpdateDialog.value = false
  editingTask.value = { ...task }
  isEdit.value = true
  if (task.type === TASK_TYPE.REPO) {
    showRepoDialog.value = true
  } else {
    showTaskDialog.value = true
  }
}

const editingAppTask = ref<Task | null>(null)

watch(showApplyDialog, (val) => {
  if (!val) {
    editingAppTask.value = null
    editingMarketApp.value = null
  }
})

function openEditApp(task: Task) {
  editingAppTask.value = task
  const cfg = getAppConfig(task)
  editingMarketApp.value = {
    id: task.id,
    name: task.name,
    version: cfg.version || '1.0.0',
    description: task.remark,
    manifest_path: cfg.manifest_path,
    manifest_url: cfg.manifest_url,
    manifest_raw: cfg.manifest_raw,
    scenarios: cfg.scenarios || [],
    env_schema: cfg.env_schema || [],
    build_opts: cfg.build_opts || null,
    env_values: cfg.env_values || {},
    current_scenario: cfg.current_scenario || ''
  }
  showApplyDialog.value = true
}

function getAppConfig(task: any) {
  if (!task || !task.unified_config) return {}
  try {
    const unified = JSON.parse(task.unified_config)
    return unified.app || {}
  } catch { return {} }
}

function uninstallApp(task: Task) {
  confirmDelete(task.id)
}

function filterByApp(sourceId: string) {
  filterSourceId.value = sourceId
  filterType.value = 'all'
  filterTags.value = ''
  filterName.value = ''
  handleTypeChange(false)
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

function openExportDialog(task: Task) {
  const cmd = generateBaihuCommand(task)
  if (!cmd) {
    toast.error('生成导出指令失败')
    return
  }
  exportCommandText.value = cmd
  showExportDialog.value = true
}

function copyCommandText() {
  if (!exportCommandText.value) return
  navigator.clipboard.writeText(exportCommandText.value)
    .then(() => toast.success('指令已复制到剪贴板'))
    .catch(() => toast.error('复制失败'))
}

function confirmDelete(id: string) {
  deleteTaskId.value = id
  showDeleteDialog.value = true
}

function confirmBatchDelete() {
  if (total.value === 0) return
  showBatchDeleteDialog.value = true
}

async function deleteTask(deleteFiles: boolean) {
  if (!deleteTaskId.value) return
  const targetTask = tasks.value.find(t => t.id === deleteTaskId.value)
  try {
    if (targetTask && targetTask.type === TASK_TYPE.APP) {
      await api.apps.remove(targetTask.id, true)
      toast.success('应用及其本地文件夹与关联任务已成功卸载')
    } else {
      await api.tasks.delete(deleteTaskId.value, { delete_files: deleteFiles })
      toast.success('任务已删除')
    }
    loadTasks()
  } catch (err: any) {
    toast.error('操作失败: ' + (err.message || '未知错误'))
  }
  showDeleteDialog.value = false
  deleteTaskId.value = null
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

async function runTask(id: string) {
  if (executingTaskId.value) return
  executingTaskId.value = id
  try {
    const res = await api.tasks.execute(id)
    toast.success('执行指令已发送')
    if (res.log_id) {
      viewLogs(id, res.log_id)
    }
  } catch (error: any) {
    toast.error(error.message || '执行失败')
  } finally {
    executingTaskId.value = null
  }
}

async function toggleTask(task: Task, enabled: boolean) {
  try {
    await api.tasks.update(task.id, { ...task, enabled })
    toast.success(enabled ? '任务已启用' : '任务已禁用')
    loadTasks()
  } catch {
    toast.error('操作失败')
  }
}

async function togglePin(task: Task) {
  const newType = task.pin_type === 'top' ? 'none' : 'top'
  try {
    await api.tasks.update(task.id, { ...task, pin_type: newType })
    toast.success(newType === 'top' ? '已置顶任务' : '已取消置顶')
    loadTasks()
  } catch {
    toast.error('操作失败')
  }
}

async function handleTaskDepInstall(task: Task) {
  try {
    const res = await api.logs.list({ task_id: task.id, page: 1, page_size: 1 })
    if (res.data && res.data.length > 0) {
      const latestLog = res.data[0]
      if (latestLog) {
        const cmdRes = await api.deps.getInstallSuggestCmd(latestLog.id)
        if (cmdRes && cmdRes.command) {
          terminalCmd.value = cmdRes.command
          showTerminalDialog.value = true
        }
      } else {
        toast.error('未找到该任务的运行日志，无法分析依赖')
      }
    } else {
      toast.error('该任务尚未运行，无日志可分析')
    }
  } catch (err: any) {
    toast.error(err.message || '获取依赖安装命令失败')
  }
}

function openBatchUpdate() {
  editingTask.value = { timeout: 30 }
  showBatchUpdateDialog.value = true
  if (filterType.value === TASK_TYPE.REPO) {
    showRepoDialog.value = true
  } else {
    showTaskDialog.value = true
  }
}

// 视图管理
async function loadViewsFromSettings() {
  try {
    const res = await api.settings.getSection('task_qviews')
    const val = res['task_views']
    if (val) {
      taskViews.value = JSON.parse(val)
      if (!route.query.agent_id && !route.query.keyword && !route.query.name && !route.query.tag && !route.query.type && route.query.enabled === undefined) {
        const defaultView = taskViews.value.find((v: any) => v.isDefault)
        if (defaultView) {
          applyViewWithoutSearch(defaultView)
        }
      }
    }
  } catch (e) {
    console.error('Failed to load views', e)
  }
}

async function saveView(name: string) {
  const newView = {
    name,
    query: {
      name: filterName.value,
      tags: filterTags.value,
      agent_id: filterAgentId.value,
      type: filterType.value,
      sort_by: sortBy.value,
      order: order.value
    },
    isDefault: false
  }
  const updatedViews = [...taskViews.value, newView]
  try {
    await api.settings.setSection('task_qviews', {
      'task_views': JSON.stringify(updatedViews)
    })
    taskViews.value = updatedViews
    toast.success('视图已保存')
  } catch (e) {
    toast.error('保存失败')
  }
}

async function deleteView(index: number) {
  const updatedViews = taskViews.value.filter((_: any, i: number) => i !== index)
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

async function toggleDefaultView(index: number) {
  if (!taskViews.value[index]) return
  const isCurrentlyDefault = !!taskViews.value[index]?.isDefault
  const updatedViews = taskViews.value.map((v, i) => ({
    ...v,
    isDefault: i === index ? !isCurrentlyDefault : false
  }))
  try {
    await api.settings.setSection('task_qviews', {
      'task_views': JSON.stringify(updatedViews)
    })
    taskViews.value = updatedViews
    toast.success(!isCurrentlyDefault ? '已设为默认视图' : '已取消默认视图')
  } catch (e) {
    toast.error('设置失败')
  }
}

function applyViewWithoutSearch(view: any) {
  filterName.value = view.query.name || ''
  filterTags.value = view.query.tags || ''
  filterAgentId.value = view.query.agent_id || null
  filterType.value = view.query.type || TASK_TYPE.NORMAL
  sortBy.value = view.query.sort_by || 'created_at'
  order.value = view.query.order || 'desc'
}

function applyView(view: any) {
  applyViewWithoutSearch(view)
  handleSearch()
}

// 日志模块
function cleanupLogSocket() {
  if (logSource) { logSource.close(); logSource = null }
  if (logFlushInterval) { clearInterval(logFlushInterval); logFlushInterval = null }
  if (logBuffer.length > 0) { logContent.value += logBuffer.join('') }
  logBuffer = []
}

function cleanupDurationTimer() {
  if (durationTimer) { clearInterval(durationTimer); durationTimer = null }
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

async function viewLogs(taskId: string, logId?: string) {
  try {
    let latestLog: TaskLog | null = null
    if (logId) {
      const task = tasks.value.find(t => t.id === taskId)
      latestLog = {
        id: logId,
        task_id: taskId,
        task_name: task?.name || '未知任务',
        command: task?.command || '',
        status: TASK_STATUS.RUNNING,
        duration: 0,
        start_time: new Date().toISOString(),
        end_time: '-'
      } as TaskLog
    } else {
      const res = await api.logs.list({ task_id: taskId, page: 1, page_size: 1 })
      if (res.data && res.data.length > 0) {
        latestLog = res.data[0] || null
      }
    }

    if (!latestLog) {
      const task = tasks.value.find(t => t.id === taskId)
      selectedLog.value = {
        id: '',
        task_id: taskId,
        task_name: task?.name || '未知任务',
        command: task?.command || '',
        status: 'UNEXECUTED',
        duration: 0,
        start_time: '-',
        end_time: '-'
      } as TaskLog
      logContent.value = ''
      logEmptyTitle.value = '该任务暂无执行记录'
      logEmptyDesc.value = '此任务尚未被触发执行，目前没有任何运行日志产生。'
      showLogViewer.value = true
      return
    }

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

    logContent.value = 'raw:'
    cleanupLogSocket()
    const protocol = window.location.protocol
    const host = window.location.host
    const baseUrl = (window as any).__BASE_URL__ || ''
    const apiVersion = (window as any).__API_VERSION__ || '/api/v1'
    const sseUrl = `${protocol}//${host}${baseUrl}${apiVersion}/logs/sse?log_id=${latestLog.id}`

    logSource = new EventSource(sseUrl)
    logSource.onmessage = (event) => {
      let parsed: any = null
      let text = ''
      try {
        parsed = JSON.parse(event.data)
        text = parsed.text || ''
      } catch {
        text = event.data
      }
      if (text) logBuffer.push(text)
      if (!logFlushInterval) {
        logFlushInterval = setInterval(() => {
          if (logBuffer.length > 0) {
            logContent.value += logBuffer.join('')
            logBuffer = []
          }
        }, 150)
      }
      if (parsed && parsed.type === 'finish') {
        cleanupDurationTimer()
        cleanupLogSocket()
        if (selectedLog.value) {
          selectedLog.value = {
            ...selectedLog.value,
            status: parsed.status || 'success',
            duration: parsed.duration !== undefined ? parsed.duration : selectedLog.value.duration,
            end_time: parsed.end_time || selectedLog.value.end_time || '-'
          } as TaskLog
        }
        loadTasks()
      }
    }
    logSource.onerror = async () => {
      cleanupLogSocket()
      try {
        const detail = await api.logs.get(latestLog.id)
        if (detail && detail.status !== TASK_STATUS.RUNNING && selectedLog.value) {
          cleanupDurationTimer()
          selectedLog.value = {
            ...selectedLog.value,
            status: detail.status,
            duration: detail.duration,
            end_time: detail.end_time
          } as TaskLog
          loadTasks()
        }
      } catch {}
    }

    cleanupDurationTimer()
    const updateLogStatus = () => {
      if (selectedLog.value && selectedLog.value.status === TASK_STATUS.RUNNING) {
        if (selectedLog.value.start_time && selectedLog.value.start_time !== '-') {
          const startMs = new Date(selectedLog.value.start_time).getTime()
          if (!isNaN(startMs)) {
            selectedLog.value.duration = Date.now() - startMs
          } else {
            selectedLog.value.duration += 1000
          }
        } else {
          selectedLog.value.duration += 1000
        }
      }
    }
    durationTimer = setInterval(updateLogStatus, 1000)
  } catch {
    toast.error('获取日志失败')
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
    loadTasks()
  } finally {
    isStopping.value = false
  }
}

onMounted(async () => {
  await loadAgents()
  const agentIdParam = route.query.agent_id
  if (agentIdParam) filterAgentId.value = String(agentIdParam)
  const keywordParam = route.query.keyword || route.query.name
  if (keywordParam) filterName.value = String(keywordParam)
  const typeParam = route.query.type
  if (typeParam && (typeParam === 'all' || typeParam === TASK_TYPE.NORMAL || typeParam === TASK_TYPE.REPO || typeParam === TASK_TYPE.APP || String(typeParam).startsWith('app:'))) {
    filterType.value = String(typeParam)
  }
  const tagParam = route.query.tag
  if (tagParam) filterTags.value = String(tagParam)
  const enabledParam = route.query.enabled
  if (enabledParam !== undefined) filterEnabled.value = String(enabledParam)

  await loadViewsFromSettings()
  loadTasks()
})

useEventBus(['task_running', 'task_queued', 'task_success', 'task_failed', 'task_timeout', 'task_cancelled'], (payload) => {
  const task = tasks.value.find(t => t.id === payload.task_id)
  if (task) task.running_status = payload.status
  if (selectedLog.value && (selectedLog.value.id === payload.log_id || selectedLog.value.task_id === payload.task_id)) {
    selectedLog.value = {
      ...selectedLog.value,
      status: payload.status,
      duration: payload.duration !== undefined ? payload.duration : selectedLog.value.duration,
      end_time: payload.end_time !== undefined ? payload.end_time : selectedLog.value.end_time
    }
    if (payload.status !== TASK_STATUS.RUNNING) {
      cleanupDurationTimer()
      cleanupLogSocket()
      loadTasks()
    }
  }
})

watch(() => route.query.agent_id, (newVal: any) => {
  filterAgentId.value = newVal ? String(newVal) : null
  currentPage.value = 1
  loadTasks()
})

watch(() => route.query.enabled, (newVal: any) => {
  filterEnabled.value = newVal !== undefined ? String(newVal) : undefined
  currentPage.value = 1
  loadTasks()
})

watch(() => route.query.type, (newVal: any) => {
  if (newVal && (newVal === 'all' || newVal === TASK_TYPE.NORMAL || newVal === TASK_TYPE.REPO || newVal === TASK_TYPE.APP || String(newVal).startsWith('app:'))) {
    filterType.value = String(newVal)
  } else if (!newVal) {
    filterType.value = TASK_TYPE.NORMAL
  }
  currentPage.value = 1
  loadTasks()
})
</script>

<template>
  <div class="space-y-6">
    <!-- 顶部工具栏与过滤器子组件 -->
    <TaskHeader
      v-model:filter-name="filterName"
      v-model:filter-tags="filterTags"
      v-model:filter-type="filterType"
      :filter-agent-id="filterAgentId"
      :filter-agent-name="filterAgentName"
      :task-views="taskViews"
      :loading="loading"
      @search="handleSearch"
      @type-change="handleTypeChange"
      @refresh="loadTasks"
      @clear-agent-filter="clearAgentFilter"
      @open-batch-update="openBatchUpdate"
      @confirm-batch-delete="confirmBatchDelete"
      @open-create-task="openCreate"
      @open-create-repo="openCreateRepo"
      @apply-view="applyView"
      @toggle-default-view="toggleDefaultView"
      @delete-view="deleteView"
      @save-view="saveView"
    />

    <!-- 列表主体容器 -->
    <TaskList
      :tasks="tasks"
      :total="total"
      :current-page="currentPage"
      :page-size="pageSize"
      :filter-type="filterType"
      :sort-by="sortBy"
      :order="order"
      :executing-task-id="executingTaskId"
      :agent-map="agentMap"
      @toggle-sort="toggleSort"
      @run-task="runTask"
      @view-logs="viewLogs"
      @open-edit-task="openEdit"
      @open-edit-app="openEditApp"
      @uninstall-app="uninstallApp"
      @filter-by-app="filterByApp"
      @toggle-task="toggleTask"
      @toggle-pin="togglePin"
      @dep-install="handleTaskDepInstall"
      @duplicate-task="duplicateTask"
      @open-export-dialog="openExportDialog"
      @confirm-delete="confirmDelete"
    />

    <!-- 分页 -->
    <Pagination :total="total" :page="currentPage" @update:page="handlePageChange" />

    <!-- 普通任务弹窗 -->
    <TaskDialog v-model:open="showTaskDialog" :task="editingTask" :is-edit="isEdit" :is-batch="showBatchUpdateDialog" @saved="loadTasks" />

    <!-- 仓库同步弹窗 -->
    <RepoDialog v-model:open="showRepoDialog" :task="editingTask" :is-edit="isEdit" :is-batch="showBatchUpdateDialog" @saved="loadTasks" />

    <!-- 日志弹窗 -->
    <LogViewer v-model:open="showLogViewer"
      title="最新日志"
      variant="full"
      :log="selectedLog"
      :content="displayLogContent"
      :is-stopping="isStopping"
      :empty-title="logEmptyTitle"
      :empty-description="logEmptyDesc"
      @stop="handleStopTask" />

    <!-- 删除确认 (单个 & 批量) -->
    <TaskDeleteDialog
      v-model:open-delete="showDeleteDialog"
      v-model:open-batch-delete="showBatchDeleteDialog"
      :target-task="tasks.find(t => t.id === deleteTaskId)"
      @confirm-delete="deleteTask"
      @confirm-batch-delete="batchDeleteTasks"
    />

    <!-- 导出指令弹窗 -->
    <TaskExportDialog
      v-model:open="showExportDialog"
      :command-text="exportCommandText"
      @copy="copyCommandText"
    />

    <!-- 依赖补全终端 -->
    <Dialog v-model:open="showTerminalDialog">
      <DialogContent class="w-[calc(100%-1rem)] sm:max-w-[90vw] lg:max-w-4xl h-[60vh] sm:h-[80vh] flex flex-col p-0 overflow-hidden bg-[#1e1e1e] border-none shadow-2xl [&>button]:hidden">
        <div class="flex items-center justify-between px-3 py-2 border-b border-[#3c3c3c]">
          <span class="text-xs sm:text-sm font-medium text-gray-300">依赖补全终端</span>
          <Button variant="ghost" size="icon" class="h-6 w-6 text-gray-400 hover:text-white" @click="showTerminalDialog = false">
            <X class="h-4 w-4" />
          </Button>
        </div>
        <div class="flex-1 overflow-hidden">
          <XTerminal v-if="showTerminalDialog" :initial-command="terminalCmd" />
        </div>
      </DialogContent>
    </Dialog>

    <!-- 应用重新配置与编辑弹窗 -->
    <ApplyDialog
      v-model:open="showApplyDialog"
      :target-app="editingMarketApp"
      :task="editingAppTask"
      mode="edit_task"
      @success="loadTasks"
    />
  </div>
</template>
