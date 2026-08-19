<script setup lang="ts">
// 消息日志总览入口控制器
import { ref, watch } from 'vue'
import { Input } from '@/components/ui/input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Search, RefreshCw, Trash2, Terminal, Cpu, Send, KeyRound } from 'lucide-vue-next'
import LoginLogTab from './tabs/LoginLogTab.vue'
import SystemEventTab from './tabs/SystemEventTab.vue'
import PushLogTab from './tabs/PushLogTab.vue'
import SchedulerLogTab from './tabs/SchedulerLogTab.vue'
import FilterLogTab from './tabs/FilterLogTab.vue'
import { LOG_LEVEL, LOG_STATUS } from '@/api'
import { ShieldAlert } from 'lucide-vue-next'

const activeTab = ref(localStorage.getItem('baihu_active_log_tab') || 'system')
const systemTabRef = ref()
const pushLogRef = ref()
const loginTabRef = ref()
const schedulerTabRef = ref()
const filterTabRef = ref()

const filters = ref({
  system: { keyword: '', level: 'all' },
  push: { keyword: '', status: 'all' },
  login: { username: '' },
  scheduler: { keyword: '', level: 'all' },
  filter: { keyword: '', level: 'all' }
})

let searchTimer: ReturnType<typeof setTimeout> | null = null

function handleSearch() {
  if (searchTimer) clearTimeout(searchTimer)
  searchTimer = setTimeout(() => {
    handleRefresh()
  }, 300)
}

const isRefreshing = ref(false)

async function handleRefresh() {
  if (isRefreshing.value) return
  isRefreshing.value = true
  try {
    if (activeTab.value === 'system') await systemTabRef.value?.fetchLogs()
    else if (activeTab.value === 'push') await pushLogRef.value?.fetchLogs()
    else if (activeTab.value === 'login') await loginTabRef.value?.loadLogs()
    else if (activeTab.value === 'scheduler') await schedulerTabRef.value?.fetchLogs()
    else if (activeTab.value === 'filter') await filterTabRef.value?.fetchLogs()
  } finally {
    setTimeout(() => {
      isRefreshing.value = false
    }, 400)
  }
}

function handleClear() {
  if (activeTab.value === 'system' && systemTabRef.value) systemTabRef.value.showClearConfirm = true
  else if (activeTab.value === 'push' && pushLogRef.value) pushLogRef.value.showClearConfirm = true
  else if (activeTab.value === 'scheduler' && schedulerTabRef.value) schedulerTabRef.value.showClearConfirm = true
  else if (activeTab.value === 'filter' && filterTabRef.value) filterTabRef.value.showClearConfirm = true
}

// 切换标签时保存到 localStorage
watch(activeTab, (newTab) => {
  localStorage.setItem('baihu_active_log_tab', newTab)
})

</script>

