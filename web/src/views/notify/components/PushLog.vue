<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { api, type AppLog, LOG_CATEGORY, LOG_STATUS } from '@/api'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import Pagination from '@/components/Pagination.vue'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { toast } from 'vue-sonner'
import { format } from 'date-fns'
import {
  RefreshCw, Trash2, Check, X, Search
} from 'lucide-vue-next'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog'
import { useSiteSettings } from '@/composables/useSiteSettings'


const { pageSize } = useSiteSettings()

const logs = ref<AppLog[]>([])
const selectedLogId = ref<string | null>(null)
const total = ref(0)
const loading = ref(false)
const showClearConfirm = ref(false)

const filters = ref({
  status: 'all',
  keyword: '',
  page: 1
})

let searchTimer: ReturnType<typeof setTimeout> | null = null


const detailDialogProps = ref({
  open: false,
  title: '',
  content: '',
  error: ''
})

async function fetchLogs() {
  loading.value = true
  try {
    const res = await api.appLogs.list({
      category: LOG_CATEGORY.PUSH_LOG,
      status: filters.value.status === 'all' ? undefined : filters.value.status,
      keyword: filters.value.keyword || undefined,
      page: filters.value.page,
      page_size: pageSize.value
    })
    logs.value = res.data || []
    total.value = res.total || 0
  } catch (e: any) {
    toast.error(e.message || '获取推送日志失败')
  } finally {
    loading.value = false
  }
}

function handleSearch() {
  if (searchTimer) clearTimeout(searchTimer)
  searchTimer = setTimeout(() => {
    filters.value.page = 1
    fetchLogs()
  }, 300)
}

function handlePageChange(page: number) {
  filters.value.page = page
  fetchLogs()
}

function handleStatusChange(val: any) {
  if (val === null || val === undefined) return
  filters.value.status = String(val)
  filters.value.page = 1
  fetchLogs()
}

function showDetail(log: AppLog) {
  selectedLogId.value = log.id
  detailDialogProps.value = {
    open: true,
    title: log.title,
    content: log.content,
    error: log.error_msg
  }
}

async function handleClear() {
  try {
    await api.appLogs.clear(LOG_CATEGORY.PUSH_LOG)
    toast.success('清空成功')
    filters.value.page = 1
    fetchLogs()
  } catch (e: any) {
    toast.error('清空失败: ' + (e.message || ''))
  }
}

onMounted(() => {
  fetchLogs()
})

const selectedLog = computed(() => logs.value.find((l: AppLog) => l.id === selectedLogId.value))

function getStatusBadgeClass(status: string) {
  switch (status) {
    case LOG_STATUS.SUCCESS:
      return 'bg-green-500/10 text-green-700 border-green-200/50 dark:bg-green-500/20 dark:text-green-400 dark:border-green-900/50'
    case LOG_STATUS.FAILED:
      return 'bg-red-500/10 text-red-700 border-red-200/50 dark:bg-red-500/20 dark:text-red-400 dark:border-red-900/50'
    default:
      return 'bg-secondary text-secondary-foreground border-transparent'
  }
}

function getLogIndex(index: number) {
  return total.value - (filters.value.page - 1) * pageSize.value - index
}

function formatDate(dateStr: string) {
  if (!dateStr) return '-'
  try {
    return format(new Date(dateStr), 'yyyy-MM-dd HH:mm:ss')
  } catch {
    return dateStr
  }
}

function onDialogClose(open: boolean) {
  if (!open) {
    selectedLogId.value = null
  }
}
</script>

