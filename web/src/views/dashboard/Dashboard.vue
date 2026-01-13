<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed, nextTick } from 'vue'
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
const isMobile = ref(window.innerWidth < 768)
const chartDays = computed(() => isMobile.value ? 15 : 30)

let lineChart: ApexCharts | null = null
let pieChart: ApexCharts | null = null
let themeObserver: MutationObserver | null = null
let themeChangeTimeout: number | null = null
let isInitializing = true

const statItems = [
  { key: 'today_execs', label: '今日执行', icon: Play, route: '/history' },
  { key: 'tasks', label: '任务总数', icon: ListTodo, route: '/tasks' },
  { key: 'envs', label: '环境变量', icon: Variable, route: '/environments' },
  { key: 'logs', label: '日志总数', icon: ScrollText, route: '/history' },
  { key: 'scheduled', label: '调度注册', icon: Clock, route: '/tasks' },
  { key: 'running', label: '正在运行', icon: Play, route: '/tasks' },
]

const isDark = ref(document.documentElement.classList.contains('dark'))

function getTextColor() {
  return isDark.value ? '#94a3b8' : '#64748b'
}

function getGridColor() {
  return isDark.value ? '#334155' : '#e2e8f0'
}

function handleThemeChange() {
  // 初始化期间忽略主题变化
  if (isInitializing) return
  
  const newIsDark = document.documentElement.classList.contains('dark')
  if (newIsDark !== isDark.value) {
    isDark.value = newIsDark
    chartsLoaded.value = false
    if (themeChangeTimeout) clearTimeout(themeChangeTimeout)
    themeChangeTimeout = window.setTimeout(() => {
      reloadCharts()
    }, 50)
  }
}

function navigateTo(route?: string) {
  if (route) router.push(route)
}

function handleResize() {
  const wasMobile = isMobile.value
  isMobile.value = window.innerWidth < 768
  if (wasMobile !== isMobile.value) {
    reloadCharts()
  }
}

async function reloadCharts() {
  // 销毁旧图表
  if (lineChart) {
    lineChart.destroy()
    lineChart = null
  }
  if (pieChart) {
    pieChart.destroy()
    pieChart = null
  }

  chartsLoaded.value = false

  // 重新获取数据
  const [sendStatsData, taskStatsData] = await Promise.all([
    api.dashboard.sendStats(chartDays.value),
    api.dashboard.taskStats(chartDays.value)
  ])
  sendStats.value = sendStatsData
  taskStats.value = taskStatsData

  // 等待 Vue 更新 DOM
  await nextTick()
  await new Promise(resolve => setTimeout(resolve, 100))
  
  // 检查容器是否存在再渲染
  const statsChart = document.querySelector("#stats-chart")
  const pieChartEl = document.querySelector("#pie-chart")
  
  if (statsChart && pieChartEl && statsChart.parentElement && pieChartEl.parentElement) {
    renderLineChart()
    renderPieChart()
    await nextTick()
    chartsLoaded.value = true
  }
}

