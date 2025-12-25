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
  DialogTitle
} from '@/components/ui/dialog'

const { pageSize } = useSiteSettings()

interface LoginLog {
  id: number
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
  <div class="space-y-6">
    <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
      <div>
        <h2 class="text-xl sm:text-2xl font-bold tracking-tight">登录日志</h2>
        <p class="text-muted-foreground text-sm">查看系统登录记录</p>
      </div>
      <div class="flex items-center gap-2">
        <div class="relative flex-1 sm:flex-none">
          <Search class="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
          <Input
            v-model="filterUsername"
            placeholder="搜索用户名..."
            class="h-9 pl-9 w-full sm:w-56 text-sm"
            @input="handleSearch"
          />
        </div>
        <Button variant="outline" size="icon" class="h-9 w-9 shrink-0" @click="loadLogs" :disabled="loading">
          <RefreshCw class="h-4 w-4" :class="{ 'animate-spin': loading }" />
        </Button>
      </div>
    </div>

    <div class="rounded-lg border bg-card overflow-x-auto">
      <!-- 表头 -->
      <div class="flex items-center gap-2 sm:gap-4 px-3 sm:px-4 py-2 border-b bg-muted/50 text-xs sm:text-sm text-muted-foreground font-medium sm:min-w-[500px]">
        <span class="w-16 sm:w-24 shrink-0">用户名</span>
        <span class="w-20 sm:w-32 shrink-0">IP 地址</span>
        <span class="w-10 sm:w-16 shrink-0 text-center">状态</span>
        <span class="hidden sm:flex sm:flex-1">User Agent</span>
        <span class="shrink-0 sm:w-40 sm:text-right">时间</span>
      </div>
      <!-- 列表 -->
      <div class="divide-y sm:min-w-[500px]">
        <div v-if="logs.length === 0" class="text-sm text-muted-foreground text-center py-8">
          暂无登录日志
        </div>
        <div
          v-for="log in logs"
          :key="log.id"
          class="flex items-center gap-2 sm:gap-4 px-3 sm:px-4 py-2 hover:bg-muted/50 transition-colors"
        >
          <span class="w-16 sm:w-24 shrink-0 font-medium text-xs sm:text-sm truncate">{{ log.username }}</span>
          <code 
            class="w-20 sm:w-32 shrink-0 text-xs text-muted-foreground bg-muted px-1 sm:px-2 py-0.5 sm:py-1 rounded truncate cursor-pointer hover:bg-muted/80 transition-colors"
            @click="showIpInfo(log.ip)"
          >{{ log.ip }}</code>
          <span class="w-10 sm:w-16 shrink-0 flex justify-center">
            <span :class="['h-2 w-2 rounded-full', log.status === 'success' ? 'bg-green-500' : 'bg-red-500']"></span>
          </span>
          <span class="hidden sm:flex sm:flex-1 text-xs text-muted-foreground truncate">
            <TextOverflow :text="log.user_agent || '-'" title="User Agent" />
          </span>
          <span class="shrink-0 sm:w-40 sm:text-right text-xs text-muted-foreground">{{ log.created_at }}</span>
        </div>
      </div>
      <!-- 分页 -->
      <Pagination :total="total" :page="currentPage" @update:page="handlePageChange" />
    </div>

    <!-- IP 地理位置弹窗 -->
    <Dialog v-model:open="ipDialogOpen">
      <DialogContent class="max-w-[90vw] sm:max-w-md">
        <DialogHeader>
          <DialogTitle class="text-base sm:text-lg">IP 信息 - {{ selectedIp }}</DialogTitle>
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
          <div class="font-medium truncate">{{ ipGeoInfo.asn }} - {{ ipGeoInfo.asn_organization || '-' }}</div>
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
