<script setup lang="ts">
import { ref, onMounted } from 'vue'
import Pagination from '@/components/Pagination.vue'
import { Loader2 } from 'lucide-vue-next'
import TextOverflow from '@/components/TextOverflow.vue'
import { api, LOG_CATEGORY } from '@/api'
import { toast } from 'vue-sonner'
import { useSiteSettings } from '@/composables/useSiteSettings'
import { useEventBus } from '@/composables/useEventBus'
import { LOG_EVENTS } from '@/constants'
import BaihuDialog from '@/components/ui/BaihuDialog.vue'

const props = defineProps<{
    username: string
}>()

const { pageSize } = useSiteSettings()

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

const logs = ref<LoginLog[]>([])
const currentPage = ref(1)
const total = ref(0)
const loading = ref(false)

// IP 地理位置弹窗
const ipDialogOpen = ref(false)
const ipGeoInfo = ref<IpGeoInfo | null>(null)
const ipGeoLoading = ref(false)
const selectedIp = ref('')

async function showIpInfo(ip: string) {
    selectedIp.value = ip
    ipDialogOpen.value = true
    ipGeoLoading.value = true
    ipGeoInfo.value = null

    try {
        const res = await fetch(`https://api.ip.sb/geoip/${ip}`)
        if (!res.ok) throw new Error('请求失败')
        ipGeoInfo.value = await res.json()
    } catch {
        toast.error('获取 IP 信息失败')
    } finally {
        ipGeoLoading.value = false
    }
}

async function loadLogs() {
    loading.value = true
    try {
        const res = await api.settings.getLoginLogs({
            page: currentPage.value,
            page_size: pageSize.value,
            username: props.username || undefined
        })
        logs.value = res.data
        total.value = res.total
    } catch {
        toast.error('加载登录日志失败')
    } finally {
        loading.value = false
    }
}

function handlePageChange(page: number) {
    currentPage.value = page
    loadLogs()
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
                <div v-if="logs.length === 0" class="text-sm text-muted-foreground text-center py-8">
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
                            <span class="text-[10px] text-muted-foreground">{{ log.created_at }}</span>
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
                    <span class="w-40 shrink-0 text-right text-xs text-muted-foreground tabular-nums opacity-60">
                        {{ log.created_at }}
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
                    <span class="w-40 shrink-0 text-right text-[13px] text-muted-foreground tabular-nums opacity-60">{{ log.created_at }}</span>
                </div>
            </div>
            <!-- 分页 -->
            <Pagination :total="total" :page="currentPage" @update:page="handlePageChange" />
        </div>

        <!-- IP 地理位置弹窗 -->
        <BaihuDialog v-model:open="ipDialogOpen" title="IP 详情">
            <template #description>
                <code class="text-[10px] bg-primary/10 text-primary px-2 py-0.5 rounded font-mono tracking-wider">{{ selectedIp }}</code>
            </template>

            <div v-if="ipGeoLoading" class="flex items-center justify-center py-12">
                <Loader2 class="h-8 w-8 animate-spin text-primary/40" />
            </div>
            <div v-else-if="ipGeoInfo" class="space-y-4">
                <div class="grid grid-cols-1 sm:grid-cols-2 gap-3 sm:gap-4">
                    <div class="p-3 rounded-xl bg-muted/20 border border-border/10 space-y-1">
                        <p class="text-[10px] uppercase tracking-wider text-muted-foreground font-bold">国家/地区</p>
                        <p class="text-sm font-medium">{{ ipGeoInfo.country }} ({{ ipGeoInfo.country_code }})</p>
                    </div>
                    <div class="p-3 rounded-xl bg-muted/20 border border-border/10 space-y-1">
                        <p class="text-[10px] uppercase tracking-wider text-muted-foreground font-bold">时区</p>
                        <p class="text-sm font-medium">{{ ipGeoInfo.timezone || '-' }}</p>
                    </div>
                </div>

                <div class="p-4 rounded-xl bg-muted/20 border border-border/10 space-y-3">
                    <div class="flex justify-between items-center text-sm gap-2">
                        <span class="text-muted-foreground shrink-0">运营商</span>
                        <span class="font-medium truncate text-right">{{ ipGeoInfo.isp || '-' }}</span>
                    </div>
                    <div class="flex justify-between items-center text-sm gap-2">
                        <span class="text-muted-foreground shrink-0">组织</span>
                        <span class="font-medium truncate text-right">{{ ipGeoInfo.organization || '-' }}</span>
                    </div>
                    <div class="flex justify-between items-start text-sm gap-2">
                        <span class="text-muted-foreground shrink-0 mt-0.5">ASN</span>
                        <div class="text-right min-w-0">
                            <div class="font-medium font-mono text-xs text-primary/80">AS{{ ipGeoInfo.asn }}</div>
                            <div class="text-xs text-muted-foreground truncate" :title="ipGeoInfo.asn_organization">{{ ipGeoInfo.asn_organization || '-' }}</div>
                        </div>
                    </div>
                </div>

                <div class="flex items-center justify-between p-3 rounded-xl bg-primary/5 border border-primary/10 text-[11px] sm:text-xs">
                    <span class="text-muted-foreground font-medium">地理坐标</span>
                    <span class="font-mono text-primary/70">{{ ipGeoInfo.latitude }}, {{ ipGeoInfo.longitude }}</span>
                </div>
            </div>
            <div v-else class="text-center text-muted-foreground py-8 italic text-sm">
                无法获取 IP 信息，请稍后再试
            </div>
        </BaihuDialog>
    </div>
</template>
