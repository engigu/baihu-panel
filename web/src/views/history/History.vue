<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import Pagination from '@/components/Pagination.vue'
import LogViewer from './LogViewer.vue'
import { RefreshCw, X, Search, Maximize2 } from 'lucide-vue-next'
import { api, type TaskLog, type LogDetail } from '@/api'
import { toast } from 'vue-sonner'
import pako from 'pako'
import { useSiteSettings } from '@/composables/useSiteSettings'

const { pageSize } = useSiteSettings()

const logs = ref<TaskLog[]>([])
const selectedLog = ref<TaskLog | null>(null)
const logDetail = ref<LogDetail | null>(null)
const filterKeyword = ref('')
const currentPage = ref(1)
const total = ref(0)
let searchTimer: ReturnType<typeof setTimeout> | null = null

// 全屏查看
const showFullscreen = ref(false)

function decompressOutput(compressed: string): string {
  if (!compressed) return '无输出'
  try {
    const binaryString = atob(compressed)
    const bytes = new Uint8Array(binaryString.length)
    for (let i = 0; i < binaryString.length; i++) {
      bytes[i] = binaryString.charCodeAt(i)
    }
    const decompressed = pako.inflate(bytes)
    return new TextDecoder().decode(decompressed)
  } catch {
    return compressed
  }
}

const decompressedOutput = computed(() => {
  if (!logDetail.value?.output) return '无输出'
  return decompressOutput(logDetail.value.output)
})

async function loadLogs() {
  try {
    const params: { page: number; page_size: number; task_name?: string } = {
      page: currentPage.value,
      page_size: pageSize.value
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
  selectedLog.value = log
  logDetail.value = null
  try {
    logDetail.value = await api.logs.detail(log.id)
  } catch {
    toast.error('加载日志详情失败')
  }
}

function closeDetail() {
  selectedLog.value = null
  logDetail.value = null
}

function formatDuration(ms: number): string {
  if (ms < 1000) return `${ms}ms`
  if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`
  return `${(ms / 60000).toFixed(1)}m`
}

onMounted(loadLogs)
</script>

<template>
  <div class="space-y-6">
    <div class="flex items-center justify-between">
      <div>
        <h2 class="text-2xl font-bold tracking-tight">执行历史</h2>
        <p class="text-muted-foreground">查看任务执行记录和日志</p>
      </div>
      <div class="flex items-center gap-2">
        <div class="relative">
          <Search class="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
          <Input v-model="filterKeyword" placeholder="搜索任务..." class="h-9 pl-9 w-56 text-sm" @input="handleSearch" />
        </div>
        <Button variant="outline" size="icon" class="h-9 w-9" @click="loadLogs">
          <RefreshCw class="h-4 w-4" />
        </Button>
      </div>
    </div>

    <div class="flex gap-4">
      <!-- 日志列表 -->
      <div class="flex-1 min-w-0 rounded-lg border bg-card">
        <!-- 表头 -->
        <div class="flex items-center gap-4 px-4 py-2 border-b bg-muted/50 text-sm text-muted-foreground font-medium">
          <span class="w-12 shrink-0">ID</span>
          <span class="w-32 shrink-0">任务名称</span>
          <span :class="selectedLog ? 'w-40 shrink-0' : 'flex-1'">命令</span>
          <span class="w-12 shrink-0 text-center">状态</span>
          <span class="w-20 text-right shrink-0">耗时</span>
          <span class="w-40 text-right shrink-0">执行时间</span>
        </div>
        <!-- 列表 -->
        <div class="divide-y">
          <div v-if="logs.length === 0" class="text-sm text-muted-foreground text-center py-8">
            暂无日志
          </div>
          <div
            v-for="log in logs"
            :key="log.id"
            :class="[
              'flex items-center gap-4 px-4 py-2 min-h-[44px] cursor-pointer hover:bg-muted/50 transition-colors',
              selectedLog?.id === log.id && 'bg-accent'
            ]"
            @click="selectLog(log)"
          >
            <span class="w-12 shrink-0 text-muted-foreground text-sm">#{{ log.id }}</span>
            <span class="w-32 font-medium truncate shrink-0 text-sm">{{ log.task_name }}</span>
            <code :class="['text-muted-foreground truncate text-xs bg-muted px-2 py-1 rounded', selectedLog ? 'w-40 shrink-0' : 'flex-1']">{{ log.command }}</code>
            <span class="w-12 flex justify-center shrink-0">
              <span :class="['w-2 h-2 rounded-full', log.status === 'success' ? 'bg-green-500' : log.status === 'failed' ? 'bg-red-500' : 'bg-yellow-500']" />
            </span>
            <span class="w-20 text-right shrink-0 text-muted-foreground text-xs">{{ formatDuration(log.duration) }}</span>
            <span class="w-40 text-right shrink-0 text-muted-foreground text-xs">{{ log.created_at }}</span>
          </div>
        </div>
        <!-- 分页 -->
        <Pagination :total="total" :page="currentPage" @update:page="handlePageChange" />
      </div>

      <!-- 日志详情侧边栏 -->
      <div
        v-if="selectedLog"
        class="w-[480px] rounded-lg border bg-card flex flex-col overflow-hidden shrink-0 max-h-[calc(100vh-180px)]"
      >
        <div class="flex items-center justify-between px-4 py-3 border-b">
          <span class="text-sm font-medium">日志详情</span>
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
            <span class="flex items-center gap-1.5">
              <span :class="['w-2 h-2 rounded-full', selectedLog.status === 'success' ? 'bg-green-500' : selectedLog.status === 'failed' ? 'bg-red-500' : 'bg-yellow-500']" />
              {{ selectedLog.status }}
            </span>
          </div>
          <div class="flex justify-between">
            <span class="text-muted-foreground">耗时</span>
            <span>{{ formatDuration(selectedLog.duration) }}</span>
          </div>
          <div class="flex justify-between">
            <span class="text-muted-foreground">执行时间</span>
            <span>{{ selectedLog.created_at }}</span>
          </div>
          <div class="pt-1">
            <span class="text-muted-foreground">命令</span>
            <code class="mt-1 block font-mono bg-muted px-2 py-1 rounded text-xs break-all">
              {{ selectedLog.command }}
            </code>
          </div>
        </div>
        <div class="flex-1 flex flex-col overflow-hidden">
          <div class="px-4 py-2 text-sm text-muted-foreground border-b bg-muted/50 flex items-center justify-between">
            <span>输出</span>
            <Button variant="ghost" size="icon" class="h-6 w-6" @click="showFullscreen = true" title="全屏查看">
              <Maximize2 class="h-3.5 w-3.5" />
            </Button>
          </div>
          <div class="flex-1 overflow-auto">
            <pre v-if="logDetail" class="p-4 text-xs font-mono whitespace-pre-wrap break-all">{{ decompressedOutput }}</pre>
            <div v-else class="p-4 text-sm text-muted-foreground">加载中...</div>
          </div>
        </div>
      </div>
    </div>

    <!-- 全屏查看日志 -->
    <LogViewer
      v-model:open="showFullscreen"
      :title="`日志输出 - ${selectedLog?.task_name || ''}`"
      :content="decompressedOutput"
    />
  </div>
</template>