const renderLineChart = () => {
  const container = document.querySelector("#stats-chart")
  if (!container || !container.parentElement) return
  
  if (lineChart) {
    lineChart.destroy()
    lineChart = null
  }
  
  // 清空容器
  container.innerHTML = ''
  
  const textColor = getTextColor()
  const gridColor = getGridColor()
  const options = {
    series: [
      { name: '执行总数', data: sendStats.value.map(item => item.total) },
      { name: '执行成功', data: sendStats.value.map(item => item.success) },
      { name: '执行失败', data: sendStats.value.map(item => item.failed) }
    ],
    chart: {
      type: 'line',
      height: 300,
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
      strokeColors: isDark.value ? '#1e293b' : '#fff',
      strokeWidth: 2,
      hover: { size: 8, sizeOffset: 3 }
    },
    xaxis: {
      categories: sendStats.value.map(item => item.day.slice(5)),
      axisBorder: { show: false },
      axisTicks: { show: false },
      labels: { style: { colors: textColor, fontSize: '12px', fontFamily: 'Inter, sans-serif' } }
    },
    yaxis: {
      labels: {
        style: { colors: textColor, fontSize: '12px', fontFamily: 'Inter, sans-serif' },
        formatter: (val: number) => String(val),
        offsetX: -15,
        minWidth: 20
      }
    },
    colors: ['#3b82f6', '#10b981', '#ef4444'],
    fill: {
      type: 'gradient',
      gradient: {
        shade: isDark.value ? 'dark' : 'light',
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
      borderColor: gridColor,
      strokeDashArray: 3,
      xaxis: { lines: { show: false } },
      yaxis: { lines: { show: true } },
      padding: { top: 0, right: 5, bottom: 0, left: 0 }
    },
    legend: {
      position: 'top',
      horizontalAlign: 'right',
      floating: true,
      offsetY: -25,
      offsetX: -5,
      fontSize: '12px',
      fontFamily: 'Inter, sans-serif',
      labels: { colors: textColor },
      markers: { width: 8, height: 8, radius: 4 }
    },
    tooltip: {
      enabled: true,
      shared: true,
      intersect: false,
      theme: isDark.value ? 'dark' : 'light',
      style: { fontSize: '12px', fontFamily: 'Inter, sans-serif' },
      x: { show: true },
      marker: { show: true },
      custom: ({ series, dataPointIndex, w }: { series: number[][], dataPointIndex: number, w: any }) => {
        const total = series[0]?.[dataPointIndex] ?? 0
        const success = series[1]?.[dataPointIndex] ?? 0
        const failed = series[2]?.[dataPointIndex] ?? 0
        const rate = total > 0 ? ((success / total) * 100).toFixed(1) : '0.0'
        return `<div class="bg-card text-foreground p-2 rounded-lg shadow-lg border border-border text-xs">
          <div class="font-medium mb-1.5">${w.globals.categoryLabels[dataPointIndex]}</div>
          <div class="space-y-0.5">
            <div class="flex items-center justify-between gap-3">
              <span class="flex items-center"><span class="w-1.5 h-1.5 bg-blue-500 rounded-full mr-1.5"></span><span class="text-muted-foreground">总数:</span></span>
              <span class="font-medium">${total} 次</span>
            </div>
            <div class="flex items-center justify-between gap-3">
              <span class="flex items-center"><span class="w-1.5 h-1.5 bg-green-500 rounded-full mr-1.5"></span><span class="text-muted-foreground">成功:</span></span>
              <span class="font-medium">${success} 次</span>
            </div>
            <div class="flex items-center justify-between gap-3">
              <span class="flex items-center"><span class="w-1.5 h-1.5 bg-red-500 rounded-full mr-1.5"></span><span class="text-muted-foreground">失败:</span></span>
              <span class="font-medium">${failed} 次</span>
            </div>
            <div class="border-t border-border pt-1 mt-1">
              <div class="flex items-center justify-between">
                <span class="text-muted-foreground">成功率:</span>
                <span class="font-medium">${rate}%</span>
              </div>
            </div>
          </div>
        </div>`
      }
    },
    responsive: [{ breakpoint: 768, options: { chart: { height: 260 }, legend: { position: 'top', horizontalAlign: 'center', floating: false, offsetY: 0 } } }]
  }
  lineChart = new ApexCharts(container, options)
  lineChart.render()
}

const renderPieChart = () => {
  if (taskStats.value.length === 0) return
  
  const container = document.querySelector("#pie-chart")
  if (!container || !container.parentElement) return
  
  if (pieChart) {
    pieChart.destroy()
    pieChart = null
  }
  
  // 清空容器
  container.innerHTML = ''
  
  const totalCount = taskStats.value.reduce((sum, item) => sum + item.count, 0)
  const textColor = getTextColor()
  const options = {
    series: taskStats.value.map(item => item.count),
    chart: {
      type: 'donut',
      height: 260,
      toolbar: { show: false },
      background: 'transparent',
      animations: {
        enabled: true,
        speed: 400,
        animateGradually: {
          enabled: false
        },
        dynamicAnimation: {
          enabled: true,
          speed: 400
        }
      }
    },
    labels: taskStats.value.map(item => item.task_name),
    colors: ['#3b82f6', '#10b981', '#f59e0b', '#ef4444', '#8b5cf6', '#ec4899', '#06b6d4', '#84cc16'],
    legend: {
      show: false
    },
    stroke: {
      show: true,
      width: 2,
      colors: [isDark.value ? '#1e293b' : '#ffffff']
    },
    plotOptions: {
      pie: {
        donut: {
          size: '50%',
          labels: {
            show: true,
            name: {
              show: true,
              fontSize: '12px',
              fontFamily: 'Inter, sans-serif',
              fontWeight: 600,
              color: textColor,
              offsetY: -10
            },
            value: {
              show: true,
              fontSize: '12px',
              fontFamily: 'Inter, sans-serif',
              fontWeight: 'bold',
              color: textColor,
              offsetY: 5,
              formatter: (val: string) => val + ' 次'
            },
            total: {
              show: true,
              showAlways: false,
              label: '总执行',
              fontSize: '12px',
              fontFamily: 'Inter, sans-serif',
              fontWeight: 600,
              color: textColor,
              formatter: () => String(totalCount) + ' 次'
            }
          }
        },
        expandOnClick: true
      }
    },
    dataLabels: {
      enabled: true,
      formatter: (val: number) => val.toFixed(1) + '%',
      style: { 
        fontSize: '11px', 
        fontFamily: 'Inter, sans-serif', 
        fontWeight: 'bold', 
        colors: ['#ffffff']
      },
      dropShadow: { 
        enabled: true,
        top: 1,
        left: 1,
        blur: 2,
        color: '#000',
        opacity: 0.5
      }
    },
    tooltip: {
      enabled: true,
      theme: isDark.value ? 'dark' : 'light',
      style: { fontSize: '12px', fontFamily: 'Inter, sans-serif' },
      y: { formatter: (val: number) => val + ' 次' }
    },
    responsive: [{ breakpoint: 768, options: { chart: { height: 260 }, legend: { position: 'bottom' } } }]
  }
  pieChart = new ApexCharts(container, options)
  pieChart.render()
}

onMounted(async () => {
  window.addEventListener('resize', handleResize)
  
  try {
    const [statsData, sendStatsData, taskStatsData] = await Promise.all([
      api.dashboard.stats(),
      api.dashboard.sendStats(chartDays.value),
      api.dashboard.taskStats(chartDays.value)
    ])
    stats.value = statsData
    sendStats.value = sendStatsData
    taskStats.value = taskStatsData
    
    // 渲染图表
    setTimeout(() => {
      renderLineChart()
      renderPieChart()
      chartsLoaded.value = true
      
      // 图表渲染完成后，延迟启动主题监听，避免初始化时的闪烁
      setTimeout(() => {
        isInitializing = false
        // 监听主题变化
        themeObserver = new MutationObserver(() => {
          handleThemeChange()
        })
        themeObserver.observe(document.documentElement, { attributes: true, attributeFilter: ['class'] })
      }, 500)
    }, 100)
  } catch {}
})

onUnmounted(() => {
  window.removeEventListener('resize', handleResize)
  if (themeObserver) {
    themeObserver.disconnect()
    themeObserver = null
  }
  if (themeChangeTimeout) {
    clearTimeout(themeChangeTimeout)
    themeChangeTimeout = null
  }
  if (lineChart) lineChart.destroy()
  if (pieChart) pieChart.destroy()
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
      <Card class="lg:col-span-7 h-[400px] sm:h-[400px]">
        <CardHeader class="pb-2">
          <CardTitle class="text-base sm:text-lg">执行统计</CardTitle>
          <CardDescription class="text-xs sm:text-sm">最近{{ chartDays }}天任务执行情况</CardDescription>
        </CardHeader>
        <CardContent class="pb-8 relative">
          <div id="stats-chart" class="w-full h-[300px] sm:h-[300px]"></div>
          <div v-if="!chartsLoaded" class="absolute inset-0 flex items-center justify-center text-muted-foreground text-sm bg-card">
            加载中...
          </div>
        </CardContent>
      </Card>

      <Card class="lg:col-span-3 h-[400px] sm:h-[400px]">
        <CardHeader class="pb-2">
          <CardTitle class="text-base sm:text-lg">任务占比</CardTitle>
          <CardDescription class="text-xs sm:text-sm">最近{{ chartDays }}天任务执行分布</CardDescription>
        </CardHeader>
        <CardContent class="pb-8 relative">
          <div id="pie-chart" class="w-full h-[300px] sm:h-[300px]"></div>
          <div v-if="!chartsLoaded" class="absolute inset-0 flex items-center justify-center text-muted-foreground text-sm bg-card">
            加载中...
          </div>
          <div v-else-if="taskStats.length === 0" class="absolute inset-0 flex items-center justify-center text-muted-foreground text-sm bg-card">
            暂无数据
          </div>
        </CardContent>
      </Card>
    </div>
  </div>
</template>
