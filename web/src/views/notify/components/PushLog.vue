<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { api, type AppLog, LOG_CATEGORY, LOG_STATUS } from '@/api'
import { Badge } from '@/components/ui/badge'
import Pagination from '@/components/Pagination.vue'
import { toast } from 'vue-sonner'
import { format } from 'date-fns'
import {
  Check, X
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
import { useEventBus } from '@/composables/useEventBus'
import { LOG_EVENTS } from '@/constants'
import BaihuDialog from '@/components/ui/BaihuDialog.vue'

const props = defineProps<{
  filters: {
    status: string
    keyword: string
  }
}>()

const { pageSize } = useSiteSettings()

const logs = ref<AppLog[]>([])
const selectedLogId = ref<string | null>(null)
const total = ref(0)
const loading = ref(false)
const showClearConfirm = ref(false)
const currentPage = ref(1)

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
      status: props.filters.status === 'all' ? undefined : props.filters.status,
      keyword: props.filters.keyword || undefined,
      page: currentPage.value,
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

function handlePageChange(index: number) {
  currentPage.value = index
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
    currentPage.value = 1
    fetchLogs()
  } catch (e: any) {
    toast.error('清空失败: ' + (e.message || ''))
  }
  showClearConfirm.value = false
}

onMounted(() => {
  fetchLogs()
})

// 实时更新：当有新推送产生且用户在第一页时刷新
useEventBus([LOG_EVENTS.ADDED], (payload) => {
  if (payload && payload.category === LOG_CATEGORY.PUSH_LOG) {
    if (currentPage.value === 1) {
      fetchLogs()
    }
  }
})

const selectedLog = computed(() => logs.value.find((l: AppLog) => l.id === selectedLogId.value))

defineExpose({
  fetchLogs,
  showClearConfirm
})

function getStatusBadgeClass(status: string) {
  switch (status) {
    case LOG_STATUS.SUCCESS:
      return 'bg-green-500/15 text-green-500 border-green-500/30'
    case LOG_STATUS.FAILED:
      return 'bg-red-500/15 text-red-500 border-red-500/30'
    default:
      return 'bg-secondary text-secondary-foreground border-transparent'
  }
}

function getLogIndex(index: number) {
  return total.value - (currentPage.value - 1) * pageSize.value - index
}

function formatDate(dateStr: string) {
  if (!dateStr) return '-'
  try {
    return format(new Date(dateStr), 'yyyy-MM-dd HH:mm:ss')
  } catch {
    return dateStr
  }
}

import { ansiToHtml } from '@/utils/ansi'

const renderedContent = computed(() => {
  return ansiToHtml(detailDialogProps.value.content)
})

function onDialogClose(open: boolean) {
  if (!open) {
    selectedLogId.value = null
  }
}
</script>

