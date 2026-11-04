<script setup lang="ts">
import { ref, onMounted } from 'vue'
import Pagination from '@/components/Pagination.vue'
import { Loader2, Copy, MapPin } from 'lucide-vue-next'
import TextOverflow from '@/components/TextOverflow.vue'
import { api, LOG_CATEGORY } from '@/api'
import { toast } from 'vue-sonner'
import { usePaginatedList } from '@/composables/usePaginatedList'
import { useEventBus } from '@/composables/useEventBus'
import { LOG_EVENTS } from '@/constants'
import BaihuDialog from '@/components/ui/BaihuDialog.vue'
import { formatDateTime } from '@/utils/date'
import { copyToClipboard } from '@/utils/clipboard'

const props = defineProps<{
    username: string
}>()

interface LoginLog {
    id: string
    username: string
    ip: string
    user_agent: string
    status: string
    message: string
    created_at: string
}

interface IpGeoInfo {
    ip: string
    country: string
    country_code: string
    organization: string
    isp: string
    asn: number
    asn_organization: string
    timezone: string
    latitude: number
    longitude: number
    continent_code: string
    offset: number
}

const {
    list: logs,
    total,
    page: currentPage,
    pageSize,
    loadData: loadLogs,
    handlePageChange,
    handlePageSizeChange
} = usePaginatedList<LoginLog>({
    pageSizeKey: 'bhp_login_logs_page_size',
    defaultPageSize: 10,
    fetchApi: (params) => api.settings.getLoginLogs({
        page: params.page,
        page_size: params.page_size,
        username: props.username || undefined
    }),
    errorMessage: '加载登录日志失败'
})

// IP 地理位置弹窗
const ipDialogOpen = ref(false)
const ipGeoInfo = ref<IpGeoInfo | null>(null)
const ipGeoLoading = ref(false)
const selectedIp = ref('')

async function showIpInfo(ip: string) {
    selectedIp.value = ip
    ipGeoLoading.value = true
    ipGeoInfo.value = null

    try {
        const res = await fetch(`https://api.ip.sb/geoip/${ip}`)
        if (!res.ok) throw new Error('请求失败')
        ipGeoInfo.value = await res.json()
        ipDialogOpen.value = true
    } catch (e) {
        ipDialogOpen.value = false
        toast.error('获取 IP 信息失败')
    } finally {
        ipGeoLoading.value = false
    }
}

function getCountryFlag(code?: string) {
    if (!code || code.length !== 2) return ''
    return String.fromCodePoint(...[...code.toUpperCase()].map(c => 0x1F1E6 + c.charCodeAt(0) - 65))
}

function formatUtcOffset(offset?: number) {
    if (offset === undefined || offset === null) return ''
    const hours = offset / 3600
    const sign = hours >= 0 ? '+' : ''
    return `UTC${sign}${hours}`
}

async function copyText(text: string, successMsg: string) {
    if (!text) return
    const success = await copyToClipboard(text)
    if (success) {
        toast.success(successMsg)
    } else {
        toast.error('复制失败')
    }
}

onMounted(loadLogs)

// 实时更新：当有新登录产生且用户在第一页时刷新
useEventBus([LOG_EVENTS.ADDED], (payload) => {
    if (payload && payload.category === LOG_CATEGORY.LOGIN_LOG) {
        if (currentPage.value === 1) {
            loadLogs()
        }
    }
})

defineExpose({
    loadLogs
})
</script>

