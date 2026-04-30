<script setup lang="ts">
import { ref, onMounted, computed, watch, nextTick } from 'vue'
import { useRoute } from 'vue-router'
import { TASK_STATUS, TASK_TYPE, TASK_STATUS_TEXT } from '@/constants'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import Pagination from '@/components/Pagination.vue'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import LogViewer from './LogViewer.vue'
import {
  RefreshCw, X, Search, GitBranch, Terminal,
  AlertCircle, Ban, Clock, Zap as ZapIcon, Check, Trash2
} from 'lucide-vue-next'
import { api, type TaskLog } from '@/api'
import LogDetailCard from '@/components/LogDetailCard.vue'
import BaihuDialog from '@/components/ui/BaihuDialog.vue'
import { toast } from 'vue-sonner'
import { useSiteSettings } from '@/composables/useSiteSettings'
import TextOverflow from '@/components/TextOverflow.vue'

const route = useRoute()
const { pageSize } = useSiteSettings()

const logs = ref<TaskLog[]>([])
const selectedLog = ref<TaskLog | null>(null)
const filterKeyword = ref('')
const filterTaskId = ref<string | undefined>(undefined)
const filterStatus = ref<string | undefined>(undefined)
const currentPage = ref(1)
const total = ref(0)

let searchTimer: ReturnType<typeof setTimeout> | null = null
let durationTimer: ReturnType<typeof setInterval> | null = null

const isRefreshing = ref(false)

// 全屏查看
const showFullscreen = ref(false)

// 清除所有日志弹窗
const showClearDialog = ref(false)

// 删除单条日志弹窗
const showDeleteDialog = ref(false)
const deleteLogId = ref<string | null>(null)

const wsContent = ref('')
const isWsLoading = ref(false)
let logSocket: WebSocket | null = null


import { decompressFromBase64 } from '@/utils/decompress'

const decompressedOutput = computed(() => {
  if (!wsContent.value) return ''
  const decoded = decompressFromBase64(wsContent.value)
  return decoded
})

async function loadLogs() {
  isRefreshing.value = true
  try {
    const params: { page: number; page_size: number; task_id?: string; task_name?: string; status?: string } = {
      page: currentPage.value,
      page_size: pageSize.value
    }
    if (filterTaskId.value) {
      params.task_id = filterTaskId.value
    }
    if (filterKeyword.value.trim()) {
      params.task_name = filterKeyword.value.trim()
    }
    if (filterStatus.value && filterStatus.value !== 'all') {
      params.status = filterStatus.value
    }
    const response = await api.logs.list(params)
    logs.value = response.data
    total.value = response.total
  } catch {
    toast.error('加载日志失败')
  } finally {
    isRefreshing.value = false
  }
}

function handleSearch() {
  if (searchTimer) clearTimeout(searchTimer)
  searchTimer = setTimeout(() => {
    currentPage.value = 1
    loadLogs()
  }, 300)
}

