<script setup lang="ts">
import { ref, onMounted, computed, watch, nextTick } from 'vue'
import { useRoute } from 'vue-router'
import { TASK_STATUS, TASK_TYPE } from '@/constants'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import Pagination from '@/components/Pagination.vue'
import LogViewer from './LogViewer.vue'
import {
  RefreshCw, X, Search, Maximize2, GitBranch, Terminal,
  CheckCircle2, XCircle, AlertCircle, Ban, Clock, Zap as ZapIcon, Check
} from 'lucide-vue-next'
import { api, type TaskLog } from '@/api'
import { Badge } from '@/components/ui/badge'
import { toast } from 'vue-sonner'
import { useSiteSettings } from '@/composables/useSiteSettings'
import TextOverflow from '@/components/TextOverflow.vue'

const route = useRoute()
const { pageSize } = useSiteSettings()

const logs = ref<TaskLog[]>([])
const selectedLog = ref<TaskLog | null>(null)
const filterKeyword = ref('')
const filterTaskId = ref<number | undefined>(undefined)
const currentPage = ref(1)
const total = ref(0)

let searchTimer: ReturnType<typeof setTimeout> | null = null
let durationTimer: ReturnType<typeof setInterval> | null = null

// 全屏查看
const showFullscreen = ref(false)

const wsContent = ref('')
const isWsLoading = ref(false)
let logSocket: WebSocket | null = null


const decompressedOutput = computed(() => {
  return wsContent.value || '无输出'
})

async function loadLogs() {
  try {
    const params: { page: number; page_size: number; task_id?: number; task_name?: string } = {
      page: currentPage.value,
      page_size: pageSize.value
    }
    if (filterTaskId.value) {
      params.task_id = filterTaskId.value
    }
    if (filterKeyword.value.trim()) {
      params.task_name = filterKeyword.value.trim()
    }
    const response = await api.logs.list(params)
    logs.value = response.data
    total.value = response.total
  } catch {
    toast.error('加载日志失败')
  }
}

function handleSearch() {
  if (searchTimer) clearTimeout(searchTimer)
  searchTimer = setTimeout(() => {
    currentPage.value = 1
    loadLogs()
  }, 300)
}

function handlePageChange(page: number) {
  currentPage.value = page
  loadLogs()
}

async function selectLog(log: TaskLog) {
  if (logSocket) {
    logSocket.onopen = null
    logSocket.onmessage = null
    logSocket.onerror = null
    logSocket.onclose = null
    logSocket.close()
  }

  // 清理旧定时器
  if (durationTimer) {
    clearInterval(durationTimer)
    durationTimer = null
  }

  selectedLog.value = log

  // 如果是运行中状态，启动定时器轮询最新日志信息（主要是更新耗时）
  if (log.status === TASK_STATUS.RUNNING) {
    const updateLog = async () => {
      try {
        const res = await api.logs.get(log.id)
        if (res && selectedLog.value && selectedLog.value.id === log.id) {
          // 只更新需要变动的字段
          selectedLog.value.duration = res.duration
          // 同步更新列表中的数据
          const listItem = logs.value.find(l => l.id === log.id)
          if (listItem) {
            listItem.duration = res.duration
          }
          // 如果状态变了，更新状态并停止轮询
          if (res.status !== TASK_STATUS.RUNNING) {
            selectedLog.value.status = res.status
            selectedLog.value.end_time = res.end_time
            if (listItem) {
              listItem.status = res.status
              listItem.end_time = res.end_time
            }
            if (durationTimer) {
              clearInterval(durationTimer)
              durationTimer = null
            }
          }
        }
      } catch { /* ignore */ }
    }
    durationTimer = setInterval(updateLog, 3000)
  }

  wsContent.value = ''
  isWsLoading.value = true

  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  const host = window.location.host
  const baseUrl = (window as any).__BASE_URL__ || ''
  const apiVersion = (window as any).__API_VERSION__ || '/api/v1'
  const wsUrl = `${protocol}//${host}${baseUrl}${apiVersion}/logs/ws?log_id=${log.id}`

  logSocket = new WebSocket(wsUrl)

  logSocket.onopen = () => {
    isWsLoading.value = false
    console.log('[LogWS] Connection opened')
  }

  logSocket.onmessage = (event) => {
    isWsLoading.value = false
    if (log.status !== TASK_STATUS.RUNNING) {
      wsContent.value = event.data
    } else {
      wsContent.value += event.data
      // 自动滚动到底部
      nextTick(() => {
        const pre = document.querySelector('.log-pre')
        if (pre) pre.scrollTop = pre.scrollHeight
      })
    }
  }

  logSocket.onerror = (e) => {
    isWsLoading.value = false
    console.error('[LogWS] Connection error', e)
    toast.error('日志连接异常')
  }

  logSocket.onclose = (e) => {
    isWsLoading.value = false
    console.log('[LogWS] Connection closed', e.code, e.reason)
  }
}