<template>
  <div class="space-y-6">
    <AlertDialog :open="showClearConfirm" @update:open="showClearConfirm = $event">
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

    <div class="rounded-lg border bg-card overflow-hidden">
      <!-- ========== 1. 大屏表头 (Large >= 1024px) ========== -->
      <div class="hidden lg:flex items-center gap-4 px-4 py-2 border-b bg-muted/20 text-sm text-muted-foreground font-medium">
        <span class="w-16 shrink-0 pl-1">序号</span>
        <span class="w-56 shrink-0 px-2 pl-6">标题及渠道</span>
        <span class="flex-1 min-w-0 px-2 font-not-medium">内容详情</span>
        <span class="w-40 shrink-0 text-right">发送时间</span>
      </div>

      <!-- ========== 2. 中屏表头 (Medium 640px - 1024px) ========== -->
      <div class="hidden sm:flex lg:hidden items-center gap-4 px-4 py-2 border-b bg-muted/20 text-sm text-muted-foreground font-medium">
        <span class="w-48 shrink-0">项目</span>
        <span class="flex-1 min-w-0">内容摘要</span>
        <span class="w-40 shrink-0 text-right">时间信息</span>
      </div>

      <!-- 列表内容 -->
      <div class="divide-y text-sm">
        <div v-if="logs.length === 0 && !loading" class="text-sm text-muted-foreground text-center py-8">
          暂无推送记录
        </div>

        <!-- ========== 1. 小屏布局 (Small < 640px) - 用户调好 ========== -->
        <div v-for="(log, index) in logs" :key="`small-${log.id}`"
          class="sm:hidden p-3 hover:bg-muted/50 transition-colors cursor-pointer group" @click="showDetail(log)"
          :class="[selectedLogId === log.id && 'bg-accent/50']">
          <div class="flex items-start justify-between mb-3 border-b border-border/40 pb-2">
            <div class="flex items-center gap-2 flex-1 min-w-0 mr-2">
              <span class="text-[10px] text-muted-foreground shrink-0">#{{ getLogIndex(index) }}</span>
              <span class="font-bold text-sm truncate">{{ log.title }}</span>
            </div>
            <span
              :class="['h-2 w-2 mt-1.5 rounded-full shrink-0', log.status === LOG_STATUS.SUCCESS ? 'bg-green-500 shadow-[0_0_8px_rgba(34,197,94,0.4)]' : 'bg-red-500 shadow-[0_0_8px_rgba(239,68,68,0.4)]']"></span>
          </div>

          <!-- 详情信息列表 -->
          <div class="space-y-1.5 text-xs text-muted-foreground mb-1 px-1">
            <div v-if="log.channel_name" class="flex items-center gap-3">
              <span class="w-8 shrink-0 font-medium opacity-70">渠道:</span>
              <span class="text-foreground bg-muted/40 px-1.5 py-0.5 rounded text-[10px]">{{ log.channel_name }}</span>
            </div>
            <div class="flex items-start gap-3">
              <span class="w-8 shrink-0 font-medium mt-0.5 opacity-70">内容:</span>
              <div class="flex-1 min-w-0 text-foreground break-all leading-relaxed line-clamp-2">
                {{ log.content || '-' }}
              </div>
            </div>
            <div class="flex items-center gap-3">
              <span class="w-8 shrink-0 font-medium opacity-70">时间:</span>
              <span class="text-[10px] text-muted-foreground">{{ formatDate(log.created_at) }}</span>
            </div>
          </div>
        </div>

        <!-- ========== 2. 中屏布局 (Medium 640px - 1024px) - 新抽取优化 ========== -->
        <div v-for="log in logs" :key="`medium-${log.id}`"
          class="hidden sm:flex lg:hidden items-center gap-4 px-4 py-2.5 hover:bg-muted/50 transition-colors cursor-pointer group"
          :class="[selectedLogId === log.id && 'bg-accent/50']" @click="showDetail(log)">
          <div class="w-48 shrink-0 flex items-center gap-3 min-w-0">
            <span :class="['h-2 w-2 rounded-full shrink-0', log.status === LOG_STATUS.SUCCESS ? 'bg-green-500 shadow-[0_0_8px_rgba(34,197,94,0.3)]' : 'bg-red-500 shadow-[0_0_8px_rgba(239,68,68,0.3)]']"></span>
            <div class="flex flex-col gap-0.5 min-w-0">
              <span class="font-medium truncate text-sm" :title="log.title">{{ log.title }}</span>
              <span v-if="log.channel_name" class="text-[10px] text-muted-foreground opacity-60">[{{ log.channel_name }}]</span>
            </div>
          </div>
          <span class="flex-1 min-w-0 text-xs text-muted-foreground line-clamp-1" :title="log.content">
            {{ log.content || '-' }}
          </span>
          <span class="w-40 shrink-0 text-right text-xs text-muted-foreground tabular-nums opacity-60">
            {{ formatDate(log.created_at) }}
          </span>
        </div>

        <!-- ========== 3. 大屏布局 (Large >= 1024px) - 用户调好 ========== -->
        <div v-for="(log, index) in logs" :key="`large-${log.id}`"
          class="hidden lg:flex items-center gap-4 px-4 py-2 hover:bg-muted/50 transition-colors cursor-pointer group"
          :class="[selectedLogId === log.id && 'bg-accent/50']" @click="showDetail(log)">
          <span class="w-16 shrink-0 text-muted-foreground text-[11px] tabular-nums pl-1">#{{ getLogIndex(index) }}</span>
          <div class="w-56 shrink-0 flex items-center gap-3 min-w-0 text-[13px]">
            <span :class="['h-2 w-2 rounded-full shrink-0', log.status === LOG_STATUS.SUCCESS ? 'bg-green-500 shadow-[0_0_8px_rgba(34,197,94,0.3)]' : 'bg-red-500 shadow-[0_0_8px_rgba(239,68,68,0.3)]']"></span>
            <span class="truncate" :title="log.title">
              <span v-if="log.channel_name" class="mr-1 text-muted-foreground opacity-60">[{{ log.channel_name }}]</span>{{ log.title }}
            </span>
          </div>
          <span class="flex-1 min-w-0 text-[13px] text-muted-foreground truncate" :title="log.content">
            {{ log.content || '-' }}
          </span>
          <span class="w-40 shrink-0 text-right text-[13px] text-muted-foreground tabular-nums opacity-60">
            {{ formatDate(log.created_at) }}
          </span>
        </div>
      </div>

      <!-- 分页 -->
      <Pagination :total="total" :page="currentPage" @update:page="handlePageChange" />
    </div>

    <BaihuDialog v-model:open="detailDialogProps.open" title="日志详情" @update:open="onDialogClose">
      <template #extra>
        <Badge variant="outline" :class="[
          'px-2 py-0.5 text-[10px] font-bold rounded-md border shadow-sm transition-all duration-300',
          selectedLog ? getStatusBadgeClass(selectedLog.status) : ''
        ]">
          <div class="flex items-center gap-1 uppercase tracking-tighter">
            <Check v-if="selectedLog?.status === LOG_STATUS.SUCCESS" class="h-3 w-3" />
            <X v-else class="h-3 w-3" />
            <span>{{ selectedLog?.status === LOG_STATUS.SUCCESS ? '发送成功' : '发送失败' }}</span>
          </div>
        </Badge>
      </template>

      <!-- 核心内容区 (依赖 BaihuDialog 自带的 p-6) -->
      <div class="space-y-8">
        <!-- 1. 基础信息列表 (扁平化布局) -->
        <div class="grid grid-cols-1 gap-4 text-sm px-2">
          <div class="flex justify-between items-center group cursor-default">
            <span class="text-muted-foreground/60 font-medium">消息标题</span>
            <span class="font-semibold text-foreground tracking-tight">{{ detailDialogProps.title }}</span>
          </div>
          <div v-if="selectedLog?.channel_name" class="flex justify-between items-center group cursor-default">
            <span class="text-muted-foreground/60 font-medium">推送渠道</span>
            <Badge variant="secondary" class="font-normal text-[11px] bg-muted/40 hover:bg-muted/60 transition-colors">
              {{ selectedLog.channel_name }}
            </Badge>
          </div>
          <div class="flex justify-between items-center group cursor-default">
            <span class="text-muted-foreground/60 font-medium">发送时间</span>
            <span class="font-mono text-[11px] text-muted-foreground/80 tabular-nums">
              {{ selectedLog ? formatDate(selectedLog.created_at) : '-' }}
            </span>
          </div>
        </div>

        <!-- 2. 推送内容展示 -->
        <div class="space-y-3">
          <div class="flex items-center gap-2 px-2">
            <div class="w-1 h-3.5 bg-primary/40 rounded-full" />
            <span class="text-xs font-bold text-muted-foreground uppercase tracking-widest">推送内容</span>
          </div>
          <div class="relative group">
            <!-- 装饰性背景，模拟高级卡片 -->
            <div class="absolute -inset-0.5 bg-gradient-to-br from-primary/5 via-transparent to-primary/5 rounded-2xl blur opacity-20 group-hover:opacity-40 transition duration-500" />
            <div v-if="detailDialogProps.content"
              class="relative bg-background/50 dark:bg-muted/10 p-5 rounded-2xl border border-border/40 whitespace-pre-wrap break-all leading-relaxed shadow-[0_2px_15px_-3px_rgba(0,0,0,0.03)] text-[13px] text-foreground/90 font-medium max-h-[50vh] overflow-y-auto custom-scrollbar"
              v-html="renderedContent">
            </div>
            <div v-else class="relative bg-muted/20 p-6 rounded-2xl border border-dashed border-border/60 text-center text-sm text-muted-foreground">
              此消息无标准推送内容
            </div>
          </div>
        </div>

        <!-- 3. 错误信息 (仅当存在时展示) -->
        <div v-if="detailDialogProps.error" class="space-y-3">
          <div class="flex items-center gap-2 px-2">
            <div class="w-1 h-3.5 bg-destructive/40 rounded-full" />
            <span class="text-xs font-bold text-destructive/80 uppercase tracking-widest">错误报告</span>
          </div>
          <div class="bg-destructive/[0.03] p-5 rounded-2xl border border-destructive/10 text-[13px] text-destructive/90 font-mono leading-relaxed break-all max-h-[200px] overflow-y-auto custom-scrollbar">
            {{ detailDialogProps.error }}
          </div>
        </div>
      </div>
    </BaihuDialog>
  </div>
</template>
