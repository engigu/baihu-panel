<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { api, type AppLog, LOG_CATEGORY, LOG_LEVEL } from '@/api'
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
    RefreshCw, Trash2, Search, Info, AlertTriangle, AlertCircle
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
    level: 'all',
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
            category: LOG_CATEGORY.SYSTEM_NOTICE,
            level: filters.value.level === 'all' ? undefined : filters.value.level,
            keyword: filters.value.keyword || undefined,
            page: filters.value.page,
            page_size: pageSize.value
        })
        logs.value = res.data || []
        total.value = res.total || 0
    } catch (e: any) {
        toast.error(e.message || '获取系统事件失败')
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

function handleLevelChange(val: any) {
    if (val === null || val === undefined) return
    filters.value.level = String(val)
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
        await api.appLogs.clear(LOG_CATEGORY.SYSTEM_NOTICE)
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

function getLevelBadgeClass(level: string) {
    switch (level) {
        case LOG_LEVEL.INFO:
            return 'bg-blue-500/15 text-blue-500 border-blue-500/30'
        case LOG_LEVEL.WARNING:
            return 'bg-amber-500/15 text-amber-500 border-amber-500/30'
        case LOG_LEVEL.ERROR:
            return 'bg-red-500/15 text-red-500 border-red-500/30'
        default:
            return 'bg-secondary text-secondary-foreground border-transparent'
    }
}

function getLevelIcon(level: string) {
    switch (level) {
        case LOG_LEVEL.INFO:
            return Info
        case LOG_LEVEL.WARNING:
            return AlertTriangle
        case LOG_LEVEL.ERROR:
            return AlertCircle
        default:
            return Info
    }
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
    <div class="space-y-4">
        <div class="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4 w-full">
            <div class="flex items-center gap-2 w-full lg:w-auto lg:ml-auto">
                <div class="relative w-full sm:w-60 group">
                    <Search class="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground group-focus-within:text-primary transition-colors" />
                    <Input v-model="filters.keyword" placeholder="搜索标题或内容..." class="h-9 pl-9 w-full text-sm bg-muted/20 border-muted-foreground/10 focus:bg-background"
                        @input="handleSearch" />
                </div>
                <div class="relative flex-1 sm:flex-none sm:w-28">
                    <Select :model-value="filters.level" @update:model-value="handleLevelChange">
                        <SelectTrigger class="h-9 w-full text-sm bg-muted/20 border-muted-foreground/10">
                            <SelectValue placeholder="级别" />
                        </SelectTrigger>
                        <SelectContent>
                            <SelectItem value="all">所有级别</SelectItem>
                            <SelectItem :value="LOG_LEVEL.INFO">信息</SelectItem>
                            <SelectItem :value="LOG_LEVEL.WARNING">警告</SelectItem>
                            <SelectItem :value="LOG_LEVEL.ERROR">错误</SelectItem>
                        </SelectContent>
                    </Select>
                </div>
                <Button variant="outline" size="icon" class="h-9 w-9 shrink-0 sm:flex" @click="fetchLogs" :disabled="loading"
                    title="刷新">
                    <RefreshCw class="h-4 w-4" :class="{ 'animate-spin': loading }" />
                </Button>
            </div>
            <AlertDialog :open="showClearConfirm" @update:open="showClearConfirm = $event">
                <Button variant="outline"
                    class="h-9 px-4 shrink-0 text-sm text-destructive hover:bg-destructive/10 hover:text-destructive border-destructive/20 w-full sm:w-auto"
                    @click="showClearConfirm = true">
                    <Trash2 class="h-4 w-4 mr-2" /> <span>清空记录</span>
                </Button>
                <AlertDialogContent>
                    <AlertDialogHeader>
                        <AlertDialogTitle>确认清空所有系统事件？</AlertDialogTitle>
                        <AlertDialogDescription>
                            此操作将永久清空当前分类下的所有系统事件记录，操作后无法恢复。确认要继续吗？
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

        <div class="rounded-lg border bg-card overflow-hidden">
            <!-- ========== 1. 大屏表头 (Large >= 1024px) ========== -->
            <div class="hidden lg:flex items-center gap-4 px-4 py-2 border-b bg-muted/20 text-sm text-muted-foreground font-medium">
                <span class="w-16 shrink-0 pl-1">序号</span>
                <span class="w-56 shrink-0 px-2 pl-8">事件信息</span>
                <span class="flex-1 min-w-0 px-2">详情内容</span>
                <span class="w-40 shrink-0 text-right">发生时间</span>
            </div>

            <!-- ========== 2. 中屏表头 (Medium 640px - 1024px) ========== -->
            <div class="hidden sm:flex lg:hidden items-center gap-4 px-4 py-2 border-b bg-muted/20 text-sm text-muted-foreground font-medium">
                <span class="w-60 shrink-0">事件信息</span>
                <span class="flex-1 min-w-0">详情内容</span>
                <span class="w-40 shrink-0 text-right">发生时间</span>
            </div>

            <!-- 列表内容 -->
            <div class="divide-y">
                <div v-if="logs.length === 0 && !loading" class="text-sm text-muted-foreground text-center py-8">
                    暂无系统事件
                </div>

                <!-- ========== 1. 小屏布局 (Small < 640px) - 用户调好 ========== -->
                <div v-for="(log, index) in logs" :key="`small-${log.id}`"
                    class="sm:hidden p-3 hover:bg-muted/50 transition-colors cursor-pointer group"
                    :class="[selectedLogId === log.id && 'bg-accent/50']" @click="showDetail(log)">
                    <div class="flex items-start justify-between mb-2">
                        <div class="flex items-center gap-2 flex-1 min-w-0 mr-2">
                            <span class="text-xs text-muted-foreground shrink-0 tabular-nums">#{{ total - (filters.page - 1) * pageSize - index }}</span>
                            <component :is="getLevelIcon(log.level)" :class="['h-4 w-4 shrink-0',
                                log.level === LOG_LEVEL.INFO ? 'text-blue-500' :
                                    log.level === LOG_LEVEL.WARNING ? 'text-yellow-500' : 'text-red-500']" />
                            <span class="font-medium text-sm truncate" :title="log.title">{{ log.title }}</span>
                        </div>
                    </div>
                    <div class="bg-muted/30 rounded px-2 py-1.5 mb-2">
                        <div class="text-muted-foreground text-xs truncate">
                            {{ log.content || '-' }}
                        </div>
                    </div>
                    <div class="text-[10px] text-muted-foreground text-right tabular-nums">
                        {{ formatDate(log.created_at) }}
                    </div>
                </div>

                <!-- ========== 2. 中屏布局 (Medium 640px - 1024px) - 新抽取优化 ========== -->
                <div v-for="log in logs" :key="`medium-${log.id}`"
                    class="hidden sm:flex lg:hidden items-center gap-4 px-4 py-2.5 hover:bg-muted/50 transition-colors cursor-pointer group"
                    :class="[selectedLogId === log.id && 'bg-accent/50']" @click="showDetail(log)">
                    <div class="w-60 shrink-0 flex items-center gap-3 min-w-0 font-medium text-sm">
                        <component :is="getLevelIcon(log.level)" :class="['h-3.5 w-3.5 shrink-0 opacity-80',
                            log.level === LOG_LEVEL.INFO ? 'text-blue-500' :
                                log.level === LOG_LEVEL.WARNING ? 'text-yellow-500' : 'text-red-500']" />
                        <span class="truncate" :title="log.title">{{ log.title }}</span>
                    </div>
                    <span class="flex-1 min-w-0 text-sm text-muted-foreground line-clamp-1" :title="log.content">
                        {{ log.content || '-' }}
                    </span>
                    <span class="w-40 shrink-0 text-right text-xs text-muted-foreground tabular-nums opacity-60">
                        {{ formatDate(log.created_at) }}
                    </span>
                </div>

                <div v-for="(log, index) in logs" :key="`large-${log.id}`"
                    class="hidden lg:flex items-center gap-4 px-4 py-2 hover:bg-muted/50 transition-colors cursor-pointer group"
                    :class="[selectedLogId === log.id && 'bg-accent/50']" @click="showDetail(log)">
                    <span class="w-16 shrink-0 text-muted-foreground text-sm tabular-nums pl-1">#{{ total - (filters.page - 1) * pageSize - index }}</span>
                    <div class="w-56 shrink-0 flex items-center gap-3 min-w-0 font-medium text-sm">
                        <component :is="getLevelIcon(log.level)" :class="['h-4 w-4 shrink-0 opacity-80',
                            log.level === LOG_LEVEL.INFO ? 'text-blue-500' :
                                log.level === LOG_LEVEL.WARNING ? 'text-yellow-500' : 'text-red-500']" />
                        <span class="truncate" :title="log.title">{{ log.title }}</span>
                    </div>
                    <span class="flex-1 min-w-0 text-sm text-muted-foreground truncate"
                        :title="log.content">
                        {{ log.content || '-' }}
                    </span>
                    <span class="w-40 shrink-0 text-right text-xs text-muted-foreground tabular-nums opacity-60">
                        {{ formatDate(log.created_at) }}
                    </span>
                </div>
            </div>

            <Pagination :total="total" :page="filters.page" @update:page="handlePageChange" />
        </div>

        <Dialog v-model:open="detailDialogProps.open" @update:open="onDialogClose">
            <DialogContent class="sm:max-w-2xl max-h-[90vh] flex flex-col p-0 overflow-hidden">
                <DialogHeader class="px-6 py-4 border-b bg-muted/20">
                    <div class="flex items-center justify-between pr-8">
                        <DialogTitle>事件详情</DialogTitle>
                        <Badge variant="outline" :class="[
                            'px-2 py-0.5 text-[10px] font-bold rounded-md border shadow-sm',
                            selectedLog ? getLevelBadgeClass(selectedLog.level) : ''
                        ]">
                            <div class="flex items-center gap-1 uppercase tracking-tighter">
                                <component :is="getLevelIcon(selectedLog?.level || 'info')" class="h-3 w-3" />
                                <span>{{ selectedLog?.level || 'INFO' }}</span>
                            </div>
                        </Badge>
                    </div>
                </DialogHeader>

                <div class="flex-1 overflow-y-auto">
                    <div class="px-6 py-4 border-b space-y-3 bg-card">
                        <div class="flex justify-between items-center text-sm">
                            <span class="text-muted-foreground font-medium">标题</span>
                            <span class="font-bold text-foreground">{{ detailDialogProps.title }}</span>
                        </div>
                        <div class="flex justify-between items-center text-sm">
                            <span class="text-muted-foreground font-medium">发生时间</span>
                            <span class="font-mono text-xs text-muted-foreground">{{ selectedLog ?
                                formatDate(selectedLog.created_at) : '-' }}</span>
                        </div>
                    </div>

                    <div class="flex flex-col min-h-0 bg-muted/5">
                        <div
                            class="px-6 py-2.5 text-[10px] font-bold text-muted-foreground border-b bg-muted/10 uppercase tracking-widest">
                            内容详情
                        </div>
                        <div class="p-6">
                            <div v-if="detailDialogProps.content"
                                class="text-sm text-foreground bg-muted/20 p-5 rounded-xl border border-border/50 whitespace-pre-wrap break-all leading-relaxed shadow-sm">
                                {{ detailDialogProps.content }}
                            </div>
                            <div v-else class="text-sm text-muted-foreground italic py-2">无内容</div>
                        </div>

                        <template v-if="detailDialogProps.error">
                            <div
                                class="px-6 py-2.5 text-[10px] font-bold uppercase tracking-widest border-y bg-muted/10 text-muted-foreground border-border/60">
                                错误信息
                            </div>
                            <div class="p-6">
                                <div v-if="detailDialogProps.error"
                                    class="text-sm p-5 rounded-xl border whitespace-pre-wrap break-all leading-relaxed shadow-sm bg-muted/20 border-border/60 text-foreground">
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
