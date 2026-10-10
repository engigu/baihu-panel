<script setup lang="ts">
import { ref, watch, onMounted, onUnmounted, computed } from 'vue'
import type { MonitorStats } from '@/api'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { RefreshCw, Cpu, MemoryStick, HardDrive, Activity, LayoutDashboard, Layers, Play, Server, CalendarClock } from 'lucide-vue-next'
import { formatDateTime } from '@/utils/date'

import {
  Chart as ChartJS,
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  BarElement,
  Title,
  Tooltip,
  Legend,
  Filler
} from 'chart.js'
import { Line, Bar } from 'vue-chartjs'

ChartJS.register(
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  BarElement,
  Title,
  Tooltip,
  Legend,
  Filler
)
ChartJS.defaults.font.family = "'Inter', 'Inter Variable', -apple-system, BlinkMacSystemFont, sans-serif"

const activeTab = ref(localStorage.getItem('monitor_active_tab') || 'charts')

watch(activeTab, (newVal) => {
  localStorage.setItem('monitor_active_tab', newVal)
})

const stats = ref<MonitorStats | null>(null)
const loading = ref(false)

const busyWorkersCount = computed(() => {
  if (!stats.value?.scheduler?.workers) return 0
  return stats.value.scheduler.workers.filter(w => w.status === 'running').length
})

const totalWorkersCount = computed(() => {
  return stats.value?.scheduler?.workers?.length || stats.value?.scheduler?.worker_count || 0
})

// --- 时序数据池 ---
const historySize = 60 // 保存最近60次请求（约3分钟@3s）
const timeLabels = ref<string[]>([])
const goroutinesData = ref<number[]>([])
const allocData = ref<number[]>([])
const sysData = ref<number[]>([])
const gcPausesData = ref<number[]>([])
const scheduledData = ref<number[]>([])
const runningData = ref<number[]>([])
const queueData = ref<number[]>([])
let lastPauseNs = 0

let ws: EventSource | null = null

const protocol = window.location.protocol === 'https:' ? 'https:' : 'http:'
const baseUrl = (window as any).__BASE_URL__ || ''
const apiVersion = (window as any).__API_VERSION__ || '/api/v1'

const connectWS = () => {
  if (ws) return
  loading.value = true
  const host = window.location.host
  const sseUrl = `${protocol}//${host}${baseUrl}${apiVersion}/monitor/sse`

  ws = new EventSource(sseUrl, { withCredentials: true })
  
  ws.onopen = () => {
    loading.value = false
  }

  ws.onmessage = (event: MessageEvent) => {
    try {
      const payload = JSON.parse(event.data)
      if (payload.code === 200 && payload.data) {
        const res = payload.data
        stats.value = res

        const nowStr = new Date().toLocaleTimeString('en-US', { hour12: false })
        if (timeLabels.value.length >= historySize) {
          timeLabels.value.shift()
          goroutinesData.value.shift()
          allocData.value.shift()
          sysData.value.shift()
          gcPausesData.value.shift()
          scheduledData.value.shift()
          runningData.value.shift()
          queueData.value.shift()
        }
        
        timeLabels.value.push(nowStr)
        goroutinesData.value.push(res.env.goroutines)
        allocData.value.push(Number((res.mem.alloc / 1024 / 1024).toFixed(2)))
        sysData.value.push(Number((res.mem.sys / 1024 / 1024).toFixed(2)))
        
        let pauseDelta = 0
        if (lastPauseNs > 0 && res.gc.pause_total_ns >= lastPauseNs) {
          pauseDelta = Number(((res.gc.pause_total_ns - lastPauseNs) / 1000000).toFixed(2))
        }
        lastPauseNs = res.gc.pause_total_ns
        gcPausesData.value.push(pauseDelta)
        scheduledData.value.push(res.scheduler.scheduled)
        runningData.value.push(res.scheduler.running)
        queueData.value.push(res.scheduler.queue_size)
      }
    } catch (e) {
      console.error('Parse WS message error:', e)
    }
  }
  
  ws.onerror = () => {
    loading.value = false
    // EventSource 会自动指数退避重连，无需我们手动控制
  }
}

const disconnectWS = () => {
  if (ws) {
    ws.close()
    ws = null
  }
}

