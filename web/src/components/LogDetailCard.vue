<script setup lang="ts">
import { ref, onMounted, onUnmounted, watch } from 'vue'
import { 
  X, Trash2, Maximize2, CheckCircle2, XCircle, AlertCircle, Clock, Ban, 
  Zap as ZapIcon, Search, MinusCircle
} from 'lucide-vue-next'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import LogContent from './LogContent.vue'
import { TASK_STATUS, TASK_STATUS_TEXT } from '@/constants'
import type { TaskLog } from '@/api'

interface Props {
  log: TaskLog | null
  content: string
  title?: string
  loading?: boolean
  isStopping?: boolean
  showClose?: boolean
  variant?: 'full' | 'simple'
  emptyTitle?: string
  emptyDescription?: string
}

const props = withDefaults(defineProps<Props>(), {
  log: null,
  content: '',
  title: '日志详情',
  loading: false,
  isStopping: false,
  showClose: true,
  variant: 'full',
  emptyTitle: undefined,
  emptyDescription: undefined
})

defineEmits<{
  'close': []
  'stop': []
  'delete': [id: string]
  'maximize': []
}>()

const searchKeyword = ref('')
const currentDuration = ref(props.log?.duration || 0)
let timer: ReturnType<typeof setInterval> | null = null

function formatDuration(ms: number): string {
  if (ms <= 0) return '0毫秒'
  if (ms < 1000) return `${ms.toFixed(0)}毫秒`
  if (ms < 60000) return `${(ms / 1000).toFixed(1)}秒`
  return `${(ms / 60000).toFixed(1)}分钟`
}

const startTimer = () => {
  stopTimer()
  if (!props.log || props.log.status !== TASK_STATUS.RUNNING || !props.log.start_time) return
  
  const startTime = new Date(props.log.start_time).getTime()
  timer = setInterval(() => {
    currentDuration.value = Date.now() - startTime
  }, 100)
}

const stopTimer = () => {
  if (timer) {
    clearInterval(timer)
    timer = null
  }
}

onMounted(() => {
  startTimer()
})

onUnmounted(() => {
  stopTimer()
})

watch(() => props.log?.status, (newStatus) => {
  if (newStatus === TASK_STATUS.RUNNING) {
    startTimer()
  } else {
    stopTimer()
    if (props.log) currentDuration.value = props.log.duration
  }
})

watch(() => props.log?.duration, (newVal) => {
  if (props.log?.status !== TASK_STATUS.RUNNING) {
    currentDuration.value = newVal || 0
  }
})

function getStatusBadgeClass(status: string) {
  switch (status) {
    case TASK_STATUS.SUCCESS:
      return 'bg-emerald-500/5 text-emerald-500 border-emerald-500/15 dark:bg-emerald-500/10 dark:text-emerald-400 dark:border-emerald-500/20'
    case TASK_STATUS.FAILED:
      return 'bg-rose-500/5 text-rose-500 border-rose-500/15 dark:bg-rose-500/10 dark:text-rose-400 dark:border-rose-500/20'
    case TASK_STATUS.RUNNING:
      return 'bg-sky-500/5 text-sky-500 border-sky-500/15 dark:bg-sky-500/10 dark:text-sky-400 dark:border-sky-500/20'
    case TASK_STATUS.PENDING:
      return 'bg-amber-500/5 text-amber-500 border-amber-500/15 dark:bg-amber-500/10 dark:text-amber-400 dark:border-amber-500/20'
    case TASK_STATUS.TIMEOUT:
      return 'bg-orange-500/5 text-orange-500 border-orange-500/15 dark:bg-orange-500/10 dark:text-orange-400 dark:border-orange-500/20'
    case TASK_STATUS.CANCELLED:
      return 'bg-zinc-500/5 text-zinc-400 border-zinc-500/15 dark:bg-zinc-500/10 dark:text-zinc-400 dark:border-zinc-500/20'
    case 'UNEXECUTED':
      return 'bg-zinc-500/5 text-zinc-400 border-zinc-500/15 border-dashed dark:bg-zinc-500/10 dark:text-zinc-400 dark:border-zinc-500/20'
    default:
      return 'bg-secondary text-secondary-foreground border-transparent'
  }
}
</script>

