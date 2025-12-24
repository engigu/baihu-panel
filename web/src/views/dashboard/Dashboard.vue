<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ListTodo, Variable, Clock, Play, ScrollText } from 'lucide-vue-next'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card'
import { api, type Stats, type DailyStats, type TaskStatsItem } from '@/api'
import ApexCharts from 'apexcharts'

const router = useRouter()
const stats = ref<Stats>({ tasks: 0, today_execs: 0, envs: 0, logs: 0, scheduled: 0, running: 0 })
const sendStats = ref<DailyStats[]>([])
const taskStats = ref<TaskStatsItem[]>([])
const chartsLoaded = ref(false)
const statItems = [
  { key: 'today_execs', label: '今日执行', icon: Play, route: '/history' },
  { key: 'tasks', label: '任务总数', icon: ListTodo, route: '/tasks' },
  { key: 'envs', label: '环境变量', icon: Variable, route: '/environments' },
  { key: 'logs', label: '日志总数', icon: ScrollText, route: '/history' },
  { key: 'scheduled', label: '调度注册', icon: Clock, route: '/tasks' },
  { key: 'running', label: '正在运行', icon: Play, route: '/tasks' },
]

function navigateTo(route?: string) {
  if (route) router.push(route)
}

const renderLineChart = () => {
  const options = {
    series: [
      { name: '执行总数', data: sendStats.value.map(item => item.total) },
      { name: '执行成功', data: sendStats.value.map(item => item.success) },
      { name: '执行失败', data: sendStats.value.map(item => item.failed) }
    ],
    chart: {
      type: 'line',
      height: 240,
      toolbar: { show: false },
      background: 'transparent',
      animations: {
        enabled: true,
        easing: 'easeinout',
        speed: 800,
        animateGradually: { enabled: true, delay: 150 },
        dynamicAnimation: { enabled: true, speed: 350 }
      },
      dropShadow: {
        enabled: true,
        color: '#000',
        top: 18,
        left: 7,
        blur: 10,
        opacity: 0.2
      }
    },
    stroke: { curve: 'smooth', width: 3, lineCap: 'round' },
    markers: {
      size: 6,
      colors: ['#3b82f6', '#10b981', '#ef4444'],
      strokeColors: '#fff',
      strokeWidth: 2,
      hover: { size: 8, sizeOffset: 3 }
    },
    xaxis: {
      categories: sendStats.value.map(item => item.day.slice(5)),
      axisBorder: { show: false },
      axisTicks: { show: false },
      labels: { style: { colors: '#64748b', fontSize: '12px', fontFamily: 'Inter, sans-serif' } }
    },
    yaxis: {
      labels: {
        style: { colors: '#64748b', fontSize: '12px', fontFamily: 'Inter, sans-serif' },
        formatter: (val: number) => val + ' 次'
      }
    },
    colors: ['#3b82f6', '#10b981', '#ef4444'],
    fill: {
      type: 'gradient',
      gradient: {
        shade: 'light',
        type: 'vertical',
        shadeIntensity: 0.5,
        gradientToColors: ['#60a5fa', '#34d399', '#f87171'],
        inverseColors: false,
        opacityFrom: 0.8,
        opacityTo: 0.1,
        stops: [0, 100]
      }
    },
    grid: {
      borderColor: '#e2e8f0',
      strokeDashArray: 3,
      xaxis: { lines: { show: false } },
      yaxis: { lines: { show: true } },
      padding: { top: 0, right: 0, bottom: 0, left: 0 }
    },
    legend: {
      position: 'top',
      horizontalAlign: 'right',
      floating: true,
      offsetY: -25,
      offsetX: -5,
      fontSize: '12px',
      fontFamily: 'Inter, sans-serif',
      markers: { width: 8, height: 8, radius: 4 }
    },
    tooltip: {
      enabled: true,
      shared: true,
      intersect: false,
      theme: document.documentElement.classList.contains('dark') ? 'dark' : 'light',
      style: { fontSize: '12px', fontFamily: 'Inter, sans-serif' },
      x: { show: true },
      marker: { show: true },
      custom: ({ series, dataPointIndex, w }: { series: number[][], dataPointIndex: number, w: any }) => {
        const total = series[0]?.[dataPointIndex] ?? 0
        const success = series[1]?.[dataPointIndex] ?? 0
        const failed = series[2]?.[dataPointIndex] ?? 0
        const rate = total > 0 ? ((success / total) * 100).toFixed(1) : '0.0'
        return `<div class="bg-card text-foreground p-3 rounded-lg shadow-lg border border-border">
          <div class="font-medium mb-2">${w.globals.categoryLabels[dataPointIndex]}</div>
          <div class="space-y-1">
            <div class="flex items-center justify-between gap-4">
              <span class="flex items-center"><span class="w-2 h-2 bg-blue-500 rounded-full mr-2"></span><span class="text-sm text-muted-foreground">总数:</span></span>
              <span class="text-sm font-medium">${total} 次</span>
            </div>
            <div class="flex items-center justify-between gap-4">
              <span class="flex items-center"><span class="w-2 h-2 bg-green-500 rounded-full mr-2"></span><span class="text-sm text-muted-foreground">成功:</span></span>
              <span class="text-sm font-medium">${success} 次</span>
            </div>
            <div class="flex items-center justify-between gap-4">
              <span class="flex items-center"><span class="w-2 h-2 bg-red-500 rounded-full mr-2"></span><span class="text-sm text-muted-foreground">失败:</span></span>
              <span class="text-sm font-medium">${failed} 次</span>
            </div>
            <div class="border-t border-border pt-1 mt-2">
              <div class="flex items-center justify-between">
                <span class="text-sm text-muted-foreground">成功率:</span>
                <span class="text-sm font-medium">${rate}%</span>
              </div>
            </div>
          </div>
        </div>`
      }
    },
    responsive: [{ breakpoint: 768, options: { chart: { height: 200 }, legend: { position: 'bottom', offsetY: 0 } } }]
  }
  new ApexCharts(document.querySelector("#stats-chart"), options).render()
}

