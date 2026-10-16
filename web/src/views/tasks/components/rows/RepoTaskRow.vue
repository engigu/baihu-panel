<script setup lang="ts">
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
  DropdownMenuSeparator,
} from '@/components/ui/dropdown-menu'
import {
  Play, Pencil, Trash2, ScrollText, GitBranch, Terminal, Monitor, Loader2, Wifi, WifiOff,
  Zap, ZapOff, Copy, Pin, PinOff, MoreHorizontal
} from 'lucide-vue-next'
import StatusDot from '@/components/StatusDot.vue'
import TextOverflow from '@/components/TextOverflow.vue'
import type { Task, Agent } from '@/api'
import { AGENT_STATUS, TASK_TYPE, TASK_TYPE_CONFIG } from '@/constants'
import { formatDateTime } from '@/utils/date'

const props = defineProps<{
  task: Task
  index: number
  total: number
  currentPage: number
  pageSize: number
  executingTaskId?: string | null
  agentMap: Record<string, Agent>
  selected?: boolean
  hasSelection?: boolean
}>()

const emit = defineEmits<{
  'run': [id: string]
  'viewLogs': [id: string]
  'edit': [task: Task]
  'toggleTask': [task: Task, enabled: boolean]
  'togglePin': [task: Task]
  'exportCmd': [task: Task]
  'duplicate': [task: Task]
  'delete': [id: string]
  'toggleSelect': [id: string]
}>()

function getExecutorName(task: Task): string {
  if (!task.agent_id) return '本地'
  const agent = props.agentMap[task.agent_id]
  return agent ? agent.name : `Agent #${task.agent_id}`
}

function getExecutorStatus(task: Task): 'local' | 'online' | 'offline' {
  if (!task.agent_id) return 'local'
  const agent = props.agentMap[task.agent_id]
  return agent?.status === AGENT_STATUS.ONLINE ? 'online' : 'offline'
}

function getRepoConfig(task: Task) {
  if (!task || !task.unified_config) return {}
  try {
    const unified = JSON.parse(task.unified_config)
    return unified.repo || {}
  } catch (e) {
    return {}
  }
}
</script>

