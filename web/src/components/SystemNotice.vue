<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { Bell, Check } from 'lucide-vue-next'
import { Button } from '@/components/ui/button'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Badge } from '@/components/ui/badge'
import { api, type AppLog, LOG_CATEGORY, LOG_STATUS } from '@/api'
import { toast } from 'vue-sonner'
import { format } from 'date-fns'

const notices = ref<AppLog[]>([])
const unreadCount = ref(0)
const loading = ref(false)
const open = ref(false)

async function fetchNotices() {
  try {
    const res = await api.appLogs.list({
      category: LOG_CATEGORY.SYSTEM_NOTICE,
      status: LOG_STATUS.UNREAD,
      page: 1,
      page_size: 50
    })
    notices.value = res.data || []
    unreadCount.value = notices.value.length
  } catch (error: any) {
    console.error('Failed to fetch notices:', error)
  }
}

async function markAsRead(id: string) {
  try {
    await api.appLogs.markAsRead({ id })
    notices.value = notices.value.filter((n: AppLog) => n.id !== id)
    unreadCount.value = notices.value.length
  } catch (error: any) {
    toast.error(error.message || '标记已读失败')
  }
}

async function markAllAsRead() {
  if (notices.value.length === 0) return
  loading.value = true
  try {
    await api.appLogs.markAsRead({ category: LOG_CATEGORY.SYSTEM_NOTICE })
    notices.value = []
    unreadCount.value = 0
    toast.success('已清空未读消息')
    open.value = false
  } catch (error: any) {
    toast.error(error.message || '全部已读失败')
  } finally {
    loading.value = false
  }
}

let timer: number
onMounted(() => {
  fetchNotices()
  timer = window.setInterval(fetchNotices, 60000) // 每分钟拉取一次
})

onUnmounted(() => {
  if (timer) clearInterval(timer)
})

function formatDate(dateStr: string) {
  if (!dateStr) return ''
  try {
    return format(new Date(dateStr), 'MM-dd HH:mm')
  } catch {
    return dateStr
  }
}

function getLevelColor(level: string) {
  switch (level) {
    case 'error': return 'destructive'
    case 'warning': return 'warning'
    default: return 'secondary'
  }
}
</script>

<template>
  <Popover v-model:open="open">
    <PopoverTrigger asChild>
      <Button variant="ghost" size="icon" class="relative">
        <Bell class="h-5 w-5" />
        <span v-if="unreadCount > 0"
          class="absolute top-1 right-1 h-2 w-2 rounded-full bg-red-500 animate-pulse"></span>
      </Button>
    </PopoverTrigger>
    <PopoverContent class="w-80 p-0" align="end">
      <div class="flex items-center justify-between px-4 py-3 border-b">
        <span class="font-medium text-sm">系统通知 <span class="text-xs text-muted-foreground ml-1" v-if="unreadCount">({{
            unreadCount }})</span></span>
        <Button variant="ghost" size="sm" class="h-auto p-0 text-xs text-muted-foreground hover:text-foreground"
          :disabled="loading || unreadCount === 0" @click="markAllAsRead">
          <Check class="h-3 w-3 mr-1" />
          全标已读
        </Button>
      </div>

      <ScrollArea class="h-[300px]" v-if="notices.length > 0">
        <div class="flex flex-col">
          <div v-for="notice in notices" :key="notice.id"
            class="p-4 border-b last:border-0 hover:bg-muted/50 transition-colors group relative">
            <div class="flex items-start justify-between gap-2 mb-1 cursor-pointer" @click="markAsRead(notice.id)">
              <div class="flex-1 min-w-0">
                <div class="flex items-center gap-2 mb-1">
                  <Badge :variant="getLevelColor(notice.level) as any" class="px-1.5 py-0 text-[10px]">{{ notice.level
                    || 'info' }}</Badge>
                  <span class="text-sm font-medium truncate">{{ notice.title }}</span>
                </div>
                <p class="text-xs text-muted-foreground line-clamp-2">{{ notice.content }}</p>
                <p class="text-[10px] text-muted-foreground mt-1">{{ formatDate(notice.created_at) }}</p>
              </div>
            </div>
          </div>
        </div>
      </ScrollArea>
      <div v-else class="py-8 text-center text-sm text-muted-foreground flex flex-col items-center">
        <Bell class="h-8 w-8 text-muted mb-2 opacity-20" />
        暂无新通知
      </div>
    </PopoverContent>
  </Popover>
</template>
