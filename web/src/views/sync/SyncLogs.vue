<script setup lang="ts">
import { ref, onMounted, watch } from 'vue'
import { useRoute } from 'vue-router'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import Pagination from '@/components/Pagination.vue'
import { Search, Eye, ArrowLeft } from 'lucide-vue-next'
import { api, type SyncLog, type SyncLogDetail } from '@/api'
import { toast } from 'vue-sonner'
import { useSiteSettings } from '@/composables/useSiteSettings'
import TextOverflow from '@/components/TextOverflow.vue'

const route = useRoute()
const { pageSize } = useSiteSettings()

const logs = ref<SyncLog[]>([])
const showDetailDialog = ref(false)
const logDetail = ref<SyncLogDetail | null>(null)

const filterTaskName = ref('')
const syncTaskId = ref<number | undefined>(undefined)
const currentPage = ref(1)
const total = ref(0)
let searchTimer: ReturnType<typeof setTimeout> | null = null

async function loadLogs() {
  try {
    const res = await api.syncLogs.list({
      page: currentPage.value,
      page_size: pageSize.value,
      sync_task_id: syncTaskId.value,
      task_name: filterTaskName.value || undefined
    })
    logs.value = res.data
    total.value = res.total
  } catch { toast.error('加载日志失败') }
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

async function viewDetail(id: number) {
  try {
    logDetail.value = await api.syncLogs.detail(id)
    showDetailDialog.value = true
  } catch { toast.error('加载详情失败') }
}

function formatDuration(ms: number): string {
  if (ms < 1000) return `${ms}ms`
  if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`
  return `${(ms / 60000).toFixed(1)}m`
}

function clearFilter() {
  syncTaskId.value = undefined
  filterTaskName.value = ''
  currentPage.value = 1
  loadLogs()
}

watch(() => route.query.sync_task_id, (val) => {
  if (val) {
    syncTaskId.value = Number(val)
  } else {
    syncTaskId.value = undefined
  }
  loadLogs()
}, { immediate: true })

onMounted(() => {
  if (route.query.sync_task_id) {
    syncTaskId.value = Number(route.query.sync_task_id)
  }
  loadLogs()
})
</script>

<template>
  <div class="space-y-6">
    <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
      <div>
        <h2 class="text-xl sm:text-2xl font-bold tracking-tight">同步日志</h2>
        <p class="text-muted-foreground text-sm">查看同步任务执行历史</p>
      </div>
      <div class="flex items-center gap-2">
        <div class="relative flex-1 sm:flex-none">
          <Search class="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
          <Input v-model="filterTaskName" placeholder="搜索任务名..." class="h-9 pl-9 w-full sm:w-56 text-sm" @input="handleSearch" />
        </div>
        <Button v-if="syncTaskId" variant="outline" size="sm" @click="clearFilter">
          <ArrowLeft class="h-4 w-4 mr-1" /> 返回全部
        </Button>
      </div>
    </div>

    <div class="rounded-lg border bg-card overflow-x-auto">
      <div class="flex items-center gap-4 px-4 py-2 border-b bg-muted/50 text-sm text-muted-foreground font-medium min-w-[600px]">
        <span class="w-12 shrink-0">ID</span>
        <span class="w-28 shrink-0">任务名称</span>
        <span class="flex-1 shrink-0">源地址</span>
        <span class="w-20 shrink-0 text-center">状态</span>
        <span class="w-20 shrink-0 text-center">耗时</span>
        <span class="w-40 shrink-0 hidden sm:block">执行时间</span>
        <span class="w-16 shrink-0 text-center">操作</span>
      </div>
      <div class="divide-y min-w-[600px]">
        <div v-if="logs.length === 0" class="text-sm text-muted-foreground text-center py-8">暂无日志</div>
        <div v-for="log in logs" :key="log.id" class="flex items-center gap-4 px-4 py-2 hover:bg-muted/50 transition-colors">
          <span class="w-12 shrink-0 text-muted-foreground text-sm">#{{ log.id }}</span>
          <span class="w-28 font-medium truncate shrink-0 text-sm"><TextOverflow :text="log.sync_task_name" title="任务名称" /></span>
          <code class="flex-1 shrink-0 text-muted-foreground truncate text-xs bg-muted px-2 py-1 rounded"><TextOverflow :text="log.source_url" title="源地址" /></code>
          <span class="w-20 flex justify-center shrink-0">
            <Badge :variant="log.status === 'success' ? 'default' : 'destructive'" class="text-xs">{{ log.status === 'success' ? '成功' : '失败' }}</Badge>
          </span>
          <span class="w-20 shrink-0 text-center text-muted-foreground text-xs">{{ formatDuration(log.duration) }}</span>
          <span class="w-40 shrink-0 text-muted-foreground text-xs hidden sm:block">{{ log.created_at }}</span>
          <span class="w-16 shrink-0 flex justify-center">
            <Button variant="ghost" size="icon" class="h-7 w-7" @click="viewDetail(log.id)" title="查看详情"><Eye class="h-3.5 w-3.5" /></Button>
          </span>
        </div>
      </div>
      <Pagination :total="total" :page="currentPage" @update:page="handlePageChange" />
    </div>

    <Dialog v-model:open="showDetailDialog">
      <DialogContent class="sm:max-w-[700px] max-h-[80vh]">
        <DialogHeader>
          <DialogTitle>同步日志详情</DialogTitle>
        </DialogHeader>
        <div v-if="logDetail" class="space-y-4">
          <div class="grid grid-cols-2 gap-4 text-sm">
            <div><span class="text-muted-foreground">任务名称:</span> {{ logDetail.sync_task_name }}</div>
            <div><span class="text-muted-foreground">状态:</span> <Badge :variant="logDetail.status === 'success' ? 'default' : 'destructive'">{{ logDetail.status === 'success' ? '成功' : '失败' }}</Badge></div>
            <div><span class="text-muted-foreground">耗时:</span> {{ formatDuration(logDetail.duration) }}</div>
            <div><span class="text-muted-foreground">执行时间:</span> {{ logDetail.created_at }}</div>
          </div>
          <div class="text-sm"><span class="text-muted-foreground">源地址:</span> <code class="text-xs bg-muted px-2 py-1 rounded break-all">{{ logDetail.source_url }}</code></div>
          <div class="text-sm"><span class="text-muted-foreground">目标路径:</span> <code class="text-xs bg-muted px-2 py-1 rounded">{{ logDetail.target_path }}</code></div>
          <div>
            <span class="text-sm text-muted-foreground">输出日志:</span>
            <pre class="mt-2 p-3 bg-muted rounded-lg text-xs overflow-auto max-h-[300px] whitespace-pre-wrap">{{ logDetail.output || '无输出' }}</pre>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  </div>
</template>