<template>
  <div>
    <!-- ========== 1. 大屏/中屏 (Desktop/Tablet >= 640px) ========== -->
    <div
      class="hidden sm:flex items-center gap-2 px-4 py-1.5 transition-colors group"
      :class="selected ? 'bg-primary/[0.07] border-l-2 border-primary' : 'hover:bg-muted/30'"
    >
      <StatusDot
        :state="task.running_status === 'running' ? 'running' : (task.running_status === 'queued' || task.running_status === 'pending' ? 'pending' : 'none')"
        :title="task.running_status === 'running' ? '运行中' : (task.running_status === 'queued' || task.running_status === 'pending' ? '排队中' : '')"
      />

      <!-- 序号列 / 复选框 (同位替换：平时显示 #序号，悬停或激活多选时变身为复选框) -->
      <div class="w-12 shrink-0 flex items-center tabular-nums text-[11px]">
        <Checkbox
          v-if="selected || hasSelection"
          :checked="selected"
          class="shrink-0"
          @update:checked="$emit('toggleSelect', task.id)"
        />
        <template v-else>
          <span class="text-muted-foreground group-hover:hidden truncate">
            #{{ total - (currentPage - 1) * pageSize - index }}
          </span>
          <Checkbox
            :checked="false"
            class="shrink-0 hidden group-hover:flex"
            @update:checked="$emit('toggleSelect', task.id)"
          />
        </template>
      </div>

      <span class="w-8 shrink-0 flex justify-center" title="仓库同步任务">
        <GitBranch class="h-4 w-4 shrink-0" :class="TASK_TYPE_CONFIG[TASK_TYPE.REPO]?.color" />
      </span>

      <div class="w-44 lg:w-56 shrink-0 flex flex-col justify-center gap-0.5 overflow-hidden">
        <div class="flex items-center gap-1.5 overflow-hidden">
          <span class="font-medium truncate cursor-help flex-1 min-w-0" :title="task.name">{{ task.name }}</span>
          <span
            v-if="task.pin_type === 'top'"
            class="shrink-0 inline-flex items-center gap-0.5 rounded px-1 py-px text-[9px] font-mono border bg-amber-500/10 text-amber-500 border-amber-500/20 leading-none"
            title="置顶任务"
          >
            <Pin class="w-2.5 h-2.5 shrink-0 rotate-45" />
            <span>置顶</span>
          </span>
        </div>
        <div v-if="task.tags" class="flex items-center gap-1 overflow-hidden">
          <span
            v-for="tag in task.tags.split(',').filter(Boolean).slice(0, 3)"
            :key="tag"
            class="truncate text-[10px] leading-none px-1 py-0.5 bg-secondary text-secondary-foreground rounded border"
          >{{ tag }}</span>
        </div>
      </div>

      <span class="hidden md:flex w-24 lg:w-32 shrink-0 items-center gap-1 text-xs" :title="getExecutorName(task)">
        <Monitor v-if="!task.agent_id" class="h-3 w-3 text-muted-foreground" />
        <template v-else>
          <Wifi v-if="getExecutorStatus(task) === 'online'" class="h-3 w-3 text-green-500" />
          <WifiOff v-else class="h-3 w-3 text-muted-foreground" />
        </template>
        <span class="truncate">{{ getExecutorName(task) }}</span>
      </span>

      <code class="flex-1 min-w-0 text-muted-foreground truncate text-xs bg-muted/40 px-2 py-1 rounded">
        <TextOverflow :text="getRepoConfig(task).source_url || task.command" title="仓库地址" class="truncate" />
      </code>

      <div class="hidden lg:flex w-24 lg:w-28 shrink-0 flex-col items-start justify-center gap-1 overflow-hidden">
        <code v-if="task.schedule" class="text-muted-foreground text-xs bg-muted/40 px-1.5 py-0.5 rounded truncate font-mono tabular-nums">{{ task.schedule }}</code>
        <span v-else class="text-xs text-muted-foreground">-</span>
      </div>

      <div class="hidden md:flex w-28 lg:w-36 shrink-0 flex-col justify-center gap-0.5 text-[11px] text-muted-foreground tabular-nums">
        <span class="truncate" :title="task.last_run ? task.last_run : ''">上: {{ task.last_run ? formatDateTime(task.last_run) : '-' }}</span>
        <span class="truncate" :title="task.schedule && task.enabled && task.next_run ? task.next_run : ''">下: {{task.schedule && task.enabled && task.next_run ? formatDateTime(task.next_run) : '-' }}</span>
      </div>

      <span class="w-12 lg:w-14 flex justify-center shrink-0 cursor-pointer group" @click="$emit('toggleTask', task, !task.enabled)">
        <div v-if="task.enabled" class="h-6 w-6 rounded-md bg-green-500/10 flex items-center justify-center group-hover:bg-green-500/20">
          <Zap class="h-3.5 w-3.5 text-green-500 fill-green-500" />
        </div>
        <div v-else class="h-6 w-6 rounded-md bg-muted flex items-center justify-center group-hover:bg-muted/80">
          <ZapOff class="h-3.5 w-3.5 text-muted-foreground" />
        </div>
      </span>

      <span class="w-24 shrink-0 flex justify-center">
        <Button variant="ghost" size="icon" class="h-6 w-6" @click="$emit('run', task.id)" :disabled="executingTaskId === task.id">
          <Loader2 v-if="executingTaskId === task.id" class="h-3 w-3 animate-spin" />
          <Play v-else class="h-3 w-3" />
        </Button>
        <Button variant="ghost" size="icon" class="h-6 w-6" @click="$emit('viewLogs', task.id)"><ScrollText class="h-3 w-3" /></Button>
        <Button variant="ghost" size="icon" class="h-6 w-6" @click="$emit('edit', task)"><Pencil class="h-3 w-3" /></Button>

        <DropdownMenu>
          <DropdownMenuTrigger as-child>
            <Button variant="ghost" size="icon" class="h-6 w-6"><MoreHorizontal class="h-3 w-3" /></Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end" class="w-32">
            <DropdownMenuItem @click="$emit('togglePin', task)">
              <Pin v-if="task.pin_type !== 'top'" class="h-3.5 w-3.5 mr-2" />
              <PinOff v-else class="h-3.5 w-3.5 mr-2 text-primary" />
              <span>{{ task.pin_type === 'top' ? '取消置顶' : '置顶任务' }}</span>
            </DropdownMenuItem>
            <DropdownMenuItem @click="$emit('exportCmd', task)">
              <Terminal class="h-3.5 w-3.5 mr-2" />
              <span>导出指令</span>
            </DropdownMenuItem>
            <DropdownMenuItem @click="$emit('duplicate', task)">
              <Copy class="h-3.5 w-3.5 mr-2" />
              <span>复制任务</span>
            </DropdownMenuItem>
            <DropdownMenuSeparator />
            <DropdownMenuItem class="text-destructive focus:text-destructive" @click="$emit('delete', task.id)">
              <Trash2 class="h-3.5 w-3.5 mr-2" />
              <span>删除任务</span>
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </span>
    </div>

    <!-- ========== 2. 小屏 (Mobile < 640px) 卡片式布局 ========== -->
    <div
      class="sm:hidden p-3 transition-all space-y-2 group select-none"
      :class="[
        selected ? 'bg-primary/[0.07] border-l-2 border-primary' : 'hover:bg-muted/30',
        hasSelection ? 'cursor-pointer active:bg-muted/50' : ''
      ]"
      @click="hasSelection && $emit('toggleSelect', task.id)"
    >
      <div class="flex items-start justify-between gap-2 border-b border-border/40 pb-2">
        <div class="flex items-center gap-2 min-w-0 flex-1">
          <StatusDot
            :state="task.running_status === 'running' ? 'running' : (task.running_status === 'queued' || task.running_status === 'pending' ? 'pending' : 'none')"
            @click.stop
          />
          <!-- 序号 / 复选框 (移动端同位替换：平时带微药丸易于触控，多选模式下显示 Checkbox，点击整卡或序号均可切换) -->
          <div
            class="shrink-0 flex items-center justify-center cursor-pointer"
            @click.stop="$emit('toggleSelect', task.id)"
          >
            <Checkbox
              v-if="selected || hasSelection"
              :checked="selected"
              class="shrink-0"
              @update:checked="$emit('toggleSelect', task.id)"
            />
            <span
              v-else
              class="h-5 min-w-[22px] px-1 rounded bg-muted/60 hover:bg-primary/10 hover:text-primary active:scale-95 text-[10px] text-muted-foreground font-mono tabular-nums flex items-center justify-center transition-all border border-border/40"
              title="点击勾选进入多选模式"
            >
              #{{ total - (currentPage - 1) * pageSize - index }}
            </span>
          </div>
          <GitBranch class="h-4 w-4 shrink-0" :class="TASK_TYPE_CONFIG[TASK_TYPE.REPO]?.color" />
          <span class="font-bold text-sm text-foreground truncate">{{ task.name }}</span>
          <span
            v-if="task.pin_type === 'top'"
            class="shrink-0 inline-flex items-center gap-0.5 rounded px-1 py-px text-[9px] font-mono border bg-amber-500/10 text-amber-500 border-amber-500/20 leading-none"
            title="置顶任务"
          >
            <Pin class="w-2.5 h-2.5 shrink-0 rotate-45" />
            <span>置顶</span>
          </span>
        </div>
        <span @click.stop="$emit('toggleTask', task, !task.enabled)" class="cursor-pointer shrink-0">
          <div v-if="task.enabled" class="h-6 w-6 rounded-md bg-green-500/10 flex items-center justify-center">
            <Zap class="h-3.5 w-3.5 text-green-500 fill-green-500" />
          </div>
          <div v-else class="h-6 w-6 rounded-md bg-muted flex items-center justify-center">
            <ZapOff class="h-3.5 w-3.5 text-muted-foreground" />
          </div>
        </span>
      </div>

      <div class="space-y-1 text-xs text-muted-foreground">
        <div class="flex items-center gap-2 text-[11px]">
          <span class="w-10 shrink-0 font-medium opacity-70">周期:</span>
          <code v-if="task.schedule" class="text-xs bg-muted/40 px-1.5 py-0.5 rounded font-mono">{{ task.schedule }}</code>
          <span v-else class="text-xs">-</span>
        </div>
        <div class="flex items-start gap-2 text-[11px]">
          <span class="w-10 shrink-0 font-medium opacity-70">地址:</span>
          <span class="truncate flex-1 font-mono opacity-90">{{ getRepoConfig(task).source_url || task.command }}</span>
        </div>
      </div>

      <div class="flex items-center justify-between pt-1.5 border-t border-border/40 text-xs" @click.stop>
        <div class="flex items-center gap-1.5">
          <Button variant="ghost" size="sm" class="h-7 text-xs px-2" @click="$emit('run', task.id)" :disabled="executingTaskId === task.id">
            <Play class="h-3 w-3 mr-1" /> 同步
          </Button>
          <Button variant="ghost" size="sm" class="h-7 text-xs px-2" @click="$emit('viewLogs', task.id)">
            <ScrollText class="h-3 w-3 mr-1" /> 日志
          </Button>
          <Button variant="ghost" size="sm" class="h-7 text-xs px-2" @click="$emit('edit', task)">
            <Pencil class="h-3 w-3 mr-1" /> 编辑
          </Button>
        </div>

        <DropdownMenu>
          <DropdownMenuTrigger as-child>
            <Button variant="ghost" size="sm" class="h-7 px-2">
              <MoreHorizontal class="h-3.5 w-3.5" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end" class="w-32">
            <DropdownMenuItem @click="$emit('togglePin', task)">
              <span>{{ task.pin_type === 'top' ? '取消置顶' : '置顶任务' }}</span>
            </DropdownMenuItem>
            <DropdownMenuItem @click="$emit('exportCmd', task)">
              <span>导出指令</span>
            </DropdownMenuItem>
            <DropdownMenuItem @click="$emit('duplicate', task)">
              <span>复制任务</span>
            </DropdownMenuItem>
            <DropdownMenuSeparator />
            <DropdownMenuItem class="text-destructive focus:text-destructive" @click="$emit('delete', task.id)">
              <span>删除任务</span>
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
    </div>
  </div>
</template>
