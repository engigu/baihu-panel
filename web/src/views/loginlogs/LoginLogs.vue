<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import Pagination from '@/components/Pagination.vue'
import { RefreshCw, Search } from 'lucide-vue-next'
import { api } from '@/api'
import { toast } from 'vue-sonner'
import { useSiteSettings } from '@/composables/useSiteSettings'

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

const logs = ref<LoginLog[]>([])
const filterUsername = ref('')
const currentPage = ref(1)
const total = ref(0)
const loading = ref(false)
let searchTimer: ReturnType<typeof setTimeout> | null = null

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
    <div class="flex items-center justify-between">
      <div>
        <h2 class="text-2xl font-bold tracking-tight">登录日志</h2>
        <p class="text-muted-foreground">查看系统登录记录</p>
      </div>
      <div class="flex items-center gap-2">
        <div class="relative">
          <Search class="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
          <Input
            v-model="filterUsername"
            placeholder="搜索用户名..."
            class="h-9 pl-9 w-56 text-sm"
            @input="handleSearch"
          />
        </div>
        <Button variant="outline" size="icon" class="h-9 w-9" @click="loadLogs" :disabled="loading">
          <RefreshCw class="h-4 w-4" :class="{ 'animate-spin': loading }" />
        </Button>
      </div>
    </div>

    <div class="rounded-lg border bg-card">
      <!-- 表头 -->
      <div class="flex items-center gap-4 px-4 py-2 border-b bg-muted/50 text-sm text-muted-foreground font-medium">
        <span class="w-24 shrink-0">用户名</span>
        <span class="w-32 shrink-0">IP 地址</span>
        <span class="w-16 shrink-0 text-center">状态</span>
        <span class="flex-1">User Agent</span>
        <span class="w-40 shrink-0 text-right">时间</span>
      </div>
      <!-- 列表 -->
      <div class="divide-y">
        <div v-if="logs.length === 0" class="text-sm text-muted-foreground text-center py-8">
          暂无登录日志
        </div>
        <div
          v-for="log in logs"
          :key="log.id"
          class="flex items-center gap-4 px-4 py-2 hover:bg-muted/50 transition-colors"
        >
          <span class="w-24 shrink-0 font-medium text-sm truncate">{{ log.username }}</span>
          <code class="w-32 shrink-0 text-xs text-muted-foreground bg-muted px-2 py-1 rounded">{{ log.ip }}</code>
          <span class="w-16 shrink-0 flex justify-center">
            <Badge :variant="log.status === 'success' ? 'default' : 'destructive'" class="text-xs">
              {{ log.status === 'success' ? '成功' : '失败' }}
            </Badge>
          </span>
          <span class="flex-1 text-xs text-muted-foreground truncate" :title="log.user_agent">{{ log.user_agent || '-' }}</span>
          <span class="w-40 shrink-0 text-right text-xs text-muted-foreground">{{ log.created_at }}</span>
        </div>
      </div>
      <!-- 分页 -->
      <Pagination :total="total" :page="currentPage" @update:page="handlePageChange" />
    </div>
  </div>
</template>