<template>
  <div v-if="log" class="w-full h-full flex flex-col overflow-hidden bg-card">
    <!-- 头部菜单 (响应式两行布局) -->
    <div class="flex flex-col border-b bg-muted/20 shrink-0">
      <!-- 第一行: 标题与核心控制按钮 -->
      <div class="flex items-center justify-between px-4 gap-4"
        :class="variant === 'full' ? 'h-[28px]' : 'h-11'">
        <div class="flex items-center gap-2 min-w-0">
          <span class="text-xs font-medium text-foreground whitespace-nowrap truncate">{{ title }}</span>
          
          <!-- Simple 模式下的状态显示 -->
          <Badge v-if="variant === 'simple'" variant="outline" :class="[
            'px-2 py-0.5 font-normal rounded-full border text-[10px] flex',
            getStatusBadgeClass(log.status)
          ]">
            {{ TASK_STATUS_TEXT[log.status] || log.status }}
          </Badge>

          <!-- 详情模式下的停止按钮 (始终在第一行进入) -->
          <Button v-if="log.status === TASK_STATUS.RUNNING && variant === 'full'" variant="destructive" size="sm"
            class="h-5 px-1.5 text-[9px] shrink-0 rounded-sm" :disabled="isStopping" @click="$emit('stop')">
            {{ isStopping ? '停止中' : '停止任务' }}
          </Button>
        </div>

        <div class="flex items-center gap-1.5 shrink-0">
          <!-- 极简模式桌面端显示的停止按钮 -->
          <Button v-if="log.status === TASK_STATUS.RUNNING && variant === 'simple'" variant="destructive" size="sm"
            class="h-5 px-1.5 text-[9px] rounded-sm hidden sm:flex" :disabled="isStopping" @click="$emit('stop')">
            {{ isStopping ? '停止中' : '停止任务' }}
          </Button>

          <div class="flex items-center">
            <Button v-if="variant === 'simple'" variant="ghost" size="icon" class="h-6 w-6 text-muted-foreground"
              title="全屏切换" @click="$emit('maximize')">
              <Maximize2 class="h-3.5 w-3.5" />
            </Button>

            <Button variant="ghost" size="icon" class="h-6 w-6 text-muted-foreground hover:text-destructive"
              title="删除该日志" @click="$emit('delete', log.id)">
              <Trash2 class="h-3.5 w-3.5" />
            </Button>
            
            <Button v-if="showClose" variant="ghost" size="icon" class="h-6 w-6" @click="$emit('close')" title="关闭">
              <X class="h-3.5 w-3.5" />
            </Button>
          </div>
        </div>
      </div>

      <!-- 第二行: 工具栏 (仅在极简模式的移动端展示) -->
      <div v-if="variant === 'simple'" 
        class="flex items-center gap-2 px-4 pb-3 sm:hidden border-t pt-2 mt-0.5 border-dashed border-muted-foreground/10">
        
        <!-- 极简模式移动端停止按钮 -->
        <Button v-if="log.status === TASK_STATUS.RUNNING" variant="destructive" size="sm"
          class="h-6 px-2 text-[10px] gap-1 flex-1 max-w-[100px] rounded-sm" :disabled="isStopping" @click="$emit('stop')">
          <Ban class="h-3 w-3" />
          {{ isStopping ? '停止中' : '停止' }}
        </Button>

        <!-- 极简模式移动端搜索框 -->
        <div class="relative flex-1">
          <Search class="absolute left-2.5 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
          <Input v-model="searchKeyword" placeholder="搜索日志内容..." class="h-8 pl-8 w-full text-sm bg-background/50 border-none shadow-inner" />
        </div>
      </div>

      <!-- 桌面端搜索栏 (Simple 模式) -->
      <div v-if="variant === 'simple'" class="hidden sm:flex items-center px-4 pb-2 -mt-1">
        <div class="relative w-64">
          <Search class="absolute left-2.5 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-muted-foreground" />
          <Input v-model="searchKeyword" placeholder="搜索内容..." class="h-7 pl-8 w-full text-xs bg-background/50" />
        </div>
      </div>
    </div>

    <!-- 任务元数据 (仅在 Full 模式下展示) -->
    <div v-if="variant === 'full'"
      class="px-4 py-3 border-b space-y-2 text-sm text-foreground/80 shrink-0 overflow-y-auto max-h-[40vh]">
      <div class="flex justify-between items-center h-6">
        <span class="text-sm font-normal text-muted-foreground">任务名称</span>
        <span class="text-xs font-normal text-muted-foreground">{{ log.task_name }}</span>
      </div>
      <div class="flex justify-between items-center h-8">
        <span class="text-sm font-normal text-muted-foreground">状态</span>
        <Badge variant="outline" :class="[
          'capitalize px-2.5 py-0.5 font-normal rounded-full border transition-all duration-300',
          getStatusBadgeClass(log.status)
        ]">
          <div class="flex items-center gap-1">
            <CheckCircle2 v-if="log.status === TASK_STATUS.SUCCESS" class="h-3 w-3 fill-emerald-500/10" />
            <XCircle v-else-if="log.status === TASK_STATUS.FAILED" class="h-3 w-3 fill-rose-500/10" />
            <span v-else-if="log.status === TASK_STATUS.RUNNING" class="relative flex h-1.5 w-1.5 mr-0.5">
              <span class="animate-ping absolute inline-flex h-full w-full rounded-full bg-sky-400 opacity-75"></span>
              <span class="relative inline-flex rounded-full h-1.5 w-1.5 bg-sky-500"></span>
            </span>
            <Clock v-else-if="log.status === TASK_STATUS.PENDING" class="h-3 w-3 fill-amber-500/10" />
            <AlertCircle v-else-if="log.status === TASK_STATUS.TIMEOUT" class="h-3 w-3 fill-orange-500/10" />
            <Ban v-else-if="log.status === TASK_STATUS.CANCELLED" class="h-3 w-3" />
            <MinusCircle v-else-if="log.status === 'UNEXECUTED'" class="h-3 w-3 opacity-40" />
            <span class="text-[10px] font-normal tracking-wide">
              {{ TASK_STATUS_TEXT[log.status] || log.status }}
            </span>
          </div>
        </Badge>
      </div>
      <div class="flex justify-between items-center h-6">
        <span class="text-sm font-normal text-muted-foreground">耗时</span>
        <span class="text-xs font-normal text-muted-foreground">{{ formatDuration(currentDuration) }}</span>
      </div>
      <div class="flex justify-between items-center h-6">
        <span class="text-sm font-normal text-muted-foreground">开始时间</span>
        <span class="text-xs font-normal text-muted-foreground">{{ log.start_time || '-' }}</span>
      </div>
      <div class="flex justify-between items-center h-6">
        <span class="text-sm font-normal text-muted-foreground">结束时间</span>
        <span class="text-xs font-normal text-muted-foreground">{{ log.end_time || '-' }}</span>
      </div>
      <div class="pt-1.5 pb-1">
        <span class="text-sm font-normal text-muted-foreground block mb-1">执行命令</span>
        <code
          class="block font-mono bg-muted/40 px-3 py-2 rounded text-xs break-all border border-muted-foreground/10 leading-relaxed overflow-y-auto max-h-24 font-normal">
          {{ log.command }}
        </code>
      </div>
    </div>

    <!-- 日志输出容器 -->
    <div class="flex-1 flex flex-col overflow-hidden"
      :class="variant === 'simple' ? 'bg-black/5 dark:bg-white/5' : 'bg-black/[0.02] dark:bg-white/[0.02]'">
      <!-- 错误信息提示 -->
      <div v-if="log.error" class="px-4 py-3 border-b bg-red-500/5 space-y-2 text-sm shrink-0">
        <div class="flex items-center gap-2 text-red-500 font-medium">
          <XCircle class="h-4 w-4" />
          <span class="font-normal">系统错误</span>
        </div>
        <code class="block font-mono bg-red-500/10 text-red-600 px-2 py-1 rounded text-xs break-all">
          {{ log.error }}
        </code>
      </div>
      
      <!-- 日志工具栏 (仅在 Full 模式展示，Simple 模式直接显示内容) -->
      <div v-if="variant === 'full'" 
        class="px-4 py-1 text-sm text-muted-foreground border-b bg-muted/20 flex items-center justify-between shrink-0">
        <span class="text-sm font-normal text-muted-foreground text-[12px]">日志输出</span>
        <Button variant="ghost" size="icon" class="h-6 w-6" @click="$emit('maximize')" title="全屏查看">
          <Maximize2 class="h-3.5 w-3.5" />
        </Button>
      </div>

      <!-- 日志列表 -->
      <div class="flex-1 overflow-auto">
        <LogContent 
          class="h-full"
          :content="content" 
          :loading="loading" 
          :empty-title="emptyTitle"
          :empty-description="emptyDescription || '此任务执行期间未产生标准输出日志'" 
        />
      </div>
    </div>
  </div>
</template>