const formatBytes = (bytes: number) => {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

const formatNs = (ns: number) => {
  if (ns < 1000) return ns + ' ns'
  if (ns < 1000000) return (ns / 1000).toFixed(2) + ' μs'
  return (ns / 1000000).toFixed(2) + ' ms'
}

const memPercent = computed(() => {
  if (!stats.value) return 0
  if (stats.value.container?.is_docker && stats.value.container.limit > 0) {
    return stats.value.container.limit_percent || 0
  }
  return stats.value.host?.mem_percent || 0
})

onMounted(() => {
  connectWS()
})

onUnmounted(() => {
  disconnectWS()
})

// --- 图表配置 ---
const chartOptions = {
  responsive: true,
  maintainAspectRatio: false,
  animation: {
    duration: 0 // 关闭过渡动画以支持实时流畅更新
  },
  interaction: {
    mode: 'index' as const,
    intersect: false,
  },
  plugins: {
    legend: {
      position: 'top' as const,
      labels: {
        usePointStyle: false,
        boxWidth: 12,
        boxHeight: 12,
        useBorderRadius: true,
        borderRadius: 2
      }
    }
  },
  scales: {
    x: {
      grid: { display: false },
      ticks: {
        maxTicksLimit: 8, // 限制 X 轴显示的标签数量，避免拥挤
        maxRotation: 0,   // 禁止标签旋转，保持水平整洁
        color: 'rgba(156, 163, 175, 0.8)', // 调整颜色更柔和 (gray-400)
        font: { size: 11 }
      }
    },
    y: {
      beginAtZero: true,
      border: { dash: [4, 4] },
      grid: { color: 'rgba(156, 163, 175, 0.1)' },
      ticks: {
        color: 'rgba(156, 163, 175, 0.8)',
        font: { size: 11 }
      }
    }
  }
}

const goroutineChartData = computed(() => ({
  labels: [...timeLabels.value],
  datasets: [
    {
      label: 'Goroutines(协程数)',
      backgroundColor: 'rgba(37, 99, 235, 0.1)',
      borderColor: '#2563eb', // blue-600
      borderWidth: 2,
      data: [...goroutinesData.value],
      tension: 0.4,
      fill: true,
      pointRadius: 0,
      pointHitRadius: 10
    }
  ]
}))

const memChartData = computed(() => ({
  labels: [...timeLabels.value],
  datasets: [
    {
      label: 'Alloc(MB) 当前分配',
      backgroundColor: 'rgba(5, 150, 105, 0.1)',
      borderColor: '#059669', // emerald-600
      borderWidth: 2,
      data: [...allocData.value],
      tension: 0.4,
      fill: true,
      pointRadius: 0,
      pointHitRadius: 10
    },
    {
      label: 'Sys(MB) 系统申请上限',
      backgroundColor: 'transparent',
      borderColor: '#d97706', // amber-600
      borderWidth: 2,
      borderDash: [5, 5],
      data: [...sysData.value],
      tension: 0.4,
      pointRadius: 0,
      pointHitRadius: 10
    }
  ]
}))

const gcChartData = computed(() => ({
  labels: [...timeLabels.value],
  datasets: [
    {
      label: 'GC Pause(ms) 垃圾回收停顿',
      backgroundColor: '#ea580c', // orange-600
      data: [...gcPausesData.value],
      borderRadius: 4
    }
  ]
}))

const schedulerChartData = computed(() => ({
  labels: [...timeLabels.value],
  datasets: [
    {
      label: '正在运行 (Running)',
      backgroundColor: 'rgba(16, 185, 129, 0.1)', // emerald-500
      borderColor: '#10b981', 
      borderWidth: 2,
      data: [...runningData.value],
      tension: 0.4,
      fill: true,
      pointRadius: 0,
      pointHitRadius: 10
    },
    {
      label: '调度中 (Scheduled)',
      backgroundColor: 'transparent',
      borderColor: '#6366f1', // indigo-500
      borderWidth: 2,
      borderDash: [5, 5],
      data: [...scheduledData.value],
      tension: 0.4,
      pointRadius: 0,
      pointHitRadius: 10
    },
    {
      label: '排队积压 (Queue)',
      backgroundColor: 'transparent',
      borderColor: '#f59e0b', // amber-500
      borderWidth: 2,
      borderDash: [2, 2],
      data: [...queueData.value],
      tension: 0.4,
      pointRadius: 0,
      pointHitRadius: 10
    }
  ]
}))
</script>

<template>
  <Tabs v-model="activeTab" class="space-y-6 w-full">
    <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
      <div>
        <h2 class="text-xl sm:text-2xl font-bold tracking-tight">系统监控</h2>
        <p class="text-muted-foreground text-sm">实时监控面板资源、内存分配和垃圾回收状态</p>
      </div>
      <div class="flex items-center gap-2 w-full sm:w-auto">
        <TabsList class="h-9 p-0.5 bg-muted/20 border border-border/40 rounded-lg w-full sm:w-[240px] flex">
          <TabsTrigger value="charts" class="px-3 h-8 text-xs gap-1.5 font-medium transition-all flex-1">
            <Activity class="w-3.5 h-3.5 opacity-70" />
            <span>实时图表</span>
          </TabsTrigger>
          <TabsTrigger value="dashboard" class="px-3 h-8 text-xs gap-1.5 font-medium transition-all flex-1">
            <LayoutDashboard class="w-3.5 h-3.5 opacity-70" />
            <span>数据视图</span>
          </TabsTrigger>
        </TabsList>
        <Button variant="outline" size="icon" class="h-9 w-9 shrink-0" @click="() => { disconnectWS(); connectWS() }" :disabled="loading" title="刷新并重连">
          <RefreshCw class="w-4 h-4" :class="{ 'animate-spin': loading }" />
        </Button>
      </div>
    </div>

    <TabsContent value="dashboard" class="mt-0 space-y-4">
      


      <!-- 物理主机与容器资源监控环形图 -->
      <div v-if="stats" class="grid grid-cols-1 md:grid-cols-3 gap-4 mt-4">
        <!-- CPU 使用率 -->
        <Card class="border-border/60">
          <CardHeader class="pb-2">
            <div class="flex items-center justify-between">
              <CardTitle class="text-sm font-semibold text-foreground flex items-center gap-2">
                <span class="p-1 rounded-md bg-blue-500/10 text-blue-500">
                  <Cpu class="w-3.5 h-3.5" />
                </span>
                CPU 使用率
              </CardTitle>
              <span class="text-[10px] px-1.5 py-0.5 rounded border border-border/60 text-muted-foreground bg-muted/20 font-medium tracking-wide">
                host cpu
              </span>
            </div>
          </CardHeader>
          <CardContent class="flex flex-col items-center pt-1">
            <div class="relative w-32 h-32 flex items-center justify-center mt-1">
              <svg class="w-full h-full transform -rotate-90 drop-shadow-sm" viewBox="0 0 36 36">
                <defs>
                  <linearGradient id="cpuGrad" x1="0%" y1="0%" x2="100%" y2="100%">
                    <stop offset="0%" stop-color="#38bdf8" />
                    <stop offset="100%" stop-color="#3b82f6" />
                  </linearGradient>
                  <linearGradient id="cpuGradWarn" x1="0%" y1="0%" x2="100%" y2="100%">
                    <stop offset="0%" stop-color="#fb7185" />
                    <stop offset="100%" stop-color="#e11d48" />
                  </linearGradient>
                </defs>
                <path class="text-muted/20" stroke-width="2.8" stroke="currentColor" fill="none" d="M18 2.0845 a 15.9155 15.9155 0 0 1 0 31.831 a 15.9155 15.9155 0 0 1 0 -31.831" />
                <path
                  :stroke="stats.host.cpu_percent > 80 ? 'url(#cpuGradWarn)' : 'url(#cpuGrad)'"
                  stroke-dasharray="100, 100"
                  :stroke-dashoffset="100 - stats.host.cpu_percent"
                  stroke-width="2.8"
                  stroke-linecap="round"
                  fill="none"
                  d="M18 2.0845 a 15.9155 15.9155 0 0 1 0 31.831 a 15.9155 15.9155 0 0 1 0 -31.831"
                  style="transition: stroke-dashoffset 0.6s ease;"
                />
              </svg>
              <div class="absolute flex flex-col items-center justify-center">
                <div class="flex items-baseline" :class="stats.host.cpu_percent > 80 ? 'text-red-500' : 'text-foreground'">
                  <span class="text-2xl font-extrabold tracking-tight tabular-nums metric-num">{{ stats.host.cpu_percent.toFixed(1) }}</span>
                  <span class="text-xs font-semibold text-muted-foreground ml-0.5">%</span>
                </div>
              </div>
            </div>
            <div class="w-full mt-3 p-2 rounded-lg bg-muted/20 border border-border/50 flex items-center justify-between text-xs">
              <span class="text-muted-foreground">核心状态</span>
              <span class="font-medium text-foreground flex items-center gap-1.5">
                <span class="w-1.5 h-1.5 rounded-full" :class="stats.host.cpu_percent > 80 ? 'bg-red-500 animate-pulse' : 'bg-blue-500'"></span>
                {{ stats.host.cpu_percent > 80 ? '高负载' : '负载正常' }}
              </span>
            </div>
          </CardContent>
        </Card>
        
        <!-- 内存使用率 -->
        <Card class="border-border/60">
          <CardHeader class="pb-2">
            <div class="flex items-center justify-between">
              <CardTitle class="text-sm font-semibold text-foreground flex items-center gap-2">
                <span class="p-1 rounded-md bg-emerald-500/10 text-emerald-500">
                  <MemoryStick class="w-3.5 h-3.5" />
                </span>
                内存使用率
              </CardTitle>
              <span v-if="stats.container?.is_docker" class="text-[10px] px-1.5 py-0.5 rounded border border-emerald-500/30 text-emerald-600 dark:text-emerald-400 font-medium">
                docker stats
              </span>
              <span v-else class="text-[10px] px-1.5 py-0.5 rounded border border-border/60 text-muted-foreground bg-muted/20 font-medium tracking-wide">
                host mem
              </span>
            </div>
          </CardHeader>
          <CardContent class="flex flex-col items-center pt-1">
            <div class="relative w-32 h-32 flex items-center justify-center mt-1">
              <svg class="w-full h-full transform -rotate-90 drop-shadow-sm" viewBox="0 0 36 36">
                <defs>
                  <linearGradient id="memGrad" x1="0%" y1="0%" x2="100%" y2="100%">
                    <stop offset="0%" stop-color="#34d399" />
                    <stop offset="100%" stop-color="#059669" />
                  </linearGradient>
                  <linearGradient id="memGradWarn" x1="0%" y1="0%" x2="100%" y2="100%">
                    <stop offset="0%" stop-color="#fb7185" />
                    <stop offset="100%" stop-color="#e11d48" />
                  </linearGradient>
                </defs>
                <path class="text-muted/20" stroke-width="2.8" stroke="currentColor" fill="none" d="M18 2.0845 a 15.9155 15.9155 0 0 1 0 31.831 a 15.9155 15.9155 0 0 1 0 -31.831" />
                <path
                  :stroke="memPercent > 80 ? 'url(#memGradWarn)' : 'url(#memGrad)'"
                  stroke-dasharray="100, 100"
                  :stroke-dashoffset="100 - memPercent"
                  stroke-width="2.8"
                  stroke-linecap="round"
                  fill="none"
                  d="M18 2.0845 a 15.9155 15.9155 0 0 1 0 31.831 a 15.9155 15.9155 0 0 1 0 -31.831"
                  style="transition: stroke-dashoffset 0.6s ease;"
                />
              </svg>
              <div class="absolute flex flex-col items-center justify-center">
                <div class="flex items-baseline" :class="memPercent > 80 ? 'text-red-500' : 'text-foreground'">
                  <span class="text-2xl font-extrabold tracking-tight tabular-nums metric-num">{{ memPercent.toFixed(1) }}</span>
                  <span class="text-xs font-semibold text-muted-foreground ml-0.5">%</span>
                </div>
              </div>
            </div>
            <div class="w-full mt-3 p-2 rounded-lg bg-muted/20 border border-border/50 flex items-center justify-between text-xs">
              <span class="text-muted-foreground">已用 / 限额</span>
              <span class="font-medium text-foreground tabular-nums metric-num">
                <template v-if="stats.container?.is_docker">
                  {{ formatBytes(stats.container.docker_used) }} / {{ stats.container.limit > 0 ? formatBytes(stats.container.limit) : '未限额' }}
                </template>
                <template v-else>
                  {{ formatBytes(stats.host.mem_used) }} / {{ formatBytes(stats.host.mem_total) }}
                </template>
              </span>
            </div>
          </CardContent>
        </Card>

        <!-- 磁盘使用率 -->
        <Card class="border-border/60">
          <CardHeader class="pb-2">
            <div class="flex items-center justify-between">
              <CardTitle class="text-sm font-semibold text-foreground flex items-center gap-2">
                <span class="p-1 rounded-md bg-purple-500/10 text-purple-500">
                  <HardDrive class="w-3.5 h-3.5" />
                </span>
                磁盘使用率
              </CardTitle>
              <span class="text-[10px] px-1.5 py-0.5 rounded border border-border/60 text-muted-foreground bg-muted/20 font-medium tracking-wide">
                disk storage
              </span>
            </div>
          </CardHeader>
          <CardContent class="flex flex-col items-center pt-1">
            <div class="relative w-32 h-32 flex items-center justify-center mt-1">
              <svg class="w-full h-full transform -rotate-90 drop-shadow-sm" viewBox="0 0 36 36">
                <defs>
                  <linearGradient id="diskGrad" x1="0%" y1="0%" x2="100%" y2="100%">
                    <stop offset="0%" stop-color="#c084fc" />
                    <stop offset="100%" stop-color="#9333ea" />
                  </linearGradient>
                  <linearGradient id="diskGradWarn" x1="0%" y1="0%" x2="100%" y2="100%">
                    <stop offset="0%" stop-color="#fb7185" />
                    <stop offset="100%" stop-color="#e11d48" />
                  </linearGradient>
                </defs>
                <path class="text-muted/20" stroke-width="2.8" stroke="currentColor" fill="none" d="M18 2.0845 a 15.9155 15.9155 0 0 1 0 31.831 a 15.9155 15.9155 0 0 1 0 -31.831" />
                <path
                  :stroke="stats.host.disk_percent > 80 ? 'url(#diskGradWarn)' : 'url(#diskGrad)'"
                  stroke-dasharray="100, 100"
                  :stroke-dashoffset="100 - stats.host.disk_percent"
                  stroke-width="2.8"
                  stroke-linecap="round"
                  fill="none"
                  d="M18 2.0845 a 15.9155 15.9155 0 0 1 0 31.831 a 15.9155 15.9155 0 0 1 0 -31.831"
                  style="transition: stroke-dashoffset 0.6s ease;"
                />
              </svg>
              <div class="absolute flex flex-col items-center justify-center">
                <div class="flex items-baseline" :class="stats.host.disk_percent > 80 ? 'text-red-500' : 'text-foreground'">
                  <span class="text-2xl font-extrabold tracking-tight tabular-nums metric-num">{{ stats.host.disk_percent.toFixed(1) }}</span>
                  <span class="text-xs font-semibold text-muted-foreground ml-0.5">%</span>
                </div>
              </div>
            </div>
            <div class="w-full mt-3 p-2 rounded-lg bg-muted/20 border border-border/50 flex items-center justify-between text-xs">
              <span class="text-muted-foreground">已用 / 总容量</span>
              <span class="font-medium text-foreground tabular-nums metric-num">
                {{ formatBytes(stats.host.disk_used) }} / {{ formatBytes(stats.host.disk_total) }}
              </span>
            </div>
          </CardContent>
        </Card>
      </div>

      <div v-if="stats" class="grid grid-cols-1 gap-4 mt-4" :class="stats.container?.is_docker ? 'lg:grid-cols-3 md:grid-cols-2' : 'md:grid-cols-2'">
          <!-- 卡片 1: 执行环境 -->
          <Card class="border-border/60">
            <CardHeader class="pb-2">
              <div class="flex items-center justify-between">
                <CardTitle class="text-sm font-semibold text-foreground flex items-center gap-2">
                  <span class="p-1 rounded-md bg-blue-500/10 text-blue-500"><Server class="w-3.5 h-3.5" /></span>
                  执行环境
                </CardTitle>
                <span class="text-[10px] px-1.5 py-0.5 rounded border border-border/60 text-muted-foreground bg-muted/20 font-medium tracking-wide">
                  runtime env
                </span>
              </div>
            </CardHeader>
            <CardContent class="space-y-3">
              <!-- 核心指标看板：仅展示动态运行时负载与协程 -->
              <div class="rounded-lg p-2.5 bg-muted/20 dark:bg-zinc-900/50 border border-border/50 space-y-1">
                <div class="flex items-center justify-between text-xs text-muted-foreground">
                  <span>活跃协程数 (Goroutines)</span>
                  <span class="text-[10px] text-emerald-500 font-medium">[GMP 调度正常]</span>
                </div>
                <div class="flex items-baseline justify-between">
                  <div class="text-xl font-bold text-foreground tabular-nums metric-num tracking-tight">
                    {{ stats.env.goroutines }} <span class="text-xs font-normal text-muted-foreground">Goroutines</span>
                  </div>
                  <div class="text-[11px] text-muted-foreground tabular-nums">
                    GOMAXPROCS: {{ stats.env.num_cpu }}
                  </div>
                </div>
                <div class="text-[10px] text-muted-foreground">
                  Go 轻量级线程并发调度模型
                </div>
              </div>

              <!-- 静态底层契约：仅展示平台、版本与硬件架构，不重复罗列协程 -->
              <div class="space-y-1 text-sm">
                <div class="flex justify-between items-center border-b border-border/40 pb-1 font-medium text-xs text-muted-foreground">
                  <span>底层平台契约</span>
                  <span class="text-foreground font-semibold tabular-nums">{{ stats.env.os }}_{{ stats.env.arch }}</span>
                </div>
                <div class="flex justify-between items-center border-b border-border/40 pb-1 text-xs">
                  <dt class="text-muted-foreground flex items-center gap-1.5 min-w-0 truncate">
                    <span class="text-muted-foreground/40 select-none shrink-0">├─</span> Go 语言版本
                  </dt>
                  <dd class="font-medium text-foreground tabular-nums shrink-0 whitespace-nowrap text-right">
                    {{ stats.env.go_version }}
                  </dd>
                </div>
                <div class="flex justify-between items-center border-b border-border/40 pb-1 text-xs">
                  <dt class="text-muted-foreground flex items-center gap-1.5 min-w-0 truncate">
                    <span class="text-muted-foreground/40 select-none shrink-0">├─</span> 操作系统
                  </dt>
                  <dd class="font-medium text-foreground tabular-nums shrink-0 whitespace-nowrap text-right">
                    {{ stats.env.os }}
                  </dd>
                </div>
                <div class="flex justify-between items-center border-b border-border/40 pb-1 text-xs">
                  <dt class="text-muted-foreground flex items-center gap-1.5 min-w-0 truncate">
                    <span class="text-muted-foreground/40 select-none shrink-0">├─</span> 硬件指令架构
                  </dt>
                  <dd class="font-medium text-foreground tabular-nums shrink-0 whitespace-nowrap text-right">
                    {{ stats.env.arch }}
                  </dd>
                </div>
                <div class="flex justify-between items-center pb-0.5 text-xs">
                  <dt class="text-muted-foreground flex items-center gap-1.5 min-w-0 truncate">
                    <span class="text-muted-foreground/40 select-none shrink-0">└─</span> 物理逻辑核心
                  </dt>
                  <dd class="font-medium text-foreground tabular-nums shrink-0 whitespace-nowrap text-right">
                    {{ stats.env.num_cpu }} 逻辑核
                  </dd>
                </div>
              </div>
            </CardContent>
          </Card>

          <!-- 卡片 2: 任务调度 -->
          <Card class="border-border/60">
            <CardHeader class="pb-2">
              <div class="flex items-center justify-between">
                <CardTitle class="text-sm font-semibold text-foreground flex items-center gap-2">
                  <span class="p-1 rounded-md bg-indigo-500/10 text-indigo-500"><CalendarClock class="w-3.5 h-3.5" /></span>
                  任务调度
                </CardTitle>
                <span class="text-[10px] px-1.5 py-0.5 rounded border border-border/60 text-muted-foreground bg-muted/20 font-medium tracking-wide">
                  queue & pool
                </span>
              </div>
            </CardHeader>
            <CardContent class="space-y-3">
              <!-- 核心看板：当前活跃任务与并发池利用率 -->
              <div class="rounded-lg p-2.5 bg-muted/20 dark:bg-zinc-900/50 border border-border/50 space-y-1">
                <div class="flex items-center justify-between text-xs text-muted-foreground">
                  <span>当前运行任务 (Running)</span>
                  <span :class="stats.scheduler.queue_size > 0 ? 'text-amber-500 font-medium' : 'text-emerald-500 font-normal'">
                    {{ stats.scheduler.queue_size > 0 ? `${stats.scheduler.queue_size} 个排队中` : '管线畅通' }}
                  </span>
                </div>
                <div class="flex items-baseline justify-between">
                  <div class="text-xl font-bold tabular-nums metric-num tracking-tight" :class="stats.scheduler.running > 0 ? 'text-emerald-500' : 'text-foreground'">
                    {{ stats.scheduler.running }} <span class="text-xs font-normal text-muted-foreground">/ {{ stats.scheduler.worker_count }} 槽位</span>
                  </div>
                  <div class="text-[11px] text-muted-foreground tabular-nums">
                    负载率 {{ (stats.scheduler.running / (stats.scheduler.worker_count || 1) * 100).toFixed(0) }}%
                  </div>
                </div>
                <div class="text-[10px] text-muted-foreground">
                  并发工作池负载与队列处理状态
                </div>
              </div>

              <!-- 调度池详细容量：不重复写 Running -->
              <div class="space-y-1 text-sm">
                <div class="flex justify-between items-center border-b border-border/40 pb-1 font-medium text-xs text-muted-foreground">
                  <span>调度池配置与积压</span>
                  <span class="text-foreground font-semibold tabular-nums">{{ stats.scheduler.worker_count }} 工作协程</span>
                </div>
                <div class="flex justify-between items-center border-b border-border/40 pb-1 text-xs">
                  <dt class="text-muted-foreground flex items-center gap-1.5 min-w-0 truncate">
                    <span class="text-muted-foreground/40 select-none shrink-0">├─</span> 并发池上限 (Worker Limit)
                  </dt>
                  <dd class="font-medium text-foreground tabular-nums shrink-0 whitespace-nowrap text-right">
                    {{ stats.scheduler.worker_count }} 线程上限
                  </dd>
                </div>
                <div class="flex justify-between items-center border-b border-border/40 pb-1 text-xs">
                  <dt class="text-muted-foreground flex items-center gap-1.5 min-w-0 truncate">
                    <span class="text-muted-foreground/40 select-none shrink-0">├─</span> 排队等待数 (Queue Size)
                  </dt>
                  <dd class="font-medium tabular-nums shrink-0 whitespace-nowrap text-right" :class="stats.scheduler.queue_size > 0 ? 'text-amber-500 font-bold' : 'text-foreground'">
                    {{ stats.scheduler.queue_size }} 个
                  </dd>
                </div>
                <div class="flex justify-between items-center pb-0.5 text-xs">
                  <dt class="text-muted-foreground flex items-center gap-1.5 min-w-0 truncate">
                    <span class="text-muted-foreground/40 select-none shrink-0">└─</span> 驻留定时任务 (Scheduled)
                  </dt>
                  <dd class="font-medium text-foreground tabular-nums shrink-0 whitespace-nowrap text-right">
                    {{ stats.scheduler.scheduled }} 个定时计划
                  </dd>
                </div>
              </div>
            </CardContent>
          </Card>

          <!-- Docker 容器内存剖析卡片 (仅在 Docker 容器环境回显) -->
          <Card v-if="stats.container?.is_docker" class="border-border/60">
            <CardHeader class="pb-2">
              <div class="flex items-center justify-between">
                <CardTitle class="text-sm font-semibold text-foreground flex items-center gap-2">
                  <span class="p-1 rounded-md bg-emerald-500/10 text-emerald-500"><Cpu class="w-3.5 h-3.5" /></span>
                  容器内存剖析
                </CardTitle>
                <span class="text-[10px] px-1.5 py-0.5 rounded border border-border/60 text-muted-foreground bg-muted/20 font-medium tracking-wide">
                  docker stats
                </span>
              </div>
            </CardHeader>
            <CardContent class="space-y-3">
              <!-- 板块 1: docker stats 官方计算公式 (减法口径: Used = Current - Inactive) -->
              <div class="rounded-lg p-2.5 bg-muted/20 dark:bg-zinc-900/50 border border-border/50 space-y-1">
                <div class="flex items-center justify-between text-xs text-muted-foreground">
                  <span>Docker 真实占用 (MEM USAGE)</span>
                  <span v-if="stats.container.limit > 0" class="tabular-nums text-foreground/80">
                    {{ stats.container.limit_percent.toFixed(2) }}% ({{ formatBytes(stats.container.limit) }})
                  </span>
                </div>
                <div class="flex items-baseline justify-between">
                  <div class="text-xl font-bold text-foreground tabular-nums metric-num tracking-tight">
                    {{ formatBytes(stats.container.docker_used) }}
                  </div>
                  <div class="text-[11px] text-muted-foreground tabular-nums">
                    = {{ formatBytes(stats.container.total_usage) }} - {{ formatBytes(stats.container.inactive_file) }}
                  </div>
                </div>
                <div class="text-[10px] text-muted-foreground flex justify-between">
                  <span>扣除随时可回收的冷缓存</span>
                  <span class="text-muted-foreground/80 font-medium">精确吻合 docker stats</span>
                </div>
              </div>

              <!-- 板块 2: 总物理内存加法拆解 (四项相加 = Total Current) -->
              <div class="space-y-1 text-sm">
                <div class="flex justify-between items-center border-b border-border/40 pb-1 font-medium text-xs text-muted-foreground">
                  <span>总物理内存构成 (四项相加)</span>
                  <span class="text-foreground font-semibold tabular-nums">{{ formatBytes(stats.container.total_usage) }} (100%)</span>
                </div>
                <div class="flex justify-between items-center border-b border-border/40 pb-1 text-xs">
                  <dt class="text-muted-foreground flex items-center gap-1.5 min-w-0 truncate">
                    <span class="text-muted-foreground/40 select-none shrink-0">├─</span> 进程堆栈 (Anon)
                  </dt>
                  <dd class="font-medium text-foreground tabular-nums shrink-0 whitespace-nowrap text-right">
                    {{ formatBytes(stats.container.anon) }}
                    <span class="text-[10px] text-muted-foreground ml-1">(白虎自身数据)</span>
                  </dd>
                </div>
                <div class="flex justify-between items-center border-b border-border/40 pb-1 text-xs">
                  <dt class="text-muted-foreground flex items-center gap-1.5 min-w-0 truncate">
                    <span class="text-muted-foreground/40 select-none shrink-0">├─</span> 活跃类库 (Active File)
                  </dt>
                  <dd class="font-medium text-foreground tabular-nums shrink-0 whitespace-nowrap text-right">
                    {{ formatBytes(stats.container.active_file) }}
                    <span class="text-[10px] text-muted-foreground ml-1">(代码段缓存)</span>
                  </dd>
                </div>
                <div class="flex justify-between items-center border-b border-border/40 pb-1 text-xs">
                  <dt class="text-muted-foreground flex items-center gap-1.5 min-w-0 truncate">
                    <span class="text-muted-foreground/40 select-none shrink-0">├─</span> 目录索引 (Slab)
                  </dt>
                  <dd class="font-medium text-foreground tabular-nums shrink-0 whitespace-nowrap text-right">
                    {{ formatBytes(stats.container.slab_reclaimable) }}
                    <span class="text-[10px] text-muted-foreground ml-1">(内核元数据)</span>
                  </dd>
                </div>
                <div class="flex justify-between items-center pb-0.5 text-xs">
                  <dt class="text-foreground font-medium flex items-center gap-1.5 min-w-0 truncate">
                    <span class="text-muted-foreground/40 select-none shrink-0">└─</span> 非活跃冷缓存 (Inactive)
                  </dt>
                  <dd class="font-semibold text-foreground tabular-nums shrink-0 whitespace-nowrap text-right">
                    {{ formatBytes(stats.container.inactive_file) }}
                    <span class="text-[10px] text-muted-foreground ml-1">[-已扣除]</span>
                  </dd>
                </div>
              </div>
            </CardContent>
          </Card>
      </div>
      <div v-if="stats && stats.scheduler.workers" class="mt-4">
        <Card class="border-border/60">
            <CardHeader class="pb-3">
              <div class="flex items-center justify-between gap-3">
                <div class="space-y-0.5">
                  <CardTitle class="text-sm font-semibold text-foreground flex items-center gap-2">
                    <span class="p-1 rounded-md bg-amber-500/10 text-amber-500">
                      <Layers class="w-3.5 h-3.5" />
                    </span>
                    并发池 Worker 调度
                  </CardTitle>
                  <CardDescription class="text-xs">底层协程并发池任务调度与实时执行状态</CardDescription>
                </div>
                <!-- 统计徽章 -->
                <div class="shrink-0">
                  <span
                    v-if="busyWorkersCount > 0"
                    class="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-medium bg-amber-500/10 text-amber-600 dark:text-amber-400 border border-amber-500/20"
                  >
                    <span class="relative flex h-2 w-2">
                      <span class="animate-ping absolute inline-flex h-full w-full rounded-full bg-amber-400 opacity-75"></span>
                      <span class="relative inline-flex rounded-full h-2 w-2 bg-amber-500"></span>
                    </span>
                    {{ busyWorkersCount }} / {{ totalWorkersCount }} 运行中
                  </span>
                  <span
                    v-else
                    class="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-medium bg-muted text-muted-foreground border border-border/60"
                  >
                    <span class="w-1.5 h-1.5 rounded-full bg-emerald-500"></span>
                    全部就绪待命 ({{ totalWorkersCount }} 并发槽)
                  </span>
                </div>
              </div>
            </CardHeader>
            <CardContent>
              <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-3">
                <div
                  v-for="worker in stats.scheduler.workers"
                  :key="worker.id"
                  class="rounded-lg p-3 transition-all duration-200 border flex flex-col justify-between"
                  :class="[
                    worker.status === 'running'
                      ? 'bg-amber-500/5 border-amber-500/30 shadow-sm shadow-amber-500/5'
                      : 'bg-muted/15 border-border/50 hover:bg-muted/25'
                  ]"
                >
                  <div class="flex items-center justify-between mb-2">
                    <div class="flex items-center gap-1.5 text-xs font-semibold text-foreground">
                      <span class="text-muted-foreground/60 font-medium">#</span><span class="tabular-nums">{{ worker.id }}</span>
                      <span class="text-[11px] font-normal text-muted-foreground">Worker</span>
                    </div>
                    <span
                      v-if="worker.status === 'idle'"
                      class="inline-flex items-center gap-1 px-1.5 py-0.5 rounded text-[10px] font-medium bg-muted/50 text-muted-foreground border border-border/40"
                    >
                      <span class="w-1.5 h-1.5 rounded-full bg-emerald-500/80"></span>
                      空闲
                    </span>
                    <span
                      v-else
                      class="inline-flex items-center gap-1 px-1.5 py-0.5 rounded text-[10px] font-medium bg-amber-500/15 text-amber-600 dark:text-amber-400 border border-amber-500/30"
                    >
                      <span class="w-1.5 h-1.5 rounded-full bg-amber-500 animate-pulse"></span>
                      运行中
                    </span>
                  </div>

                  <div v-if="worker.status === 'running'" class="space-y-1.5 text-xs pt-1">
                    <div class="flex items-center gap-1.5 min-w-0" :title="worker.task_name || worker.task_id">
                      <Play class="w-3 h-3 text-amber-500 shrink-0 fill-amber-500/20" />
                      <span class="font-medium text-foreground truncate text-[11px]">
                        {{ worker.task_name || worker.task_id }}
                      </span>
                    </div>
                    <div class="flex items-center justify-between text-[10px] text-muted-foreground pt-1 border-t border-amber-500/10">
                      <span>已运行</span>
                      <span class="tabular-nums metric-num text-amber-600 dark:text-amber-400 font-semibold">{{ worker.duration || 0 }}s</span>
                    </div>
                  </div>
                  <div v-else class="flex items-center justify-between text-[11px] text-muted-foreground/60 pt-1">
                    <span>暂无调度</span>
                    <span class="text-[10px] tracking-wide text-muted-foreground/50 font-medium">IDLE</span>
                  </div>
                </div>
              </div>
            </CardContent>
        </Card>
      </div>

      <div v-if="stats" class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4 mt-4">
          <!-- 卡片 3: 内存概览 -->
          <Card class="border-border/60">
            <CardHeader class="pb-2">
              <div class="flex items-center justify-between">
                <CardTitle class="text-sm font-semibold text-foreground flex items-center gap-2">
                  <span class="p-1 rounded-md bg-emerald-500/10 text-emerald-500"><MemoryStick class="w-3.5 h-3.5" /></span>
                  内存概览
                </CardTitle>
                <span class="text-[10px] px-1.5 py-0.5 rounded border border-border/60 text-muted-foreground bg-muted/20 font-medium tracking-wide">
                  process mem
                </span>
              </div>
            </CardHeader>
            <CardContent class="space-y-3">
              <!-- 核心看板：仅展示当前常驻分配 Alloc 与利用率，不写 Sys -->
              <div class="rounded-lg p-2.5 bg-muted/20 dark:bg-zinc-900/50 border border-border/50 space-y-1">
                <div class="flex items-center justify-between text-xs text-muted-foreground">
                  <span>当前活跃堆内存 (Alloc)</span>
                  <span class="text-[11px] tabular-nums text-foreground/80">占系统申请 {{ (stats.mem.alloc / (stats.mem.sys || 1) * 100).toFixed(1) }}%</span>
                </div>
                <div class="flex items-baseline justify-between">
                  <div class="text-xl font-bold text-foreground tabular-nums metric-num tracking-tight">
                    {{ formatBytes(stats.mem.alloc) }}
                  </div>
                  <div class="text-[11px] text-muted-foreground tabular-nums">
                    常驻纯净堆
                  </div>
                </div>
                <div class="text-[10px] text-muted-foreground">
                  Go 活跃存活堆对象实际空间
                </div>
              </div>

              <!-- 内存生命周期拆解：不重复写 Alloc -->
              <div class="space-y-1 text-sm">
                <div class="flex justify-between items-center border-b border-border/40 pb-1 font-medium text-xs text-muted-foreground">
                  <span>系统堆与对象生命周期</span>
                  <span class="text-foreground font-semibold tabular-nums">{{ formatBytes(stats.mem.sys) }} (保留堆)</span>
                </div>
                <div class="flex justify-between items-center border-b border-border/40 pb-1 text-xs">
                  <dt class="text-muted-foreground flex items-center gap-1.5 min-w-0 truncate">
                    <span class="text-muted-foreground/40 select-none shrink-0">├─</span> OS 申请上限 (Sys)
                  </dt>
                  <dd class="font-medium text-foreground tabular-nums shrink-0 whitespace-nowrap text-right">
                    {{ formatBytes(stats.mem.sys) }}
                  </dd>
                </div>
                <div class="flex justify-between items-center border-b border-border/40 pb-1 text-xs">
                  <dt class="text-muted-foreground flex items-center gap-1.5 min-w-0 truncate">
                    <span class="text-muted-foreground/40 select-none shrink-0">├─</span> 累计历史分配 (TotalAlloc)
                  </dt>
                  <dd class="font-medium text-foreground tabular-nums shrink-0 whitespace-nowrap text-right">
                    {{ formatBytes(stats.mem.total_alloc) }}
                  </dd>
                </div>
                <div class="flex justify-between items-center border-b border-border/40 pb-1 text-xs">
                  <dt class="text-muted-foreground flex items-center gap-1.5 min-w-0 truncate">
                    <span class="text-muted-foreground/40 select-none shrink-0">├─</span> 对象分配总次数 (Mallocs)
                  </dt>
                  <dd class="font-medium text-foreground tabular-nums shrink-0 whitespace-nowrap text-right">
                    {{ stats.mem.mallocs.toLocaleString() }} 次
                  </dd>
                </div>
                <div class="flex justify-between items-center pb-0.5 text-xs">
                  <dt class="text-muted-foreground flex items-center gap-1.5 min-w-0 truncate">
                    <span class="text-muted-foreground/40 select-none shrink-0">└─</span> 对象释放总次数 (Frees)
                  </dt>
                  <dd class="font-medium text-foreground tabular-nums shrink-0 whitespace-nowrap text-right">
                    {{ stats.mem.frees.toLocaleString() }} 次
                    <span class="text-[10px] text-emerald-500 font-normal ml-1">[正常]</span>
                  </dd>
                </div>
              </div>
            </CardContent>
          </Card>

          <!-- 卡片 4: 堆栈明细 -->
          <Card class="border-border/60">
            <CardHeader class="pb-2">
              <div class="flex items-center justify-between">
                <CardTitle class="text-sm font-semibold text-foreground flex items-center gap-2">
                  <span class="p-1 rounded-md bg-purple-500/10 text-purple-500"><HardDrive class="w-3.5 h-3.5" /></span>
                  堆栈明细
                </CardTitle>
                <span class="text-[10px] px-1.5 py-0.5 rounded border border-border/60 text-muted-foreground bg-muted/20 font-medium tracking-wide">
                  heap profile
                </span>
              </div>
            </CardHeader>
            <CardContent class="space-y-3">
              <!-- 核心看板：使用中堆内存与利用率，不塞入空闲堆与系统堆 -->
              <div class="rounded-lg p-2.5 bg-muted/20 dark:bg-zinc-900/50 border border-border/50 space-y-1">
                <div class="flex items-center justify-between text-xs text-muted-foreground">
                  <span>使用中堆内存 (HeapInuse)</span>
                  <span class="tabular-nums text-foreground/80">堆利用率: {{ (stats.heap.heap_inuse / (stats.heap.heap_sys || 1) * 100).toFixed(1) }}%</span>
                </div>
                <div class="flex items-baseline justify-between">
                  <div class="text-xl font-bold text-foreground tabular-nums metric-num tracking-tight">
                    {{ formatBytes(stats.heap.heap_inuse) }}
                  </div>
                  <div class="text-[11px] text-muted-foreground tabular-nums">
                    {{ stats.heap.heap_objects.toLocaleString() }} 个存活对象
                  </div>
                </div>
                <div class="text-[10px] text-muted-foreground">
                  当前处于活跃状态的堆结构切片
                </div>
              </div>

              <!-- 堆内存去向拆解：清晰罗列空闲、归还，不重复写 HeapInuse/HeapAlloc -->
              <div class="space-y-1 text-sm">
                <div class="flex justify-between items-center border-b border-border/40 pb-1 font-medium text-xs text-muted-foreground">
                  <span>堆内存结构拆解 (总堆 {{ formatBytes(stats.heap.heap_sys) }})</span>
                  <span class="text-foreground font-semibold tabular-nums">100%</span>
                </div>
                <div class="flex justify-between items-center border-b border-border/40 pb-1 text-xs">
                  <dt class="text-muted-foreground flex items-center gap-1.5 min-w-0 truncate">
                    <span class="text-muted-foreground/40 select-none shrink-0">├─</span> 跨协程空闲堆 (HeapIdle)
                  </dt>
                  <dd class="font-medium text-foreground tabular-nums shrink-0 whitespace-nowrap text-right">
                    {{ formatBytes(stats.heap.heap_idle) }}
                    <span class="text-[10px] text-muted-foreground ml-1">(可复用)</span>
                  </dd>
                </div>
                <div class="flex justify-between items-center border-b border-border/40 pb-1 text-xs">
                  <dt class="text-muted-foreground flex items-center gap-1.5 min-w-0 truncate">
                    <span class="text-muted-foreground/40 select-none shrink-0">├─</span> 操作系统退还 (Released)
                  </dt>
                  <dd class="font-medium text-foreground tabular-nums shrink-0 whitespace-nowrap text-right">
                    {{ formatBytes(stats.heap.heap_released) }}
                    <span class="text-[10px] text-muted-foreground ml-1">(Decommit)</span>
                  </dd>
                </div>
                <div class="flex justify-between items-center pb-0.5 text-xs">
                  <dt class="text-muted-foreground flex items-center gap-1.5 min-w-0 truncate">
                    <span class="text-muted-foreground/40 select-none shrink-0">└─</span> 系统堆保留总量 (HeapSys)
                  </dt>
                  <dd class="font-medium text-foreground tabular-nums shrink-0 whitespace-nowrap text-right">
                    {{ formatBytes(stats.heap.heap_sys) }}
                  </dd>
                </div>
              </div>
            </CardContent>
          </Card>

          <!-- 卡片 5: 垃圾回收 (GC) -->
          <Card class="border-border/60">
            <CardHeader class="pb-2">
              <div class="flex items-center justify-between">
                <CardTitle class="text-sm font-semibold text-foreground flex items-center gap-2">
                  <span class="p-1 rounded-md bg-amber-500/10 text-amber-500"><RefreshCw class="w-3.5 h-3.5" /></span>
                  垃圾回收 (GC)
                </CardTitle>
                <span class="text-[10px] px-1.5 py-0.5 rounded border border-border/60 text-muted-foreground bg-muted/20 font-medium tracking-wide">
                  gc stats
                </span>
              </div>
            </CardHeader>
            <CardContent class="space-y-3">
              <!-- 核心看板：仅展示下次触发阈值 NextGC，不重复塞入停顿与次数 -->
              <div class="rounded-lg p-2.5 bg-muted/20 dark:bg-zinc-900/50 border border-border/50 space-y-1">
                <div class="flex items-center justify-between text-xs text-muted-foreground">
                  <span>下次触发阈值 (NextGC)</span>
                  <span class="text-[10px] text-foreground/80 font-medium">GOGC=80 激进回收</span>
                </div>
                <div class="flex items-baseline justify-between">
                  <div class="text-xl font-bold text-foreground tabular-nums metric-num tracking-tight">
                    {{ formatBytes(stats.gc.next_gc) }}
                  </div>
                  <div class="text-[11px] text-muted-foreground tabular-nums">
                    堆达此大小触发下一轮
                  </div>
                </div>
                <div class="text-[10px] text-muted-foreground">
                  自动平衡吞吐量与内存占用上限
                </div>
              </div>

              <!-- 统计拆解：在此展示停顿、次数、上次时间，不与看板重复 -->
              <div class="space-y-1 text-sm">
                <div class="flex justify-between items-center border-b border-border/40 pb-1 font-medium text-xs text-muted-foreground">
                  <span>GC 运行指标与停顿统计</span>
                  <span class="text-foreground font-semibold tabular-nums">{{ stats.gc.num_gc }} 次回收</span>
                </div>
                <div class="flex justify-between items-center border-b border-border/40 pb-1 text-xs">
                  <dt class="text-muted-foreground flex items-center gap-1.5 min-w-0 truncate">
                    <span class="text-muted-foreground/40 select-none shrink-0">├─</span> 最近触发时间 (LastGC)
                  </dt>
                  <dd class="font-medium text-foreground tabular-nums text-[11px] shrink-0 whitespace-nowrap text-right">
                    {{ stats.gc.last_gc ? formatDateTime(new Date(stats.gc.last_gc / 1000000)) : '-' }}
                  </dd>
                </div>
                <div class="flex justify-between items-center border-b border-border/40 pb-1 text-xs">
                  <dt class="text-muted-foreground flex items-center gap-1.5 min-w-0 truncate">
                    <span class="text-muted-foreground/40 select-none shrink-0">├─</span> 累计停顿耗时 (PauseTotal)
                  </dt>
                  <dd class="font-medium text-foreground tabular-nums shrink-0 whitespace-nowrap text-right">
                    {{ formatNs(stats.gc.pause_total_ns) }}
                    <span class="text-[10px] text-muted-foreground ml-1">(微秒级)</span>
                  </dd>
                </div>
                <div class="flex justify-between items-center pb-0.5 text-xs">
                  <dt class="text-muted-foreground flex items-center gap-1.5 min-w-0 truncate">
                    <span class="text-muted-foreground/40 select-none shrink-0">└─</span> 物理页主动退还
                  </dt>
                  <dd class="font-medium text-foreground shrink-0 whitespace-nowrap text-right">
                    FreeOSMemory
                    <span class="text-[10px] text-emerald-500 font-normal ml-1">[已激活]</span>
                  </dd>
                </div>
              </div>
            </CardContent>
          </Card>
      </div>
    </TabsContent>

    <TabsContent value="charts" class="mt-0 space-y-4">
      <div class="grid grid-cols-1 lg:grid-cols-2 gap-4">
          <!-- Scheduler 图表 -->
          <Card>
            <CardHeader class="pt-4 pb-1 px-4">
              <CardTitle class="text-sm font-semibold text-indigo-600">任务调度 (Scheduler)</CardTitle>
              <CardDescription class="text-xs">当前节点上正在运行和调度的任务数</CardDescription>
            </CardHeader>
            <CardContent class="px-4 pb-4 pt-0">
              <div class="h-52 w-full mt-2">
                <Line :data="schedulerChartData" :options="chartOptions" />
              </div>
            </CardContent>
          </Card>

          <!-- Goroutines 图表 -->
          <Card>
            <CardHeader class="pt-4 pb-1 px-4">
              <CardTitle class="text-sm font-semibold text-blue-600">并发协程 (Goroutines)</CardTitle>
              <CardDescription class="text-xs">轻量级线程的并发数量跟踪</CardDescription>
            </CardHeader>
            <CardContent class="px-4 pb-4 pt-0">
              <div class="h-52 w-full mt-2">
                <Line :data="goroutineChartData" :options="chartOptions" />
              </div>
            </CardContent>
          </Card>

          <!-- Memory 图表 -->
          <Card>
            <CardHeader class="pt-4 pb-1 px-4">
              <CardTitle class="text-sm font-semibold text-emerald-600">内存分配 (Memory Alloc vs Sys)</CardTitle>
              <CardDescription class="text-xs">实际使用内存与系统申请上限对比</CardDescription>
            </CardHeader>
            <CardContent class="px-4 pb-4 pt-0">
              <div class="h-52 w-full mt-2">
                <Line :data="memChartData" :options="chartOptions" />
              </div>
            </CardContent>
          </Card>

          <!-- GC 图表 -->
          <Card>
            <CardHeader class="pt-4 pb-1 px-4">
              <CardTitle class="text-sm font-semibold text-orange-600">垃圾回收停顿 (GC Pauses)</CardTitle>
              <CardDescription class="text-xs">探针周期内 GC 暂停总耗时（ms）</CardDescription>
            </CardHeader>
            <CardContent class="px-4 pb-4 pt-0">
              <div class="h-52 w-full mt-2">
                <Bar :data="gcChartData" :options="chartOptions" />
              </div>
            </CardContent>
          </Card>
        </div>
    </TabsContent>
  </Tabs>
</template>

<style scoped>
.metric-num {
  font-family: var(--font-main);
  font-feature-settings: 'tnum' 1, 'cv05' 1, 'cv08' 1, 'cv11' 1, 'ss01' 1;
  font-variant-numeric: tabular-nums;
}

/* 小屏下数值卡字体强化加粗 */
@media (max-width: 639px) {
  .metric-num {
    font-weight: 800 !important;
  }

  /* 环形图中央核心数值进一步强化至 900，更饱满厚实 */
  .metric-num.text-2xl {
    font-weight: 900 !important;
  }
}
</style>