<template>
  <div class="space-y-6 h-full flex flex-col">
    <div class="flex flex-col lg:flex-row lg:items-center justify-between gap-4 shrink-0 px-1">
      <div class="flex flex-col shrink-0">
        <h2 class="text-xl sm:text-2xl font-bold tracking-tight">运行日志</h2>
        <p class="text-muted-foreground text-sm">
          {{ activeTab === 'system' ? '查看系统重要运行事件' :
            activeTab === 'push' ? '查看消息推送历史记录' : 
            activeTab === 'scheduler' ? '查看后台调度器执行与配置装载日志' : 
            activeTab === 'filter' ? '查看通知被规则匹配过滤/拦截的日志' : '查看系统用户登录记录' }}
        </p>
      </div>

      <!-- 统一的响应式父容器：小屏下垂直两行，大屏下横向并排 -->
      <div class="flex flex-col lg:flex-row lg:items-center justify-between gap-2.5 w-full lg:w-auto lg:ml-auto">
        
        <!-- 第一行（小屏）/ 左侧（大屏）：搜索 + 级别/状态过滤 -->
        <div class="flex items-center gap-2 w-full lg:w-auto">
          <!-- 搜索输入框：手机端弹性占据剩余空间，平板及大屏固定宽度 -->
          <div v-if="activeTab !== 'login'" class="relative flex-1 sm:w-56 lg:w-60 group">
            <Search class="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground group-focus-within:text-primary transition-colors" />
            <Input 
              v-if="activeTab === 'system'"
              v-model="filters.system.keyword" 
              placeholder="搜索系统事件..." 
              class="h-9 pl-9 w-full bg-muted/20 border-muted-foreground/10 focus:bg-background text-sm"
              @input="handleSearch" 
            />
            <Input 
              v-else-if="activeTab === 'push'"
              v-model="filters.push.keyword" 
              placeholder="搜索推送日志..." 
              class="h-9 pl-9 w-full bg-muted/20 border-muted-foreground/10 focus:bg-background text-sm"
              @input="handleSearch" 
            />
            <Input 
              v-else-if="activeTab === 'scheduler'"
              v-model="filters.scheduler.keyword" 
              placeholder="搜索调度日志..." 
              class="h-9 pl-9 w-full bg-muted/20 border-muted-foreground/10 focus:bg-background text-sm"
              @input="handleSearch" 
            />
            <Input 
              v-else-if="activeTab === 'filter'"
              v-model="filters.filter.keyword" 
              placeholder="搜索过滤日志..." 
              class="h-9 pl-9 w-full bg-muted/20 border-muted-foreground/10 focus:bg-background text-sm"
              @input="handleSearch" 
            />
          </div>
          <div v-else class="relative flex-1 sm:w-48 group">
            <Search class="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground group-focus-within:text-primary transition-colors" />
            <Input 
              v-model="filters.login.username" 
              placeholder="搜索用户..." 
              class="h-9 pl-9 w-full bg-muted/20 border-muted-foreground/10 focus:bg-background text-sm"
              @input="handleSearch" 
            />
          </div>

          <!-- 系统事件 级别筛选 -->
          <div v-if="activeTab === 'system'" class="relative w-[112px] sm:w-28 shrink-0">
            <Select v-model="filters.system.level" @update:model-value="handleRefresh">
              <SelectTrigger class="h-9 w-full text-xs sm:text-sm bg-muted/20 border-muted-foreground/10">
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

          <!-- 过滤日志 级别筛选 -->
          <div v-if="activeTab === 'filter'" class="relative w-[112px] sm:w-28 shrink-0">
            <Select v-model="filters.filter.level" @update:model-value="handleRefresh">
              <SelectTrigger class="h-9 w-full text-xs sm:text-sm bg-muted/20 border-muted-foreground/10">
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

          <!-- 推送日志 状态筛选 -->
          <div v-if="activeTab === 'push'" class="relative w-[112px] sm:w-28 shrink-0">
            <Select v-model="filters.push.status" @update:model-value="handleRefresh">
              <SelectTrigger class="h-9 w-full text-xs sm:text-sm bg-muted/20 border-muted-foreground/10">
                <SelectValue placeholder="状态" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">所有状态</SelectItem>
                <SelectItem :value="LOG_STATUS.SUCCESS">发送成功</SelectItem>
                <SelectItem :value="LOG_STATUS.FAILED">发送失败</SelectItem>
              </SelectContent>
            </Select>
          </div>
        </div>

        <!-- 第二行（小屏）/ 右侧（大屏）：切换日志视图 + 纯图标功能按纽 -->
        <div class="flex items-center gap-2 w-full lg:w-auto shrink-0">
          
          <!-- 极简高雅的下拉菜单组合切换器 (手机端弹性充满，大屏下 w-[120px] 紧凑) -->
          <div class="flex-1 lg:w-[125px] shrink-0">
            <Select v-model="activeTab">
              <SelectTrigger class="w-full h-9 bg-background/50 hover:bg-background/80 border-border/60 hover:border-border transition-all shadow-sm focus:ring-0 focus:ring-offset-0 text-xs font-semibold px-3 gap-2">
                <template v-if="activeTab === 'system'">
                  <Terminal class="w-3.5 h-3.5 text-primary" />
                  <span>系统日志</span>
                </template>
                <template v-else-if="activeTab === 'scheduler'">
                  <Cpu class="w-3.5 h-3.5 text-primary" />
                  <span>调度日志</span>
                </template>
                <template v-else-if="activeTab === 'push'">
                  <Send class="w-3.5 h-3.5 text-primary" />
                  <span>推送日志</span>
                </template>
                <template v-else-if="activeTab === 'filter'">
                  <ShieldAlert class="w-3.5 h-3.5 text-primary" />
                  <span>过滤日志</span>
                </template>
                <template v-else-if="activeTab === 'login'">
                  <KeyRound class="w-3.5 h-3.5 text-primary" />
                  <span>登录日志</span>
                </template>
              </SelectTrigger>
              <SelectContent class="min-w-[120px] shadow-xl backdrop-blur-md bg-popover/95 border-border/40">
                <SelectItem value="system" class="text-xs font-medium focus:bg-accent/80 cursor-pointer">
                  <div class="flex items-center gap-2">
                    <Terminal class="w-3.5 h-3.5 text-muted-foreground/80" />
                    <span>系统日志</span>
                  </div>
                </SelectItem>
                <SelectItem value="scheduler" class="text-xs font-medium focus:bg-accent/80 cursor-pointer">
                  <div class="flex items-center gap-2">
                    <Cpu class="w-3.5 h-3.5 text-muted-foreground/80" />
                    <span>调度日志</span>
                  </div>
                </SelectItem>
                <SelectItem value="push" class="text-xs font-medium focus:bg-accent/80 cursor-pointer">
                  <div class="flex items-center gap-2">
                    <Send class="w-3.5 h-3.5 text-muted-foreground/80" />
                    <span>推送日志</span>
                  </div>
                </SelectItem>
                <SelectItem value="filter" class="text-xs font-medium focus:bg-accent/80 cursor-pointer">
                  <div class="flex items-center gap-2">
                    <ShieldAlert class="w-3.5 h-3.5 text-muted-foreground/80" />
                    <span>过滤日志</span>
                  </div>
                </SelectItem>
                <SelectItem value="login" class="text-xs font-medium focus:bg-accent/80 cursor-pointer">
                  <div class="flex items-center gap-2">
                    <KeyRound class="w-3.5 h-3.5 text-muted-foreground/80" />
                    <span>登录日志</span>
                  </div>
                </SelectItem>
              </SelectContent>
            </Select>
          </div>

          <!-- 功能按钮组合：保持 h-9 w-9 纯正方图标按钮，精致利落 -->
          <div class="flex items-center gap-2 shrink-0">
            <button type="button" class="inline-flex items-center justify-center h-9 w-9 rounded-md border border-border bg-background hover:bg-accent hover:text-accent-foreground shadow-sm transition-all cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed" :disabled="isRefreshing" @click="handleRefresh" title="刷新数据">
              <RefreshCw class="h-4 w-4 block" :class="{ 'animate-spin': isRefreshing }" />
            </button>

            <button v-if="activeTab !== 'login'" type="button" class="inline-flex items-center justify-center h-9 w-9 rounded-md border border-destructive/20 bg-background text-destructive hover:bg-destructive/10 shadow-sm transition-colors cursor-pointer" @click="handleClear" title="清空记录">
              <Trash2 class="h-4 w-4 block" />
            </button>
          </div>
        </div>

      </div>
    </div>

    <div class="flex-1 min-h-0">
      <div v-show="activeTab === 'system'" class="h-full">
        <SystemEventTab ref="systemTabRef" :filters="filters.system" />
      </div>

      <div v-show="activeTab === 'scheduler'" class="h-full">
        <SchedulerLogTab ref="schedulerTabRef" :filters="filters.scheduler" />
      </div>

      <div v-show="activeTab === 'push'" class="h-full">
        <PushLogTab ref="pushLogRef" :filters="filters.push" />
      </div>

      <div v-show="activeTab === 'filter'" class="h-full">
        <FilterLogTab ref="filterTabRef" :filters="filters.filter" />
      </div>

      <div v-show="activeTab === 'login'" class="h-full">
        <LoginLogTab ref="loginTabRef" :username="filters.login.username" />
      </div>
    </div>
  </div>
</template>