const renderPieChart = () => {
  if (taskStats.value.length === 0) return
  const options = {
    series: taskStats.value.map(item => item.count),
    chart: {
      type: 'pie',
      height: 240,
      toolbar: { show: false },
      background: 'transparent',
      animations: {
        enabled: true,
        easing: 'easeinout',
        speed: 800,
        animateGradually: { enabled: true, delay: 150 },
        dynamicAnimation: { enabled: true, speed: 350 }
      }
    },
    labels: taskStats.value.map(item => item.task_name),
    colors: ['#3b82f6', '#10b981', '#f59e0b', '#ef4444', '#8b5cf6', '#ec4899', '#06b6d4', '#84cc16'],
    legend: {
      position: 'bottom',
      fontSize: '12px',
      fontFamily: 'Inter, sans-serif',
      markers: { width: 8, height: 8, radius: 4 }
    },
    plotOptions: {
      pie: {
        donut: { size: '0%' },
        expandOnClick: true
      }
    },
    dataLabels: {
      enabled: true,
      formatter: (val: number) => val.toFixed(1) + '%',
      style: { fontSize: '10px', fontFamily: 'Inter, sans-serif', fontWeight: 'bold' }
    },
    tooltip: {
      enabled: true,
      theme: document.documentElement.classList.contains('dark') ? 'dark' : 'light',
      style: { fontSize: '12px', fontFamily: 'Inter, sans-serif' },
      y: { formatter: (val: number) => val + ' 次' }
    },
    responsive: [{ breakpoint: 768, options: { chart: { height: 200 }, legend: { position: 'bottom' } } }]
  }
  new ApexCharts(document.querySelector("#pie-chart"), options).render()
}

onMounted(async () => {
  try {
    const [statsData, sendStatsData, taskStatsData] = await Promise.all([
      api.dashboard.stats(),
      api.dashboard.sendStats(),
      api.dashboard.taskStats()
    ])
    stats.value = statsData
    sendStats.value = sendStatsData
    taskStats.value = taskStatsData
    setTimeout(() => {
      renderLineChart()
      renderPieChart()
      chartsLoaded.value = true
    }, 100)
  } catch {}
})
</script>

<template>
  <div class="space-y-4">
    <div>
      <h2 class="text-2xl font-bold tracking-tight">数据仪表</h2>
      <p class="text-muted-foreground">查看系统运行状态和统计数据</p>
    </div>

    <div class="grid gap-4 grid-cols-2 sm:grid-cols-3 lg:grid-cols-6">
      <Card v-for="item in statItems" :key="item.key" class="cursor-pointer hover:bg-accent/50 transition-colors" @click="navigateTo(item.route)">
        <CardHeader class="flex flex-row items-center justify-between space-y-0 pb-2">
          <CardTitle class="text-xs sm:text-sm font-medium">{{ item.label }}</CardTitle>
          <component :is="item.icon" class="h-4 w-4 text-muted-foreground" />
        </CardHeader>
        <CardContent>
          <div class="text-xl sm:text-2xl font-bold">{{ stats[item.key as keyof Stats] }}</div>
        </CardContent>
      </Card>
    </div>

    <div class="grid gap-4 lg:grid-cols-10">
      <Card class="lg:col-span-7 h-[300px] sm:h-[340px]">
        <CardHeader class="pb-2">
          <CardTitle class="text-base sm:text-lg">执行统计</CardTitle>
          <CardDescription class="text-xs sm:text-sm">最近30天任务执行情况</CardDescription>
        </CardHeader>
        <CardContent>
          <div id="stats-chart" class="w-full h-[200px] sm:h-[240px]">
            <div v-if="!chartsLoaded" class="h-full flex items-center justify-center text-muted-foreground text-sm">
              加载中...
            </div>
          </div>
        </CardContent>
      </Card>

      <Card class="lg:col-span-3 h-[300px] sm:h-[340px]">
        <CardHeader class="pb-2">
          <CardTitle class="text-base sm:text-lg">任务占比</CardTitle>
          <CardDescription class="text-xs sm:text-sm">最近30天任务执行分布</CardDescription>
        </CardHeader>
        <CardContent>
          <div id="pie-chart" class="w-full h-[200px] sm:h-[240px]">
            <div v-if="!chartsLoaded" class="h-full flex items-center justify-center text-muted-foreground text-sm">
              加载中...
            </div>
            <div v-else-if="taskStats.length === 0" class="h-full flex items-center justify-center text-muted-foreground text-sm">
              暂无数据
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  </div>
</template>
