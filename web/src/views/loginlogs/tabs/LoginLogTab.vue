<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import Pagination from '@/components/Pagination.vue'
import { RefreshCw, Search, Loader2 } from 'lucide-vue-next'
import TextOverflow from '@/components/TextOverflow.vue'
import { api } from '@/api'
import { toast } from 'vue-sonner'
import { useSiteSettings } from '@/composables/useSiteSettings'
import {
    Dialog,
    DialogContent,
    DialogHeader,
    DialogTitle,
    DialogDescription
} from '@/components/ui/dialog'

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
const filterUsername = ref('')
const currentPage = ref(1)
const total = ref(0)
const loading = ref(false)
let searchTimer: ReturnType<typeof setTimeout> | null = null

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
            username: filterUsername.value || undefined
        })
        logs.value = res.data
        total.value = res.total
    } catch {
        toast.error('加载登录日志失败')
    } finally {
        loading.value = false
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

onMounted(loadLogs)
</script>

<template>
    <div class="space-y-4">
        <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4 w-full">
            <div class="flex items-center gap-2 w-full lg:w-auto lg:ml-auto">
                <div class="relative w-full sm:w-60 group">
                    <Search class="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground group-focus-within:text-primary transition-colors" />
                    <Input v-model="filterUsername" placeholder="搜索用户名..." class="h-9 pl-9 w-full text-sm bg-muted/20 border-muted-foreground/10 focus:bg-background"
                        @input="handleSearch" />
                </div>
                <Button variant="outline" size="icon" class="h-9 w-9 shrink-0" @click="loadLogs" :disabled="loading"
                    title="刷新">
                    <RefreshCw class="h-4 w-4" :class="{ 'animate-spin': loading }" />
                </Button>
            </div>
        </div>

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

                <!-- ========== 1. 小屏布局 (Small < 640px) - 用户调好 ========== -->
                <div v-for="(log, index) in logs" :key="`small-${log.id}`"
                    class="sm:hidden p-3 hover:bg-muted/50 transition-colors">
                    <div class="flex items-start justify-between mb-3 border-b border-border/40 pb-2">
                        <div class="flex items-center gap-2 flex-1 min-w-0 mr-2">
                            <span class="text-xs text-muted-foreground shrink-0">#{{ total - (currentPage - 1) * pageSize - index }}</span>
                            <span class="font-bold text-sm truncate">{{ log.username }}</span>
                        </div>
                        <span
                            :class="['h-2 w-2 mt-1.5 rounded-full shrink-0', log.status === 'success' ? 'bg-green-500 shadow-[0_0_8px_rgba(34,197,94,0.4)]' : 'bg-red-500 shadow-[0_0_8px_rgba(239,68,68,0.4)]']"></span>
                    </div>

                    <!-- 详情信息列表 -->
                    <div class="space-y-1.5 text-xs text-muted-foreground mb-1 px-1">
                        <div class="flex items-center gap-3">
                            <span class="w-8 shrink-0 font-medium opacity-70">IP:</span>
                            <span
                                class="text-foreground bg-muted/40 px-1.5 py-0.5 rounded cursor-pointer hover:bg-muted/80 transition-colors"
                                @click="showIpInfo(log.ip)">{{ log.ip }}</span>
                        </div>
                        <div class="flex items-center gap-3">
                            <span class="w-8 shrink-0 font-medium opacity-70">时间:</span>
                            <span class="text-muted-foreground">{{ log.created_at }}</span>
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
                    <span class="w-16 shrink-0 text-muted-foreground text-sm pl-1">#{{ total - (currentPage - 1) * pageSize - index }}</span>
                    <div class="w-32 shrink-0 flex items-center gap-2 min-w-0 font-medium text-sm">
                        <span :class="['h-2 w-2 rounded-full shrink-0', log.status === 'success' ? 'bg-green-500 shadow-[0_0_8px_rgba(34,197,94,0.3)]' : 'bg-red-500 shadow-[0_0_8px_rgba(239,68,68,0.3)]']"></span>
                        <span class="truncate">{{ log.username }}</span>
                    </div>
                    <div class="w-40 shrink-0 overflow-hidden">
                        <code class="block w-full text-xs text-muted-foreground bg-muted px-2 py-1 rounded truncate cursor-pointer hover:bg-muted/80 transition-colors tabular-nums"
                            @click="showIpInfo(log.ip)" :title="log.ip">{{ log.ip }}</code>
                    </div>
                    <span class="flex-1 min-w-0 text-xs text-muted-foreground truncate">
                        <TextOverflow :text="log.user_agent || '-'" title="User Agent" />
                    </span>
                    <span class="w-40 shrink-0 text-right text-xs text-muted-foreground tabular-nums opacity-60">{{ log.created_at }}</span>
                </div>
            </div>
            <!-- 分页 -->
            <Pagination :total="total" :page="currentPage" @update:page="handlePageChange" />
        </div>

        <!-- IP 地理位置弹窗 -->
        <Dialog v-model:open="ipDialogOpen">
            <DialogContent class="max-w-[90vw] sm:max-w-md">
                <DialogHeader>
                    <DialogTitle>IP 详情</DialogTitle>
                    <DialogDescription>
                        <code class="text-xs bg-muted px-2 py-0.5 rounded">{{ selectedIp }}</code>
                    </DialogDescription>
                </DialogHeader>
                <div v-if="ipGeoLoading" class="flex items-center justify-center py-8">
                    <Loader2 class="h-6 w-6 animate-spin text-muted-foreground" />
                </div>
                <div v-else-if="ipGeoInfo" class="grid grid-cols-[auto_1fr] gap-x-4 gap-y-2 text-xs sm:text-sm">
                    <div class="text-muted-foreground">国家</div>
                    <div class="font-medium">{{ ipGeoInfo.country }} ({{ ipGeoInfo.country_code }})</div>
                    <div class="text-muted-foreground">运营商</div>
                    <div class="font-medium truncate">{{ ipGeoInfo.isp || '-' }}</div>
                    <div class="text-muted-foreground">组织</div>
                    <div class="font-medium truncate">{{ ipGeoInfo.organization || '-' }}</div>
                    <div class="text-muted-foreground">ASN</div>
                    <div class="font-medium truncate">{{ ipGeoInfo.asn }} - {{ ipGeoInfo.asn_organization || '-' }}
                    </div>
                    <div class="text-muted-foreground">时区</div>
                    <div class="font-medium">{{ ipGeoInfo.timezone || '-' }}</div>
                    <div class="text-muted-foreground">坐标</div>
                    <div class="font-medium">{{ ipGeoInfo.latitude }}, {{ ipGeoInfo.longitude }}</div>
                </div>
                <div v-else class="text-center text-muted-foreground py-4">
                    无法获取 IP 信息
                </div>
            </DialogContent>
        </Dialog>
    </div>
</template>