function closeDetail() {
  if (durationTimer) {
    clearInterval(durationTimer)
    durationTimer = null
  }
  if (logSocket) {
    logSocket.onopen = null
    logSocket.onmessage = null
    logSocket.onerror = null
    logSocket.onclose = null
    logSocket.close()
    logSocket = null
  }
  selectedLog.value = null
  wsContent.value = ''
}

const isStopping = ref(false)
async function stopTask() {
  if (!selectedLog.value || isStopping.value) return

  try {
    isStopping.value = true
    await api.tasks.stop(selectedLog.value.id)
    toast.success('停止请求已发送')
  } catch (err: any) {
    toast.error(err.message || '停止失败')
  } finally {
    isStopping.value = false
  }
}

function formatDuration(ms: number): string {
  if (ms < 1000) return `${ms}毫秒`
  if (ms < 60000) return `${(ms / 1000).toFixed(1)}秒`
  return `${(ms / 60000).toFixed(1)}分钟`
}

function getStatusBadgeClass(status: string) {
  switch (status) {
    case TASK_STATUS.SUCCESS:
      return 'bg-green-500/10 text-green-700 border-green-200/50 dark:bg-green-500/20 dark:text-green-400 dark:border-green-900/50'
    case TASK_STATUS.FAILED:
      return 'bg-red-500/10 text-red-700 border-red-200/50 dark:bg-red-500/20 dark:text-red-400 dark:border-red-900/50'
    case TASK_STATUS.RUNNING:
      return 'bg-blue-500/10 text-blue-700 border-blue-200/50 dark:bg-blue-500/20 dark:text-blue-400 dark:border-blue-900/50'
    case TASK_STATUS.PENDING:
      return 'bg-amber-500/10 text-amber-700 border-amber-200/50 dark:bg-amber-500/20 dark:text-amber-400 dark:border-amber-900/50'
    case TASK_STATUS.TIMEOUT:
      return 'bg-orange-500/10 text-orange-700 border-orange-200/50 dark:bg-orange-500/20 dark:text-orange-400 dark:border-orange-900/50'
    case TASK_STATUS.CANCELLED:
      return 'bg-muted text-muted-foreground border-transparent'
    default:
      return 'bg-secondary text-secondary-foreground border-transparent'
  }
}

function getTaskTypeTitle(type: string) {
  return type === TASK_TYPE.REPO ? '仓库同步' : '普通任务'
}

onMounted(() => {
  // 从 URL 读取 task_id 参数
  const taskIdParam = route.query.task_id
  if (taskIdParam) {
    filterTaskId.value = Number(taskIdParam)
  }
  loadLogs()
})

// 监听路由变化
watch(() => route.query.task_id, (newTaskId) => {
  filterTaskId.value = newTaskId ? Number(newTaskId) : undefined
  currentPage.value = 1
  loadLogs()
})
</script>

