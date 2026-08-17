<script setup lang="ts">
// 消息日志总览入口控制器
import { ref, watch } from 'vue'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Input } from '@/components/ui/input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Search, RefreshCw, Trash2, Terminal, Cpu, Send, KeyRound } from 'lucide-vue-next'
import LoginLogTab from './tabs/LoginLogTab.vue'
import SystemEventTab from './tabs/SystemEventTab.vue'
import PushLogTab from './tabs/PushLogTab.vue'
import SchedulerLogTab from './tabs/SchedulerLogTab.vue'
import { LOG_LEVEL, LOG_STATUS } from '@/api'

const activeTab = ref('system')
const systemTabRef = ref()
const pushLogRef = ref()
const loginTabRef = ref()
const schedulerTabRef = ref()

const filters = ref({
  system: { keyword: '', level: 'all' },
  push: { keyword: '', status: 'all' },
  login: { username: '' },
  scheduler: { keyword: '', level: 'all' }
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
}

// 切换标签时重置搜索
watch(activeTab, () => {
  // handleRefresh()
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
            activeTab === 'scheduler' ? '查看后台调度器执行与配置装载日志' : '查看系统用户登录记录' }}
        </p>
      </div>

      <div :class="[activeTab === 'login' ? 'flex flex-row lg:flex-row' : 'flex flex-col lg:flex-row', 'lg:items-center gap-2 lg:gap-3 w-full lg:w-auto lg:ml-auto lg:justify-end']">
        <!-- 搜索与筛选区域 -->
        <div :class="[activeTab === 'login' ? 'flex-1 min-w-0' : 'w-full lg:w-auto', 'flex items-center gap-2']">
          <!-- 系统事件 / 推送日志 / 调度日志 搜索框 -->
          <div v-if="activeTab !== 'login'" class="relative flex-1 lg:w-60 group">
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
          </div>
          <!-- 登录日志 搜索框 -->
          <div v-else class="relative flex-1 lg:w-48 group">
            <Search class="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground group-focus-within:text-primary transition-colors" />
            <Input 
              v-model="filters.login.username" 
              placeholder="搜索..." 
              class="h-9 pl-9 w-full bg-muted/20 border-muted-foreground/10 focus:bg-background text-sm"
              @input="handleSearch" 
            />
          </div>

          <!-- 系统事件 级别筛选 -->
          <div v-if="activeTab === 'system'" class="relative w-24 sm:w-28 shrink-0">
            <Select v-model="filters.system.level" @update:model-value="handleRefresh">
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

          <!-- 推送日志 状态筛选 -->
          <div v-if="activeTab === 'push'" class="relative w-24 sm:w-28 shrink-0">
            <Select v-model="filters.push.status" @update:model-value="handleRefresh">
              <SelectTrigger class="h-9 w-full text-sm bg-muted/20 border-muted-foreground/10">
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

        <!-- 操作区域 -->
        <div :class="[activeTab === 'login' ? 'flex-1 sm:flex-none' : 'w-full lg:w-auto', 'flex items-center gap-2']">
          <button type="button" class="inline-flex items-center justify-center h-9 w-9 rounded-md border border-border bg-background hover:bg-accent hover:text-accent-foreground shrink-0 shadow-sm transition-all cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed" :disabled="isRefreshing" @click="handleRefresh">
            <RefreshCw class="h-4 w-4 block transition-transform" :class="{ 'animate-spin': isRefreshing }" />
          </button>

          <button v-if="activeTab !== 'login'" type="button" class="inline-flex items-center justify-center h-9 px-3 rounded-md border border-destructive/20 bg-background text-destructive hover:bg-destructive/10 shrink-0 shadow-sm transition-colors gap-1.5 cursor-pointer" @click="handleClear">
            <Trash2 class="h-4 w-4 block" />
            <span class="text-sm font-medium" :class="activeTab !== 'login' ? '' : 'hidden'">清空记录</span>
          </button>

          <!-- 桌面端标签切换 -->
          <Tabs v-model="activeTab" class="w-auto">
            <TabsList class="h-9 p-0.5 bg-muted/20 border border-border/40 shrink-0 hidden lg:flex rounded-lg">
              <TabsTrigger value="system" class="px-3 h-8 text-xs gap-1.5 font-medium transition-all">
                <Terminal class="w-3.5 h-3.5 opacity-70" />
                <span>系统</span>
              </TabsTrigger>
              <TabsTrigger value="scheduler" class="px-3 h-8 text-xs gap-1.5 font-medium transition-all">
                <Cpu class="w-3.5 h-3.5 opacity-70" />
                <span>调度</span>
              </TabsTrigger>
              <TabsTrigger value="push" class="px-3 h-8 text-xs gap-1.5 font-medium transition-all">
                <Send class="w-3.5 h-3.5 opacity-70" />
                <span>推送</span>
              </TabsTrigger>
              <TabsTrigger value="login" class="px-3 h-8 text-xs gap-1.5 font-medium transition-all">
                <KeyRound class="w-3.5 h-3.5 opacity-70" />
                <span>登录</span>
              </TabsTrigger>
            </TabsList>
          </Tabs>

          <!-- 移动端标签切换 (简易版) -->
          <div class="lg:hidden flex-1 shrink-0 min-w-0">
            <Select v-model="activeTab">
              <SelectTrigger class="h-9 w-full text-sm bg-muted/20 border-muted-foreground/10">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="system">系统事件</SelectItem>
                <SelectItem value="scheduler">调度日志</SelectItem>
                <SelectItem value="push">推送日志</SelectItem>
                <SelectItem value="login">登录日志</SelectItem>
              </SelectContent>
            </Select>
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

      <div v-show="activeTab === 'login'" class="h-full">
        <LoginLogTab ref="loginTabRef" :username="filters.login.username" />
      </div>
    </div>
  </div>
</template>