<template>
  <div class="space-y-6">
    <div class="flex items-center justify-between gap-2">
      <div class="flex items-center gap-2">
        <div class="relative flex-1 sm:flex-none">
          <Search class="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
          <Input v-model="filters.keyword" placeholder="搜索标题或内容..." class="h-9 pl-9 w-full sm:w-56 text-sm"
            @input="handleSearch" />
        </div>
        <div class="relative flex-1 sm:flex-none">
          <Select :model-value="filters.status" @update:model-value="handleStatusChange">
            <SelectTrigger class="h-9 w-full sm:w-28 text-sm">
              <SelectValue placeholder="状态" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">所有状态</SelectItem>
              <SelectItem :value="LOG_STATUS.SUCCESS">发送成功</SelectItem>
              <SelectItem :value="LOG_STATUS.FAILED">发送失败</SelectItem>
            </SelectContent>
          </Select>
        </div>
        <Button variant="outline" size="icon" class="h-9 w-9 shrink-0" @click="fetchLogs" :disabled="loading"
          title="刷新">
          <RefreshCw class="h-4 w-4" :class="{ 'animate-spin': loading }" />
        </Button>
      </div>
      <AlertDialog :open="showClearConfirm" @update:open="showClearConfirm = $event">
        <Button variant="outline"
          class="h-9 px-4 shrink-0 text-sm text-destructive hover:bg-destructive/10 hover:text-destructive border-destructive/20"
          @click="showClearConfirm = true">
          <Trash2 class="h-4 w-4 sm:mr-2" /> <span class="hidden sm:inline" style="padding-left: 2px;">清空记录</span>
        </Button>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>确认清空所有推送日志？</AlertDialogTitle>
            <AlertDialogDescription>
              此操作将永久清空当前分类下的所有消息推送历史记录，操作后无法恢复。确认要继续吗？
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>取消</AlertDialogCancel>
            <AlertDialogAction @click="handleClear" variant="destructive">
              确认清空
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>

    <div class="rounded-lg border bg-card overflow-x-auto">
      <!-- 表头 -->
      <div
        class="flex items-center gap-2 sm:gap-4 px-3 sm:px-4 py-2 border-b bg-muted/50 text-xs sm:text-sm text-muted-foreground font-medium sm:min-w-[700px]">
        <span class="w-12 sm:w-16 shrink-0">序号</span>
        <span class="w-40 sm:w-56 shrink-0">标题及渠道</span>
        <span class="hidden sm:flex sm:flex-1">内容详情</span>
        <span class="w-10 sm:w-16 shrink-0 text-center">状态</span>
        <span class="shrink-0 w-24 sm:w-40 sm:text-right">发送时间</span>
      </div>

      <!-- 列表 -->
      <div class="divide-y sm:min-w-[700px]">
        <div v-if="logs.length === 0 && !loading" class="text-sm text-muted-foreground text-center py-8">
          暂无推送记录
        </div>
        <div v-for="(log, index) in logs" :key="log.id"
          class="flex items-center gap-2 sm:gap-4 px-3 sm:px-4 py-2 hover:bg-muted/50 transition-colors cursor-pointer group"
          :class="[selectedLogId === log.id && 'bg-accent/50']" @click="showDetail(log)">
          <span class="w-12 sm:w-16 shrink-0 text-muted-foreground text-xs sm:text-sm">#{{ getLogIndex(index) }}</span>
          <span class="w-40 sm:w-56 shrink-0 font-medium text-xs sm:text-sm truncate" :title="log.title">
            <span v-if="log.channel_name" class="mr-1 text-muted-foreground">[{{ log.channel_name }}]</span>{{ log.title
            }}
          </span>
          <span class="hidden sm:flex sm:flex-1 text-xs sm:text-sm text-muted-foreground truncate" :title="log.content">
            {{ log.content || '-' }}
          </span>
          <span class="w-10 sm:w-16 shrink-0 flex justify-center">
            <span
              :class="['h-2 w-2 rounded-full', log.status === LOG_STATUS.SUCCESS ? 'bg-green-500' : 'bg-red-500']"></span>
          </span>
          <span class="shrink-0 w-24 sm:w-40 sm:text-right text-xs text-muted-foreground">
            {{ formatDate(log.created_at) }}
          </span>
        </div>
      </div>

      <!-- 分页 -->
      <Pagination :total="total" :page="filters.page" @update:page="handlePageChange" />
    </div>

    <Dialog v-model:open="detailDialogProps.open" @update:open="onDialogClose">
      <DialogContent class="sm:max-w-2xl max-h-[90vh] flex flex-col p-0 overflow-hidden">
        <DialogHeader class="px-6 py-4 border-b bg-muted/20">
          <div class="flex items-center justify-between pr-8">
            <DialogTitle>日志详情</DialogTitle>
            <Badge variant="outline" :class="[
              'px-2 py-0.5 text-[10px] font-bold rounded-full border shadow-sm transition-all duration-300',
              selectedLog ? getStatusBadgeClass(selectedLog.status) : ''
            ]">
              <div class="flex items-center gap-1.5 uppercase tracking-wider">
                <Check v-if="selectedLog?.status === LOG_STATUS.SUCCESS" class="h-3 w-3" />
                <X v-else class="h-3 w-3" />
                <span>{{ selectedLog?.status === LOG_STATUS.SUCCESS ? 'Success' : 'Failed' }}</span>
              </div>
            </Badge>
          </div>
        </DialogHeader>

        <div class="flex-1 overflow-y-auto">
          <!-- 基础信息区 -->
          <div class="px-6 py-4 border-b space-y-3 bg-card">
            <div class="flex justify-between items-center text-sm">
              <span class="text-muted-foreground">标题</span>
              <span class="font-medium text-foreground">{{ detailDialogProps.title }}</span>
            </div>
            <div v-if="selectedLog?.channel_name" class="flex justify-between items-center text-sm">
              <span class="text-muted-foreground">发送渠道</span>
              <span class="font-medium text-foreground">{{ selectedLog.channel_name }}</span>
            </div>
            <div class="flex justify-between items-center text-sm">
              <span class="text-muted-foreground">发生时间</span>
              <span class="font-mono text-xs text-muted-foreground">{{ selectedLog ? formatDate(selectedLog.created_at)
                : '-' }}</span>
            </div>
          </div>

          <!-- 内容输出区 -->
          <div class="flex flex-col min-h-0 bg-muted/5">
            <div
              class="px-6 py-2.5 text-xs font-semibold text-muted-foreground border-b bg-muted/10 uppercase tracking-wider">
              推送内容
            </div>
            <div class="p-6">
              <div v-if="detailDialogProps.content"
                class="text-sm text-foreground bg-muted/20 p-5 rounded-xl border border-border/50 whitespace-pre-wrap break-all leading-relaxed shadow-sm">
                {{ detailDialogProps.content }}
              </div>
              <div v-else class="text-sm text-muted-foreground italic py-2">无推送内容</div>
            </div>

            <template v-if="detailDialogProps.error">
              <div
                class="px-6 py-2.5 text-xs font-semibold text-red-500 border-y bg-red-500/5 uppercase tracking-wider">
                错误信息
              </div>
              <div class="p-6">
                <div v-if="detailDialogProps.error"
                  class="text-sm text-red-700 bg-red-500/10 p-5 rounded-xl border border-red-200/50 whitespace-pre-wrap break-all leading-relaxed shadow-sm">
                  {{ detailDialogProps.error }}
                </div>
                <div v-else class="text-sm text-muted-foreground italic py-2">无错误信息</div>
              </div>
            </template>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  </div>
</template>
