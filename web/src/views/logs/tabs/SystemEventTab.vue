<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { api, type AppLog, LOG_CATEGORY, LOG_LEVEL } from '@/api'
import { Button } from '@/components/ui/button'
import {
    Info, AlertTriangle, AlertCircle
} from 'lucide-vue-next'
import { Badge } from '@/components/ui/badge'
import Pagination from '@/components/Pagination.vue'
import BaihuDialog from '@/components/ui/BaihuDialog.vue'
import { toast } from 'vue-sonner'
import { format } from 'date-fns'
import { useSiteSettings } from '@/composables/useSiteSettings'
import { useEventBus } from '@/composables/useEventBus'
import { LOG_EVENTS } from '@/constants'

const props = defineProps<{
    filters: {
        level: string
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
            category: LOG_CATEGORY.SYSTEM_NOTICE,
            level: props.filters.level === 'all' ? undefined : props.filters.level,
            keyword: props.filters.keyword || undefined,
            page: currentPage.value,
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
        await api.appLogs.clear(LOG_CATEGORY.SYSTEM_NOTICE)
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

// 实时更新：当有新系统事件产生且用户在第一页时刷新
useEventBus([LOG_EVENTS.ADDED], (payload) => {
    if (payload && payload.category === LOG_CATEGORY.SYSTEM_NOTICE) {
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
        <BaihuDialog v-model:open="showClearConfirm" title="清空系统事件确认">
            <div class="text-sm text-muted-foreground leading-relaxed">
                此操作将永久清空当前分类下的所有系统事件记录，操作后无法恢复。确认要继续吗？
            </div>
            <template #footer>
                <Button variant="ghost" @click="showClearConfirm = false">取消</Button>
                <Button variant="destructive" class="shadow-lg shadow-destructive/20" @click="handleClear">确认清空</Button>
            </template>
        </BaihuDialog>

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

                <!-- ========== 1. 小屏布局 (Small < 640px) - 统一风格 ========== -->
                <div v-for="(log, index) in logs" :key="`small-${log.id}`"
                    class="sm:hidden p-3 hover:bg-muted/50 transition-colors cursor-pointer group"
                    :class="[selectedLogId === log.id && 'bg-accent/50']" @click="showDetail(log)">
                    <div class="flex items-start justify-between mb-3 border-b border-border/40 pb-2">
                        <div class="flex items-center gap-2 flex-1 min-w-0 mr-2">
                            <span class="text-[10px] text-muted-foreground shrink-0 tabular-nums">#{{ total - (currentPage - 1) * pageSize - index }}</span>
                            <span class="font-bold text-sm truncate" :title="log.title">{{ log.title }}</span>
                        </div>
                        <span :class="['h-2 w-2 mt-1.5 rounded-full shrink-0 shadow-[0_0_8px]',
                            log.level === LOG_LEVEL.INFO ? 'bg-blue-500 shadow-blue-500/40' :
                                log.level === LOG_LEVEL.WARNING ? 'bg-yellow-500 shadow-yellow-500/40' : 'bg-red-500 shadow-red-500/40']"></span>
                    </div>

                    <!-- 详情信息列表 (仿 PushLog 格式) -->
                    <div class="space-y-1.5 text-xs text-muted-foreground mb-1 px-1">
                        <div class="flex items-center gap-3">
                            <span class="w-8 shrink-0 font-medium opacity-70">级别:</span>
                            <span :class="['px-1.5 py-0.5 rounded text-[10px] font-medium',
                                log.level === LOG_LEVEL.INFO ? 'bg-blue-500/10 text-blue-500' :
                                    log.level === LOG_LEVEL.WARNING ? 'bg-yellow-500/10 text-yellow-500' : 'bg-red-500/10 text-red-500']">
                                {{ log.level === LOG_LEVEL.INFO ? '信息' : log.level === LOG_LEVEL.WARNING ? '警告' : '错误' }}
                            </span>
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
                    <span class="w-16 shrink-0 text-muted-foreground text-[11px] tabular-nums pl-1">#{{ total - (currentPage - 1) * pageSize - index }}</span>
                    <div class="w-56 shrink-0 flex items-center gap-3 min-w-0 text-[13px]">
                        <component :is="getLevelIcon(log.level)" :class="['h-4 w-4 shrink-0 opacity-80',
                            log.level === LOG_LEVEL.INFO ? 'text-blue-500' :
                                log.level === LOG_LEVEL.WARNING ? 'text-yellow-500' : 'text-red-500']" />
                        <span class="truncate" :title="log.title">{{ log.title }}</span>
                    </div>
                    <span class="flex-1 min-w-0 text-[13px] text-muted-foreground truncate"
                        :title="log.content">
                        {{ log.content || '-' }}
                    </span>
                    <span class="w-40 shrink-0 text-right text-[13px] text-muted-foreground tabular-nums opacity-60">
                        {{ formatDate(log.created_at) }}
                    </span>
                </div>
            </div>

            <Pagination :total="total" :page="currentPage" @update:page="handlePageChange" />
        </div>

        <BaihuDialog v-model:open="detailDialogProps.open" :title="detailDialogProps.title" @update:open="onDialogClose">
            <template #description>
                <div class="flex items-center gap-2">
                    <Badge v-if="selectedLog" variant="outline" :class="[
                        'px-2 py-0.5 text-[10px] font-bold rounded-md border shadow-sm',
                        getLevelBadgeClass(selectedLog.level)
                    ]">
                        <div class="flex items-center gap-1 uppercase tracking-tighter">
                            <component :is="getLevelIcon(selectedLog.level)" class="h-3 w-3" />
                            <span>{{ selectedLog.level === LOG_LEVEL.INFO ? '信息' : selectedLog.level === LOG_LEVEL.WARNING ? '警告' : '错误' }}</span>
                        </div>
                    </Badge>
                    <span class="hidden sm:inline text-[10px] text-muted-foreground font-mono">{{ selectedLog ? formatDate(selectedLog.created_at) : '-' }}</span>
                </div>
            </template>

            <div class="space-y-6">
                <div class="space-y-4">
                    <div class="p-4 rounded-xl bg-muted/20 border border-border/10 space-y-2">
                        <p class="text-[10px] uppercase tracking-widest text-muted-foreground font-bold">事件内容</p>
                        <div v-if="detailDialogProps.content" class="text-sm leading-relaxed text-foreground/80 break-all whitespace-pre-wrap">
                            {{ detailDialogProps.content }}
                        </div>
                        <div v-else class="text-sm text-muted-foreground italic">无内容</div>
                    </div>

                    <div v-if="detailDialogProps.error" class="p-4 rounded-xl bg-destructive/5 border border-destructive/10 space-y-2">
                        <p class="text-[10px] uppercase tracking-widest text-destructive font-bold">错误堆栈/信息</p>
                        <div class="text-sm leading-relaxed text-destructive/90 break-all whitespace-pre-wrap font-mono bg-destructive/5 p-3 rounded-lg border border-destructive/10">
                            {{ detailDialogProps.error }}
                        </div>
                    </div>

                    <!-- 移动端/小屏下在右下角展示时间 -->
                    <div v-if="selectedLog" class="sm:hidden flex justify-end text-[10px] text-muted-foreground font-mono opacity-80 pt-1">
                        {{ formatDate(selectedLog.created_at) }}
                    </div>
                </div>
            </div>
        </BaihuDialog>
    </div>
</template>
