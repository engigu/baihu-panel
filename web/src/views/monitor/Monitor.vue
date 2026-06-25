<script setup lang="ts">
import { ref, watch, onMounted, onUnmounted, computed } from 'vue'
import type { MonitorStats } from '@/api'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import StatusDot from '@/components/StatusDot.vue'
import { RefreshCw, Cpu, MemoryStick, HardDrive, Activity, LayoutDashboard } from 'lucide-vue-next'

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

const activeTab = ref(localStorage.getItem('monitor_active_tab') || 'charts')

watch(activeTab, (newVal) => {
  localStorage.setItem('monitor_active_tab', newVal)
})

const stats = ref<MonitorStats | null>(null)
const loading = ref(false)

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
      


      <!-- 物理主机监控环形图 -->
      <div v-if="stats" class="grid grid-cols-1 md:grid-cols-3 gap-4 mt-4">
        <Card>
          <CardHeader class="pb-2">
            <CardTitle class="text-base font-medium text-muted-foreground flex items-center"><Cpu class="w-4 h-4 mr-2" /> CPU 使用率</CardTitle>
          </CardHeader>
          <CardContent class="flex flex-col items-center">
            <div class="relative w-32 h-32 flex items-center justify-center mt-2">
              <svg class="w-full h-full transform -rotate-90" viewBox="0 0 36 36">
                <path class="text-muted/20" stroke-width="3" stroke="currentColor" fill="none" d="M18 2.0845 a 15.9155 15.9155 0 0 1 0 31.831 a 15.9155 15.9155 0 0 1 0 -31.831" />
                <path :class="stats.host.cpu_percent > 80 ? 'text-red-500' : 'text-blue-500'" stroke-dasharray="100, 100" :stroke-dashoffset="100 - stats.host.cpu_percent" stroke-width="3" stroke-linecap="round" stroke="currentColor" fill="none" d="M18 2.0845 a 15.9155 15.9155 0 0 1 0 31.831 a 15.9155 15.9155 0 0 1 0 -31.831" style="transition: stroke-dashoffset 0.5s ease 0s;" />
              </svg>
              <div class="absolute text-2xl font-bold" :class="stats.host.cpu_percent > 80 ? 'text-red-500' : 'text-foreground'">{{ stats.host.cpu_percent.toFixed(1) }}%</div>
            </div>
            <div class="text-xs text-muted-foreground mt-4">宿主机物理核心负载</div>
          </CardContent>
        </Card>
        
        <Card>
          <CardHeader class="pb-2">
            <CardTitle class="text-base font-medium text-muted-foreground flex items-center"><MemoryStick class="w-4 h-4 mr-2" /> 内存使用率</CardTitle>
          </CardHeader>
          <CardContent class="flex flex-col items-center">
            <div class="relative w-32 h-32 flex items-center justify-center mt-2">
              <svg class="w-full h-full transform -rotate-90" viewBox="0 0 36 36">
                <path class="text-muted/20" stroke-width="3" stroke="currentColor" fill="none" d="M18 2.0845 a 15.9155 15.9155 0 0 1 0 31.831 a 15.9155 15.9155 0 0 1 0 -31.831" />
                <path :class="stats.host.mem_percent > 80 ? 'text-red-500' : 'text-emerald-500'" stroke-dasharray="100, 100" :stroke-dashoffset="100 - stats.host.mem_percent" stroke-width="3" stroke-linecap="round" stroke="currentColor" fill="none" d="M18 2.0845 a 15.9155 15.9155 0 0 1 0 31.831 a 15.9155 15.9155 0 0 1 0 -31.831" style="transition: stroke-dashoffset 0.5s ease 0s;" />
              </svg>
              <div class="absolute text-2xl font-bold" :class="stats.host.mem_percent > 80 ? 'text-red-500' : 'text-foreground'">{{ stats.host.mem_percent.toFixed(1) }}%</div>
            </div>
            <div class="text-xs text-muted-foreground mt-4">{{ formatBytes(stats.host.mem_used) }} / {{ formatBytes(stats.host.mem_total) }}</div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader class="pb-2">
            <CardTitle class="text-base font-medium text-muted-foreground flex items-center"><HardDrive class="w-4 h-4 mr-2" /> 磁盘使用率</CardTitle>
          </CardHeader>
          <CardContent class="flex flex-col items-center">
            <div class="relative w-32 h-32 flex items-center justify-center mt-2">
              <svg class="w-full h-full transform -rotate-90" viewBox="0 0 36 36">
                <path class="text-muted/20" stroke-width="3" stroke="currentColor" fill="none" d="M18 2.0845 a 15.9155 15.9155 0 0 1 0 31.831 a 15.9155 15.9155 0 0 1 0 -31.831" />
                <path :class="stats.host.disk_percent > 80 ? 'text-red-500' : 'text-purple-500'" stroke-dasharray="100, 100" :stroke-dashoffset="100 - stats.host.disk_percent" stroke-width="3" stroke-linecap="round" stroke="currentColor" fill="none" d="M18 2.0845 a 15.9155 15.9155 0 0 1 0 31.831 a 15.9155 15.9155 0 0 1 0 -31.831" style="transition: stroke-dashoffset 0.5s ease 0s;" />
              </svg>
              <div class="absolute text-2xl font-bold" :class="stats.host.disk_percent > 80 ? 'text-red-500' : 'text-foreground'">{{ stats.host.disk_percent.toFixed(1) }}%</div>
            </div>
            <div class="text-xs text-muted-foreground mt-4">{{ formatBytes(stats.host.disk_used) }} / {{ formatBytes(stats.host.disk_total) }}</div>
          </CardContent>
        </Card>
      </div>

      <div v-if="stats" class="grid grid-cols-1 md:grid-cols-2 gap-4 mt-4">
          <Card>
            <CardHeader class="pb-2">
              <CardTitle class="text-base text-blue-600">执行环境</CardTitle>
            </CardHeader>
            <CardContent>
              <dl class="space-y-1 text-sm">
                <div class="flex justify-between border-b border-border/50 pb-1"><dt class="text-muted-foreground">Go 版本</dt><dd class="font-medium">{{ stats.env.go_version }}</dd></div>
                <div class="flex justify-between border-b border-border/50 pb-1"><dt class="text-muted-foreground">系统 / 架构</dt><dd class="font-medium">{{ stats.env.os }} / {{ stats.env.arch }}</dd></div>
                <div class="flex justify-between border-b border-border/50 pb-1"><dt class="text-muted-foreground">逻辑 CPU 数量</dt><dd class="font-medium">{{ stats.env.num_cpu }}</dd></div>
                <div class="flex justify-between border-b border-border/50 pb-1"><dt class="text-muted-foreground">当前协程数 (Goroutines)</dt><dd class="font-bold text-blue-600">{{ stats.env.goroutines }}</dd></div>
              </dl>
            </CardContent>
          </Card>

          <Card>
            <CardHeader class="pb-2">
              <CardTitle class="text-base text-indigo-600">任务调度</CardTitle>
            </CardHeader>
            <CardContent>
              <dl class="space-y-1 text-sm">
                <div class="flex justify-between border-b border-border/50 pb-1"><dt class="text-muted-foreground">驻留常驻任务 (Scheduled)</dt><dd class="font-medium text-indigo-600">{{ stats.scheduler.scheduled }}</dd></div>
                <div class="flex justify-between border-b border-border/50 pb-1"><dt class="text-muted-foreground">当前运行中 (Running)</dt><dd class="font-bold text-emerald-600">{{ stats.scheduler.running }}</dd></div>
                <div class="flex justify-between border-b border-border/50 pb-1"><dt class="text-muted-foreground">排队积压 (Queue Size)</dt><dd class="font-bold text-amber-500">{{ stats.scheduler.queue_size }}</dd></div>
                <div class="flex justify-between border-b border-border/50 pb-1"><dt class="text-muted-foreground">并发池限制 (Worker Count)</dt><dd class="font-medium">{{ stats.scheduler.worker_count }}</dd></div>
              </dl>
            </CardContent>
          </Card>
      </div>
      <div v-if="stats && stats.scheduler.workers" class="mt-4">
        <Card>
            <CardHeader class="pb-2">
              <CardTitle class="text-base text-amber-600">并发池 Worker 状态</CardTitle>
              <CardDescription>精确监控底层协程池调度执行情况</CardDescription>
            </CardHeader>
            <CardContent>
              <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4 mt-2">
                <div v-for="worker in stats.scheduler.workers" :key="worker.id" class="border rounded-md p-3 flex flex-col justify-between" :class="[worker.status === 'running' ? 'bg-amber-50 border-amber-200 dark:bg-amber-950/20 dark:border-amber-900/50' : 'bg-green-50 border-green-200 dark:bg-green-950/20 dark:border-green-900/50']">
                  <div class="flex items-center justify-between mb-2">
                    <span class="font-semibold text-sm">Worker #{{ worker.id }}</span>
                    <span v-if="worker.status === 'idle'" class="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-green-100 text-green-800 dark:bg-green-900/40 dark:text-green-300">
                      <StatusDot state="online" class="mr-1.5" />
                      空闲 (Idle)
                    </span>
                    <span v-else class="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-amber-100 text-amber-800 dark:bg-amber-900/40 dark:text-amber-300">
                      <StatusDot state="running" class="mr-1.5" />
                      执行中 (Running)
                    </span>
                  </div>
                  <div v-if="worker.status === 'running'" class="text-xs text-muted-foreground mt-1 space-y-1">
                    <div class="truncate" :title="worker.task_name"><span class="font-medium text-foreground">任务:</span> {{ worker.task_name || worker.task_id }}</div>
                    <div><span class="font-medium text-foreground">运行时间:</span> {{ worker.duration || 0 }}s</div>
                  </div>
                  <div v-else class="text-xs text-muted-foreground mt-1">
                    当前暂无任务分配
                  </div>
                </div>
              </div>
            </CardContent>
        </Card>
      </div>

      <div v-if="stats" class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4 mt-4">
          <Card>
            <CardHeader class="pb-2">
              <CardTitle class="text-base text-emerald-600">内存概览</CardTitle>
            </CardHeader>
            <CardContent>
              <dl class="space-y-1 text-sm">
                <div class="flex justify-between border-b border-border/50 pb-1"><dt class="text-muted-foreground">当前分配内存 (Alloc)</dt><dd class="font-bold text-emerald-600">{{ formatBytes(stats.mem.alloc) }}</dd></div>
                <div class="flex justify-between border-b border-border/50 pb-1"><dt class="text-muted-foreground">累计分配内存 (TotalAlloc)</dt><dd class="font-medium">{{ formatBytes(stats.mem.total_alloc) }}</dd></div>
                <div class="flex justify-between border-b border-border/50 pb-1"><dt class="text-muted-foreground">向系统获取内存 (Sys)</dt><dd class="font-medium">{{ formatBytes(stats.mem.sys) }}</dd></div>
                <div class="flex justify-between border-b border-border/50 pb-1"><dt class="text-muted-foreground">内存分配次数 (Mallocs)</dt><dd class="font-medium">{{ stats.mem.mallocs }}</dd></div>
                <div class="flex justify-between border-b border-border/50 pb-1"><dt class="text-muted-foreground">内存释放次数 (Frees)</dt><dd class="font-medium">{{ stats.mem.frees }}</dd></div>
                <div class="flex justify-between border-b border-border/50 pb-1"><dt class="text-muted-foreground">指针查找次数 (Lookups)</dt><dd class="font-medium">{{ stats.mem.lookups }}</dd></div>
              </dl>
            </CardContent>
          </Card>

          <Card>
            <CardHeader class="pb-2">
              <CardTitle class="text-base text-purple-600">堆栈明细</CardTitle>
            </CardHeader>
            <CardContent>
              <dl class="space-y-1 text-sm">
                <div class="flex justify-between border-b border-border/50 pb-1"><dt class="text-muted-foreground">堆分配内存 (HeapAlloc)</dt><dd class="font-medium">{{ formatBytes(stats.heap.heap_alloc) }}</dd></div>
                <div class="flex justify-between border-b border-border/50 pb-1"><dt class="text-muted-foreground">堆系统内存 (HeapSys)</dt><dd class="font-medium">{{ formatBytes(stats.heap.heap_sys) }}</dd></div>
                <div class="flex justify-between border-b border-border/50 pb-1"><dt class="text-muted-foreground">空闲堆内存 (HeapIdle)</dt><dd class="font-medium">{{ formatBytes(stats.heap.heap_idle) }}</dd></div>
                <div class="flex justify-between border-b border-border/50 pb-1"><dt class="text-muted-foreground">使用中堆内存 (HeapInuse)</dt><dd class="font-medium">{{ formatBytes(stats.heap.heap_inuse) }}</dd></div>
                <div class="flex justify-between border-b border-border/50 pb-1"><dt class="text-muted-foreground">已释放堆内存 (HeapReleased)</dt><dd class="font-medium">{{ formatBytes(stats.heap.heap_released) }}</dd></div>
                <div class="flex justify-between border-b border-border/50 pb-1"><dt class="text-muted-foreground">堆对象数量 (HeapObjects)</dt><dd class="font-medium">{{ stats.heap.heap_objects }}</dd></div>
              </dl>
            </CardContent>
          </Card>

          <Card>
            <CardHeader class="pb-2">
              <CardTitle class="text-base text-orange-600">垃圾回收 (GC)</CardTitle>
            </CardHeader>
            <CardContent>
              <dl class="space-y-1 text-sm">
                <div class="flex justify-between border-b border-border/50 pb-1"><dt class="text-muted-foreground">下次 GC目标 (NextGC)</dt><dd class="font-medium">{{ formatBytes(stats.gc.next_gc) }}</dd></div>
                <div class="flex justify-between border-b border-border/50 pb-1"><dt class="text-muted-foreground">上次 GC时间 (LastGC)</dt><dd class="font-medium">{{ stats.gc.last_gc ? new Date(stats.gc.last_gc / 1000000).toLocaleString() : '-' }}</dd></div>
                <div class="flex justify-between border-b border-border/50 pb-1"><dt class="text-muted-foreground">GC停顿总时 (PauseTotal)</dt><dd class="font-medium">{{ formatNs(stats.gc.pause_total_ns) }}</dd></div>
                <div class="flex justify-between border-b border-border/50 pb-1"><dt class="text-muted-foreground">执行次数 (NumGC)</dt><dd class="font-medium">{{ stats.gc.num_gc }}</dd></div>
              </dl>
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