<template>
    <div class="space-y-4">

        <div class="rounded-lg border bg-card overflow-hidden">
            <!-- ========== 1. 大屏表头 (Large >= 1024px) ========== -->
            <div class="hidden lg:flex items-center gap-4 px-4 py-2 border-b bg-muted/20 text-sm text-muted-foreground font-medium">
                <span class="w-16 shrink-0 pl-1">序号</span>
                <span class="w-32 shrink-0">用户信息</span>
                <span class="w-40 shrink-0">IP 地址</span>
                <span class="flex-1 min-w-0">User Agent</span>
                <span class="w-40 shrink-0 text-right">登录时间</span>
            </div>

            <!-- ========== 2. 中屏表头 (Medium 640px - 1024px) ========== -->
            <div class="hidden sm:flex lg:hidden items-center gap-4 px-4 py-2 border-b bg-muted/20 text-sm text-muted-foreground font-medium">
                <span class="w-24 shrink-0">用户信息</span>
                <span class="w-32 shrink-0">IP 地址</span>
                <span class="flex-1 min-w-0">设备信息 (UA)</span>
                <span class="w-40 shrink-0 text-right">登录时间</span>
            </div>

            <!-- 列表内容 -->
            <div class="divide-y text-sm">
                <div v-if="!logs || logs.length === 0" class="text-sm text-muted-foreground text-center py-8">
                    暂无登录日志
                </div>

                <!-- ========== 1. 小屏布局 (Small < 640px) - 统一风格 ========== -->
                <div v-for="(log, index) in logs" :key="`small-${log.id}`"
                    class="sm:hidden p-3 hover:bg-muted/50 transition-colors cursor-pointer group" @click="showIpInfo(log.ip)">
                    <div class="flex items-start justify-between mb-3 border-b border-border/40 pb-2">
                        <div class="flex items-center gap-2 flex-1 min-w-0 mr-2">
                            <span class="text-[10px] text-muted-foreground shrink-0 tabular-nums">#{{ total - (currentPage - 1) * pageSize - index }}</span>
                            <span class="font-bold text-sm truncate">{{ log.username }}</span>
                        </div>
                        <span :class="['h-2 w-2 mt-1.5 rounded-full shrink-0 shadow-[0_0_8px]',
                            log.status === 'success' ? 'bg-green-500 shadow-green-500/40' : 'bg-red-500 shadow-red-500/40']"></span>
                    </div>

                    <!-- 详情信息列表 (仿 PushLog 格式) -->
                    <div class="space-y-1.5 text-xs text-muted-foreground mb-1 px-1">
                        <div class="flex items-center gap-3">
                            <span class="w-8 shrink-0 font-medium opacity-70">IP:</span>
                            <span class="text-foreground bg-muted/40 px-1.5 py-0.5 rounded text-[10px] tabular-nums">{{ log.ip }}</span>
                        </div>
                        <div class="flex items-start gap-3">
                            <span class="w-8 shrink-0 font-medium mt-0.5 opacity-70">信息:</span>
                            <div class="flex-1 min-w-0 text-foreground break-all leading-relaxed line-clamp-2">
                                {{ log.message || '-' }}
                            </div>
                        </div>
                        <div class="flex items-center gap-3">
                            <span class="w-8 shrink-0 font-medium opacity-70">时间:</span>
                            <span class="text-[10px] text-muted-foreground"
                                  :title="log.created_at"
                            >{{ formatDateTime(log.created_at) }}</span>
                        </div>
                    </div>
                </div>

                <div v-for="log in logs" :key="`medium-${log.id}`"
                    class="hidden sm:flex lg:hidden items-center gap-4 px-4 py-2.5 hover:bg-muted/50 transition-colors">
                    <div class="w-24 shrink-0 flex items-center gap-2 min-w-0">
                        <span :class="['h-1.5 w-1.5 rounded-full shrink-0', log.status === 'success' ? 'bg-green-500 shadow-[0_0_8px_rgba(34,197,94,0.3)]' : 'bg-red-500 shadow-[0_0_8px_rgba(239,68,68,0.3)]']"></span>
                        <span class="font-medium truncate text-sm">{{ log.username }}</span>
                    </div>
                    <div class="w-32 shrink-0 overflow-hidden">
                        <code class="block w-full text-[11px] text-muted-foreground bg-muted/50 px-1.5 py-0.5 rounded truncate cursor-pointer hover:bg-muted/80 transition-colors tabular-nums"
                            @click="showIpInfo(log.ip)" :title="log.ip">{{ log.ip }}</code>
                    </div>
                    <span class="flex-1 min-w-0 text-xs text-muted-foreground line-clamp-1">
                        {{ log.user_agent || '-' }}
                    </span>
                    <span class="w-40 shrink-0 text-right text-xs text-muted-foreground tabular-nums opacity-60"
                          :title="log.created_at"
                    >
                        {{ formatDateTime(log.created_at) }}
                    </span>
                </div>

                <!-- ========== 3. 大屏布局 (Large >= 1024px) - 用户调好 ========== -->
                <div v-for="(log, index) in logs" :key="`large-${log.id}`"
                    class="hidden lg:flex items-center gap-4 px-4 py-2 hover:bg-muted/50 transition-colors">
                    <span class="w-16 shrink-0 text-muted-foreground text-[11px] pl-1">#{{ total - (currentPage - 1) * pageSize - index }}</span>
                    <div class="w-32 shrink-0 flex items-center gap-2 min-w-0 text-[13px]">
                        <span :class="['h-2 w-2 rounded-full shrink-0', log.status === 'success' ? 'bg-green-500 shadow-[0_0_8px_rgba(34,197,94,0.3)]' : 'bg-red-500 shadow-[0_0_8px_rgba(239,68,68,0.3)]']"></span>
                        <span class="truncate">{{ log.username }}</span>
                    </div>
                    <div class="w-40 shrink-0 overflow-hidden">
                        <code class="block w-full text-[13px] text-muted-foreground bg-muted px-2 py-1 rounded truncate cursor-pointer hover:bg-muted/80 transition-colors tabular-nums"
                            @click="showIpInfo(log.ip)" :title="log.ip">{{ log.ip }}</code>
                    </div>
                    <span class="flex-1 min-w-0 text-[13px] text-muted-foreground truncate">
                        <TextOverflow :text="log.user_agent || '-'" title="User Agent" />
                    </span>
                    <span class="w-40 shrink-0 text-right text-[13px] text-muted-foreground tabular-nums opacity-60"
                          :title="log.created_at"
                    >{{ formatDateTime(log.created_at) }}</span>
                </div>
            </div>
            <!-- 分页 -->
            <Pagination
                :total="total"
                :page="currentPage"
                :page-size="pageSize"
                @update:page="handlePageChange"
                @update:page-size="handlePageSizeChange"
            />
        </div>

        <!-- IP 地理位置弹窗 -->
        <BaihuDialog v-model:open="ipDialogOpen" title="IP 详情" size="sm" icon="Globe">
            <template #description>
                <button
                    type="button"
                    class="inline-flex items-center gap-1.5 bg-primary/10 hover:bg-primary/20 text-primary px-2.5 py-0.5 rounded-md font-inter text-xs font-medium tracking-tight transition-colors group cursor-pointer"
                    title="点击复制 IP 地址"
                    @click="copyText(selectedIp, 'IP 地址已复制')"
                >
                    <span>{{ selectedIp }}</span>
                    <Copy class="h-3 w-3 opacity-60 group-hover:opacity-100 transition-opacity" />
                </button>
            </template>

            <div v-if="ipGeoLoading" class="flex items-center justify-center py-12">
                <Loader2 class="h-8 w-8 animate-spin text-primary/40" />
            </div>
            <div v-else-if="ipGeoInfo" class="space-y-3 font-inter">
                <!-- 顶部快捷信息网格：国家/地区、时区 -->
                <div class="grid grid-cols-1 sm:grid-cols-2 gap-2.5 sm:gap-3">
                    <div class="p-3 rounded-xl bg-muted/20 border border-border/10 space-y-1">
                        <p class="text-[10px] uppercase tracking-wider text-muted-foreground font-bold font-inter">国家 / 地区</p>
                        <p class="text-sm font-semibold text-foreground flex items-center gap-1.5 flex-wrap font-inter">
                            <span v-if="getCountryFlag(ipGeoInfo.country_code)" class="text-base leading-none select-none">
                                {{ getCountryFlag(ipGeoInfo.country_code) }}
                            </span>
                            <span class="break-words">{{ ipGeoInfo.country || '-' }}</span>
                            <span v-if="ipGeoInfo.country_code" class="text-xs text-muted-foreground font-medium font-inter">
                                ({{ ipGeoInfo.country_code }})
                            </span>
                        </p>
                    </div>
                    <div class="p-3 rounded-xl bg-muted/20 border border-border/10 space-y-1">
                        <p class="text-[10px] uppercase tracking-wider text-muted-foreground font-bold font-inter">时区</p>
                        <p class="text-sm font-semibold text-foreground flex items-center gap-1.5 flex-wrap font-inter">
                            <span class="break-words">{{ ipGeoInfo.timezone || '-' }}</span>
                            <span v-if="ipGeoInfo.offset !== undefined && ipGeoInfo.offset !== null" class="text-xs text-muted-foreground font-medium font-inter">
                                ({{ formatUtcOffset(ipGeoInfo.offset) }})
                            </span>
                        </p>
                    </div>
                </div>

                <!-- 核心归属详情：运营商、组织、ASN（纯正 Inter 字体，左对齐自然折行） -->
                <div class="p-3.5 sm:p-4 rounded-xl bg-muted/20 border border-border/10 space-y-3 font-inter">
                    <!-- 运营商 -->
                    <div class="flex items-start gap-3 text-sm font-inter">
                        <span class="text-xs text-muted-foreground shrink-0 w-14 pt-0.5 font-medium font-inter">运营商</span>
                        <div class="font-medium text-foreground break-words flex-1 leading-relaxed select-text min-w-0 font-inter">
                            {{ ipGeoInfo.isp || '-' }}
                        </div>
                    </div>

                    <!-- 组织 -->
                    <div class="flex items-start gap-3 text-sm font-inter">
                        <span class="text-xs text-muted-foreground shrink-0 w-14 pt-0.5 font-medium font-inter">组织</span>
                        <div class="font-medium text-foreground break-words flex-1 leading-relaxed select-text min-w-0 font-inter">
                            {{ ipGeoInfo.organization || '-' }}
                        </div>
                    </div>

                    <!-- ASN -->
                    <div class="flex items-start gap-3 text-sm font-inter">
                        <span class="text-xs text-muted-foreground shrink-0 w-14 pt-0.5 font-medium font-inter">ASN</span>
                        <div class="flex-1 min-w-0 space-y-1.5 font-inter">
                            <div class="inline-flex items-center gap-1 px-2 py-0.5 rounded-md bg-primary/10 text-primary font-inter text-xs font-semibold tracking-wide">
                                AS{{ ipGeoInfo.asn || '-' }}
                            </div>
                            <div v-if="ipGeoInfo.asn_organization" class="text-xs text-muted-foreground break-words leading-relaxed select-text font-inter">
                                {{ ipGeoInfo.asn_organization }}
                            </div>
                        </div>
                    </div>
                </div>

                <!-- 地理坐标 -->
                <div v-if="ipGeoInfo.latitude !== undefined && ipGeoInfo.longitude !== undefined" class="flex items-center justify-between p-3 rounded-xl bg-primary/5 border border-primary/10 text-xs gap-2 font-inter">
                    <div class="flex items-center gap-2 text-muted-foreground font-medium shrink-0 font-inter">
                        <MapPin class="h-3.5 w-3.5 text-primary/70 shrink-0" />
                        <span>地理坐标</span>
                    </div>
                    <div class="flex items-center gap-1.5 min-w-0 justify-end font-inter">
                        <span class="font-inter text-primary/90 font-medium tabular-nums truncate select-all" :title="`${ipGeoInfo.latitude}, ${ipGeoInfo.longitude}`">
                            {{ ipGeoInfo.latitude }}, {{ ipGeoInfo.longitude }}
                        </span>
                        <button
                            type="button"
                            class="text-muted-foreground hover:text-foreground transition-colors p-1 rounded hover:bg-background/80 shrink-0 cursor-pointer"
                            title="复制坐标"
                            @click="copyText(`${ipGeoInfo.latitude}, ${ipGeoInfo.longitude}`, '地理坐标已复制')"
                        >
                            <Copy class="h-3.5 w-3.5" />
                        </button>
                    </div>
                </div>
            </div>
            <div v-else class="text-center text-muted-foreground py-8 italic text-sm font-inter">
                无法获取 IP 信息，请稍后再试
            </div>
        </BaihuDialog>
    </div>
</template>

<style scoped>
:deep(.font-inter),
.font-inter {
  font-family: 'Inter', 'Inter Variable', -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif !important;
}
</style>