function handleStatusChange() {
  currentPage.value = 1
  loadLogs()
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

  if (log.status !== TASK_STATUS.RUNNING) {
    try {
      const res = await api.logs.get(log.id)
      wsContent.value = res.output
    } catch {
      toast.error('加载详情失败')
    } finally {
      isWsLoading.value = false
    }
    return
  }

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
    wsContent.value += event.data
    // 自动滚动到底部
    nextTick(() => {
      const pre = document.querySelector('.log-pre')
      if (pre) pre.scrollTop = pre.scrollHeight
    })
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

async function handleClearLogs() {
  try {
    await api.logs.clear(filterTaskId.value)
    toast.success('相关日志已清空')
    showClearDialog.value = false
    loadLogs()
  } catch (error: any) {
    toast.error(error.message || '清空失败')
  }
}

function confirmDeleteLog(id: string) {
  deleteLogId.value = id
  showDeleteDialog.value = true
}

async function handleDeleteLog() {
  if (!deleteLogId.value) return
  try {
    await api.logs.delete(deleteLogId.value)
    toast.success('该日志已删除')

    // 如果当前选中的是这条日志，关闭详情页
    if (selectedLog.value?.id === deleteLogId.value) {
      closeDetail()
    }

    showDeleteDialog.value = false
    loadLogs()
  } catch (err: any) {
    toast.error(err.message || '删除失败')
  }
}



function getTaskTypeTitle(type: string) {
  return type === TASK_TYPE.REPO ? '仓库同步' : '普通任务'
}

onMounted(() => {
  // 从 URL 读取参数
  const taskIdParam = route.query.task_id
  if (taskIdParam) {
    filterTaskId.value = String(taskIdParam)
  }
  const statusParam = route.query.status
  if (statusParam) {
    filterStatus.value = String(statusParam)
  }
  loadLogs()
})

// 监听路由变化
watch(() => route.query, (newQuery) => {
  filterTaskId.value = newQuery.task_id ? String(newQuery.task_id) : undefined
  filterStatus.value = newQuery.status ? String(newQuery.status) : undefined
  currentPage.value = 1
  loadLogs()
}, { deep: true })
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
          <Input v-model="filterKeyword" placeholder="搜索任务..." class="h-9 pl-9 w-full sm:w-40 md:w-56 text-sm"
            @input="handleSearch" />
        </div>
        <div class="relative flex-1 sm:flex-none">
          <Select v-model="filterStatus" @update:model-value="handleStatusChange">
            <SelectTrigger class="h-9 w-full sm:w-28 text-sm">
              <SelectValue placeholder="状态" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">所有状态</SelectItem>
              <SelectItem :value="TASK_STATUS.RUNNING">{{ TASK_STATUS_TEXT[TASK_STATUS.RUNNING] }}</SelectItem>
              <SelectItem :value="TASK_STATUS.SUCCESS">{{ TASK_STATUS_TEXT[TASK_STATUS.SUCCESS] }}</SelectItem>
              <SelectItem :value="TASK_STATUS.FAILED">{{ TASK_STATUS_TEXT[TASK_STATUS.FAILED] }}</SelectItem>
              <SelectItem :value="TASK_STATUS.TIMEOUT">{{ TASK_STATUS_TEXT[TASK_STATUS.TIMEOUT] }}</SelectItem>
              <SelectItem :value="TASK_STATUS.CANCELLED">{{ TASK_STATUS_TEXT[TASK_STATUS.CANCELLED] }}</SelectItem>
            </SelectContent>
          </Select>
        </div>
        <Button variant="outline" size="icon" class="h-9 w-9 shrink-0" @click="loadLogs" title="刷新" :disabled="isRefreshing">
          <RefreshCw class="h-4 w-4" :class="{ 'animate-spin': isRefreshing }" />
        </Button>
        <Button variant="outline"
          class="h-9 px-4 shrink-0 text-sm text-destructive hover:bg-destructive/10 hover:text-destructive border-destructive/20"
          @click="showClearDialog = true">
          <Trash2 class="h-4 w-4 sm:mr-2" /> <span class="hidden sm:inline" style="padding-left: 2px;">清空日志</span>
        </Button>
      </div>
    </div>

    <div class="flex flex-col lg:flex-row gap-4" style="height: 520px;">
      <!-- 日志列表 -->
      <div class="flex-1 min-w-0 rounded-lg border bg-card overflow-hidden flex flex-col">
        <!-- 小屏表头 -->
        <div
          class="flex sm:hidden items-center gap-2 px-3 py-2 border-b bg-muted/20 text-xs text-muted-foreground font-medium">
          <span class="w-14 shrink-0">序号</span>
          <span class="w-8 shrink-0 text-center">类型</span>
          <span class="flex-1 min-w-0">任务名称</span>
          <span class="w-8 shrink-0 text-center">状态</span>
          <span class="w-16 text-right shrink-0">耗时</span>
          <span class="w-8 text-center shrink-0"></span>
        </div>
        <!-- 大屏表头 -->
        <div
          class="hidden sm:flex items-center gap-4 px-4 h-11 border-b bg-muted/20 text-sm text-muted-foreground font-medium">
          <span class="w-16 shrink-0">序号</span>
          <span class="w-12 shrink-0 text-center">类型</span>
          <span class="w-36 shrink-0">任务名称</span>
          <span class="flex-1 min-w-0">命令</span>
          <span class="w-12 shrink-0 text-center">状态</span>
          <span class="w-16 text-right shrink-0">耗时</span>
          <span v-if="!selectedLog" class="w-40 text-right shrink-0 hidden md:block">执行时间</span>
          <span class="w-10 shrink-0 text-center"></span>
        </div>
        <!-- 列表 -->
        <div class="divide-y flex-1">
          <div v-if="logs.length === 0" class="text-sm text-muted-foreground text-center py-8">
            暂无日志
          </div>
          <div v-for="(log, index) in logs" :key="log.id" :class="[
            'cursor-pointer hover:bg-muted/30 transition-colors group',
            selectedLog?.id === log.id && 'bg-accent/50'
          ]" @click="selectLog(log)">
            <!-- 小屏行 -->
            <div class="flex sm:hidden items-center gap-2 px-3 py-2">
              <span class="w-14 shrink-0 text-muted-foreground text-xs">#{{ total - (currentPage - 1) * pageSize - index
                }}</span>
              <span class="w-8 shrink-0 flex justify-center" :title="getTaskTypeTitle(log.task_type || 'task')">
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
              <span class="w-16 text-right shrink-0 text-muted-foreground text-xs whitespace-nowrap">{{ formatDuration(log.duration)
                }}</span>
              <span class="w-8 shrink-0 flex justify-center opacity-100">
                <Button variant="ghost" size="icon"
                  class="h-6 w-6 text-muted-foreground hover:text-destructive shrink-0"
                  @click.stop="confirmDeleteLog(log.id)" title="删除该日志">
                  <Trash2 class="h-3.5 w-3.5" />
                </Button>
              </span>
            </div>
            <!-- 大屏行 -->
            <div class="hidden sm:flex items-center gap-4 px-4 py-2">
              <span class="w-16 shrink-0 text-muted-foreground text-sm">#{{ total - (currentPage - 1) * pageSize - index
                }}</span>
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
              <span class="w-10 shrink-0 flex justify-center opacity-100">
                <Button variant="ghost" size="icon"
                  class="h-6 w-6 text-muted-foreground hover:text-destructive shrink-0"
                  @click.stop="confirmDeleteLog(log.id)" title="删除该日志">
                  <Trash2 class="h-3.5 w-3.5" />
                </Button>
              </span>
            </div>
          </div>
        </div>
        <!-- 分页 -->
        <Pagination :total="total" :page="currentPage" @update:page="handlePageChange" />
      </div>

      <!-- 日志详情侧边栏 -->
      <div v-if="selectedLog"
        class="w-full lg:w-[480px] rounded-lg border bg-card flex flex-col overflow-hidden shrink-0">
        <LogDetailCard 
          :log="selectedLog" 
          :content="decompressedOutput" 
          :loading="isWsLoading" 
          :is-stopping="isStopping"
          @close="closeDetail"
          @stop="stopTask"
          @delete="confirmDeleteLog"
          @maximize="showFullscreen = true"
        />
      </div>
    </div>

    <!-- 全屏查看日志 -->
    <LogViewer v-model:open="showFullscreen"
      :log="selectedLog"
      :is-stopping="isStopping"
      :content="decompressedOutput"
      @stop="stopTask" />

    <!-- 清空日志确认弹窗 -->
    <BaihuDialog v-model:open="showClearDialog" title="确认清空日志?">
      <div class="text-sm text-muted-foreground leading-relaxed">
        此操作将永久删除{{ filterTaskId ? '当前任务的' : '所有' }}任务历史记录，包括控制台输出，并且无法撤销。
        <p class="mt-2 text-destructive font-medium">⚠️ 风险：该操作不可撤销。</p>
      </div>
      <template #footer>
        <Button variant="ghost" @click="showClearDialog = false">取消</Button>
        <Button variant="destructive" class="shadow-lg shadow-destructive/20" @click="handleClearLogs">立即清空</Button>
      </template>
    </BaihuDialog>

    <!-- 单条删除确认弹窗 -->
    <BaihuDialog v-model:open="showDeleteDialog" title="确认删除这条日志?">
      <div class="text-sm text-muted-foreground leading-relaxed">
        此操作将永久删除该次运行记录和日志文件，且不可恢复。
      </div>
      <template #footer>
        <Button variant="ghost" @click="showDeleteDialog = false">取消</Button>
        <Button variant="destructive" class="shadow-lg shadow-destructive/20" @click="handleDeleteLog">确认删除</Button>
      </template>
    </BaihuDialog>
  </div>
</template>

<style scoped>
:deep(.log-pre code) {
  display: block;
  padding: 0 !important;
  margin: 0 !important;
  background: transparent !important;
}

:deep(.log-pre span) {
  vertical-align: top;
}
</style>