<template>
  <div class="space-y-6">
    <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
      <div>
        <h2 class="text-xl sm:text-2xl font-bold tracking-tight">执行历史</h2>
        <p class="text-muted-foreground text-sm">查看任务执行记录和日志</p>
      </div>
      <div class="flex items-center gap-2">
        <div class="relative flex-1 sm:flex-none">
          <Search class="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
          <Input v-model="filterKeyword" placeholder="搜索任务..." class="h-9 pl-9 w-full sm:w-56 text-sm"
            @input="handleSearch" />
        </div>
        <Button variant="outline" size="icon" class="h-9 w-9 shrink-0" @click="loadLogs">
          <RefreshCw class="h-4 w-4" />
        </Button>
      </div>
    </div>

    <div class="flex flex-col lg:flex-row gap-4">
      <!-- 日志列表 -->
      <div class="flex-1 min-w-0 rounded-lg border bg-card overflow-hidden">
        <!-- 小屏表头 -->
        <div
          class="flex sm:hidden items-center gap-2 px-3 py-2 border-b bg-muted/20 text-xs text-muted-foreground font-medium">
          <span class="w-14 shrink-0">ID</span>
          <span class="w-10 shrink-0 text-center">类型</span>
          <span class="flex-1 min-w-0">任务名称</span>
          <span class="w-8 shrink-0 text-center">状态</span>
          <span class="w-12 text-right shrink-0">耗时</span>
        </div>
        <!-- 大屏表头 -->
        <div
          class="hidden sm:flex items-center gap-4 px-4 py-2 border-b bg-muted/20 text-sm text-muted-foreground font-medium">
          <span class="w-16 shrink-0">ID</span>
          <span class="w-12 shrink-0 text-center">类型</span>
          <span class="w-36 shrink-0">任务名称</span>
          <span class="flex-1 min-w-0">命令</span>
          <span class="w-12 shrink-0 text-center">状态</span>
          <span class="w-16 text-right shrink-0">耗时</span>
          <span v-if="!selectedLog" class="w-40 text-right shrink-0 hidden md:block">执行时间</span>
        </div>
        <!-- 列表 -->
        <div class="divide-y">
          <div v-if="logs.length === 0" class="text-sm text-muted-foreground text-center py-8">
            暂无日志
          </div>
          <div v-for="log in logs" :key="log.id" :class="[
            'cursor-pointer hover:bg-muted/30 transition-colors',
            selectedLog?.id === log.id && 'bg-accent/50'
          ]" @click="selectLog(log)">
            <!-- 小屏行 -->
            <div class="flex sm:hidden items-center gap-2 px-3 py-2">
              <span class="w-14 shrink-0 text-muted-foreground text-xs">#{{ log.id }}</span>
              <span class="w-6 shrink-0 flex justify-center" :title="getTaskTypeTitle(log.task_type || 'task')">
                <GitBranch v-if="log.task_type === TASK_TYPE.REPO" class="h-3.5 w-3.5 text-primary" />
                <Terminal v-else class="h-3.5 w-3.5 text-primary" />
              </span>
              <span class="flex-1 min-w-0 font-medium truncate text-xs">{{ log.task_name }}</span>
              <span class="w-8 flex justify-center shrink-0">
                <div v-if="log.status === TASK_STATUS.SUCCESS"
                  class="h-5 w-5 rounded-full bg-green-500/10 flex items-center justify-center">
                  <Check class="h-3 w-3 text-green-500 stroke-[3]" />
                </div>
                <div v-else-if="log.status === TASK_STATUS.FAILED"
                  class="h-5 w-5 rounded-full bg-red-500/10 flex items-center justify-center">
                  <X class="h-3 w-3 text-red-500 stroke-[3]" />
                </div>
                <div v-else-if="log.status === TASK_STATUS.RUNNING"
                  class="h-5 w-5 rounded-full bg-yellow-500/10 flex items-center justify-center">
                  <ZapIcon class="h-3 w-3 text-yellow-500 fill-yellow-500 animate-pulse" />
                </div>
                <div v-else-if="log.status === TASK_STATUS.PENDING"
                  class="h-5 w-5 rounded-full bg-yellow-500/10 flex items-center justify-center">
                  <Clock class="h-3 w-3 text-yellow-500" />
                </div>
                <div v-else-if="log.status === TASK_STATUS.TIMEOUT"
                  class="h-5 w-5 rounded-full bg-orange-500/10 flex items-center justify-center">
                  <AlertCircle class="h-3 w-3 text-orange-500" />
                </div>
                <div v-else-if="log.status === TASK_STATUS.CANCELLED"
                  class="h-5 w-5 rounded-full bg-muted flex items-center justify-center">
                  <Ban class="h-3 w-3 text-muted-foreground" />
                </div>
              </span>
              <span class="w-12 text-right shrink-0 text-muted-foreground text-xs">{{ formatDuration(log.duration)
                }}</span>
            </div>
            <!-- 大屏行 -->
            <div class="hidden sm:flex items-center gap-4 px-4 py-2">
              <span class="w-16 shrink-0 text-muted-foreground text-sm">#{{ log.id }}</span>
              <span class="w-10 shrink-0 flex justify-center" :title="getTaskTypeTitle(log.task_type || 'task')">
                <GitBranch v-if="log.task_type === TASK_TYPE.REPO" class="h-4 w-4 text-primary" />
                <Terminal v-else class="h-4 w-4 text-primary" />
              </span>
              <span class="w-36 shrink-0 font-medium truncate text-sm">{{ log.task_name }}</span>
              <code class="flex-1 min-w-0 text-muted-foreground truncate text-xs bg-muted/40 px-2 py-1 rounded">
                <TextOverflow :text="log.command" title="执行命令" />
              </code>
              <span class="w-12 flex justify-center shrink-0">
                <div v-if="log.status === TASK_STATUS.SUCCESS"
                  class="h-6 w-6 rounded-full bg-green-500/10 flex items-center justify-center">
                  <Check class="h-3.5 w-3.5 text-green-500 stroke-[3]" />
                </div>
                <div v-else-if="log.status === TASK_STATUS.FAILED"
                  class="h-6 w-6 rounded-full bg-red-500/10 flex items-center justify-center">
                  <X class="h-3.5 w-3.5 text-red-500 stroke-[3]" />
                </div>
                <div v-else-if="log.status === TASK_STATUS.RUNNING"
                  class="h-6 w-6 rounded-full bg-yellow-500/10 flex items-center justify-center">
                  <ZapIcon class="h-3.5 w-3.5 text-yellow-500 fill-yellow-500 animate-pulse" />
                </div>
                <div v-else-if="log.status === TASK_STATUS.PENDING"
                  class="h-6 w-6 rounded-full bg-yellow-500/10 flex items-center justify-center">
                  <Clock class="h-3.5 w-3.5 text-yellow-500" />
                </div>
                <div v-else-if="log.status === TASK_STATUS.TIMEOUT"
                  class="h-6 w-6 rounded-full bg-orange-500/10 flex items-center justify-center">
                  <AlertCircle class="h-3.5 w-3.5 text-orange-500" />
                </div>
                <div v-else-if="log.status === TASK_STATUS.CANCELLED"
                  class="h-6 w-6 rounded-full bg-muted flex items-center justify-center">
                  <Ban class="h-3.5 w-3.5 text-muted-foreground" />
                </div>
              </span>
              <span class="w-16 text-right shrink-0 text-muted-foreground text-xs">{{ formatDuration(log.duration)
                }}</span>
              <span v-if="!selectedLog"
                class="w-40 text-right shrink-0 text-muted-foreground text-xs hidden md:block">{{ log.start_time ||
                  log.created_at }}</span>
            </div>
          </div>
        </div>
        <!-- 分页 -->
        <Pagination :total="total" :page="currentPage" @update:page="handlePageChange" />
      </div>

      <!-- 日志详情侧边栏 -->
      <div v-if="selectedLog"
        class="w-full lg:w-[480px] rounded-lg border bg-card flex flex-col overflow-hidden shrink-0 max-h-[60vh] lg:max-h-[calc(100vh-180px)]">
        <div class="flex items-center justify-between px-4 py-3 border-b">
          <div class="flex items-center gap-2">
            <span class="text-sm font-medium">日志详情</span>
            <Button v-if="selectedLog.status === TASK_STATUS.RUNNING" variant="destructive" size="sm"
              class="h-6 px-2 text-[10px]" :disabled="isStopping" @click="stopTask">
              {{ isStopping ? '停止中...' : '停止任务' }}
            </Button>
          </div>
          <Button variant="ghost" size="icon" class="h-7 w-7" @click="closeDetail">
            <X class="h-3.5 w-3.5" />
          </Button>
        </div>
        <div class="px-4 py-3 border-b space-y-2 text-sm">
          <div class="flex justify-between">
            <span class="text-muted-foreground">任务名称</span>
            <span class="font-medium">{{ selectedLog.task_name }}</span>
          </div>
          <div class="flex justify-between items-center">
            <span class="text-muted-foreground">状态</span>
            <Badge variant="outline" :class="[
              'capitalize px-3 py-1 font-semibold rounded-full border shadow-sm transition-all duration-300',
              getStatusBadgeClass(selectedLog.status)
            ]">
              <div class="flex items-center gap-1.5">
                <CheckCircle2 v-if="selectedLog.status === TASK_STATUS.SUCCESS" class="h-3.5 w-3.5" />
                <XCircle v-else-if="selectedLog.status === TASK_STATUS.FAILED" class="h-3.5 w-3.5" />
                <ZapIcon v-else-if="selectedLog.status === TASK_STATUS.RUNNING"
                  class="h-3.5 w-3.5 fill-current animate-pulse text-blue-500" />
                <Clock v-else-if="selectedLog.status === TASK_STATUS.PENDING" class="h-3.5 w-3.5" />
                <AlertCircle v-else-if="selectedLog.status === TASK_STATUS.TIMEOUT" class="h-3.5 w-3.5" />
                <Ban v-else-if="selectedLog.status === TASK_STATUS.CANCELLED" class="h-3.5 w-3.5" />
                <span class="text-xs tracking-wide uppercase">{{ selectedLog.status }}</span>
              </div>
            </Badge>
          </div>
          <div class="flex justify-between">
            <span class="text-muted-foreground">耗时</span>
            <span>{{ formatDuration(selectedLog.duration) }}</span>
          </div>
          <div class="flex justify-between">
            <span class="text-muted-foreground">开始时间</span>
            <span>{{ selectedLog.start_time || '-' }}</span>
          </div>
          <div class="flex justify-between">
            <span class="text-muted-foreground">结束时间</span>
            <span>{{ selectedLog.end_time || '-' }}</span>
          </div>
          <div class="pt-1">
            <span class="text-muted-foreground">命令</span>
            <code class="mt-1 block font-mono bg-muted/40 px-2 py-1 rounded text-xs break-all">
              {{ selectedLog.command }}
            </code>
          </div>
        </div>
        <div class="flex-1 flex flex-col overflow-hidden">
          <div v-if="selectedLog.error" class="px-4 py-3 border-b bg-red-500/5 space-y-2 text-sm">
            <div class="flex items-center gap-2 text-red-500 font-medium">
              <X class="h-4 w-4" />
              <span>系统错误</span>
            </div>
            <code class="block font-mono bg-red-500/10 text-red-600 px-2 py-1 rounded text-xs break-all">
              {{ selectedLog.error }}
            </code>
          </div>
          <div class="px-4 py-2 text-sm text-muted-foreground border-b bg-muted/20 flex items-center justify-between">
            <span>输出</span>
            <Button variant="ghost" size="icon" class="h-6 w-6" @click="showFullscreen = true" title="全屏查看">
              <Maximize2 class="h-3.5 w-3.5" />
            </Button>
          </div>
          <div class="flex-1 overflow-auto">
            <pre class="p-4 text-xs font-mono whitespace-pre-wrap break-all log-pre">{{ decompressedOutput }}</pre>
            <div v-if="isWsLoading" class="p-4 text-sm text-muted-foreground italic">连接中...</div>
          </div>
        </div>
      </div>
    </div>

    <!-- 全屏查看日志 -->
    <LogViewer v-model:open="showFullscreen" :title="`日志输出 - ${selectedLog?.task_name || ''}`"
      :content="decompressedOutput" :status="selectedLog?.status" />
  </div>
</template>
