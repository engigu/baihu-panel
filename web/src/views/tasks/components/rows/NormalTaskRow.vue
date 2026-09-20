<script setup lang="ts">
import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
  DropdownMenuSeparator,
} from '@/components/ui/dropdown-menu'
import {
  Play, Pencil, Trash2, ScrollText, Terminal, Monitor, Loader2, Wifi, WifiOff,
  Zap, ZapOff, Copy, Pin, PinOff, MoreHorizontal, Wrench, Package
} from 'lucide-vue-next'
import StatusDot from '@/components/StatusDot.vue'
import TextOverflow from '@/components/TextOverflow.vue'
import type { Task, Agent } from '@/api'
import { AGENT_STATUS, TRIGGER_TYPE, TASK_TYPE, TASK_TYPE_CONFIG } from '@/constants'

const props = defineProps<{
  task: Task
  index: number
  total: number
  currentPage: number
  pageSize: number
  executingTaskId?: string | null
  agentMap: Record<string, Agent>
}>()

const emit = defineEmits<{
  'run': [id: string]
  'viewLogs': [id: string]
  'edit': [task: Task]
  'toggleTask': [task: Task, enabled: boolean]
  'togglePin': [task: Task]
  'depInstall': [task: Task]
  'duplicate': [task: Task]
  'delete': [id: string]
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

function getLangBadgeClass(name: string) {
  const n = name.toLowerCase()
  const colors: Record<string, string> = {
    python: 'bg-amber-500/10 text-amber-500 border-amber-500/20 dark:bg-amber-500/10 dark:text-amber-400 dark:border-amber-500/20',
    py: 'bg-amber-500/10 text-amber-500 border-amber-500/20 dark:bg-amber-500/10 dark:text-amber-400 dark:border-amber-500/20',
    node: 'bg-emerald-500/10 text-emerald-500 border-emerald-500/20 dark:bg-emerald-500/10 dark:text-emerald-400 dark:border-emerald-500/20',
    nodejs: 'bg-emerald-500/10 text-emerald-500 border-emerald-500/20 dark:bg-emerald-500/10 dark:text-emerald-400 dark:border-emerald-500/20',
    go: 'bg-cyan-500/10 text-cyan-500 border-cyan-500/20 dark:bg-cyan-500/10 dark:text-cyan-400 dark:border-cyan-500/20',
    golang: 'bg-cyan-500/10 text-cyan-500 border-cyan-500/20 dark:bg-cyan-500/10 dark:text-cyan-400 dark:border-cyan-500/20',
    rust: 'bg-orange-500/10 text-orange-500 border-orange-500/20 dark:bg-orange-500/10 dark:text-orange-400 dark:border-orange-500/20',
    php: 'bg-violet-500/10 text-violet-500 border-violet-500/20 dark:bg-violet-500/10 dark:text-violet-400 dark:border-violet-500/20',
    ruby: 'bg-red-500/10 text-red-500 border-red-500/20 dark:bg-red-500/10 dark:text-red-400 dark:border-red-500/20',
    bun: 'bg-pink-500/10 text-pink-500 border-pink-500/20 dark:bg-pink-500/10 dark:text-pink-400 dark:border-pink-500/20',
    dotnet: 'bg-indigo-500/10 text-indigo-500 border-indigo-500/20 dark:bg-indigo-500/10 dark:text-indigo-400 dark:border-indigo-500/20'
  }
  return colors[n] || 'bg-slate-500/10 text-slate-500 border-slate-500/20 dark:bg-slate-500/10 dark:text-slate-400 dark:border-slate-500/20'
}

function getShortLangName(name: string): string {
  const n = name.toLowerCase()
  const mapping: Record<string, string> = {
    python: 'py',
    nodejs: 'node',
    golang: 'go'
  }
  return mapping[n] || n
}
</script>

<template>
  <div>
    <!-- ========== 1. 大屏/中屏 (Desktop/Tablet >= 640px) ========== -->
    <div class="hidden sm:flex items-center gap-2 px-4 py-1.5 hover:bg-muted/30 transition-colors">
      <StatusDot
        :state="task.running_status === 'running' ? 'running' : (task.running_status === 'queued' || task.running_status === 'pending' ? 'pending' : 'none')"
        :title="task.running_status === 'running' ? '运行中' : (task.running_status === 'queued' || task.running_status === 'pending' ? '排队中' : '')"
      />

      <div class="w-12 shrink-0 text-muted-foreground tabular-nums text-[11px]">
        #{{ total - (currentPage - 1) * pageSize - index }}
      </div>

      <span class="w-8 shrink-0 flex justify-center" title="普通脚本任务">
        <Package v-if="(task.source_id || '').startsWith('app:')" class="h-4 w-4 shrink-0" :class="TASK_TYPE_CONFIG[TASK_TYPE.APP]?.color" />
        <Terminal v-else class="h-4 w-4 shrink-0" :class="TASK_TYPE_CONFIG[TASK_TYPE.NORMAL]?.color" />
      </span>

      <div class="w-44 lg:w-56 shrink-0 flex flex-col justify-center gap-0.5 overflow-hidden">
        <div class="flex items-center gap-1.5 overflow-hidden">
          <span class="font-medium truncate cursor-help flex-1 min-w-0" :title="task.name">{{ task.name }}</span>
          <span v-if="(task.source_id || '').startsWith('app:')" class="shrink-0 inline-flex items-center rounded px-1 py-px text-[9px] font-mono border bg-emerald-500/10 text-emerald-500 border-emerald-500/20 leading-none">
            应用
          </span>
          <span
            v-for="lang in task.languages"
            :key="lang.name"
            class="shrink-0 inline-flex items-center rounded px-1 py-px text-[9px] font-mono border transition-all hover:opacity-90 leading-none ml-auto"
            :class="getLangBadgeClass(lang.name)"
            :title="lang.name + (lang.version ? '@' + lang.version : '')"
          >
            {{ getShortLangName(lang.name) }}{{ lang.version ? ':' + lang.version : '' }}
          </span>
          <Pin v-if="task.pin_type === 'top'" class="h-3 w-3 text-primary fill-primary shrink-0 rotate-45" />
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
        <TextOverflow :text="task.command" title="执行命令" class="truncate" />
      </code>

      <div class="hidden lg:flex w-24 lg:w-28 shrink-0 flex-col items-start justify-center gap-1 overflow-hidden">
        <span v-if="task.trigger_type === TRIGGER_TYPE.BAIHU_STARTUP" class="text-[10px] leading-tight bg-blue-500/10 text-blue-500 px-2 py-1 rounded-md">服务启动时</span>
        <code v-else-if="task.schedule" class="text-muted-foreground text-xs bg-muted/40 px-1.5 py-0.5 rounded truncate font-mono tabular-nums">{{ task.schedule }}</code>
      </div>

      <div class="hidden md:flex w-28 lg:w-36 shrink-0 flex-col justify-center gap-0.5 text-[11px] text-muted-foreground tabular-nums">
        <span class="truncate">上: {{ task.last_run || '-' }}</span>
        <span class="truncate">下: {{ task.next_run || '-' }}</span>
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
            <DropdownMenuItem @click="$emit('depInstall', task)">
              <Wrench class="h-3.5 w-3.5 mr-2 text-amber-500" />
              <span>补全依赖</span>
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
    <div class="sm:hidden p-3 hover:bg-muted/30 transition-colors space-y-2">
      <div class="flex items-start justify-between gap-2 border-b border-border/40 pb-2">
        <div class="flex items-center gap-2 min-w-0 flex-1">
          <StatusDot
            :state="task.running_status === 'running' ? 'running' : (task.running_status === 'queued' || task.running_status === 'pending' ? 'pending' : 'none')"
          />
          <span class="text-[10px] text-muted-foreground tabular-nums shrink-0">#{{ total - (currentPage - 1) * pageSize - index }}</span>
          <Terminal class="h-4 w-4 shrink-0" :class="TASK_TYPE_CONFIG[TASK_TYPE.NORMAL]?.color" />
          <span class="font-bold text-sm text-foreground truncate">{{ task.name }}</span>
          <Pin v-if="task.pin_type === 'top'" class="h-3 w-3 text-primary fill-primary shrink-0 rotate-45" />
        </div>
        <span @click="$emit('toggleTask', task, !task.enabled)" class="cursor-pointer shrink-0">
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
          <span class="w-10 shrink-0 font-medium opacity-70">定时:</span>
          <span v-if="task.trigger_type === TRIGGER_TYPE.BAIHU_STARTUP" class="text-[10px] bg-blue-500/10 text-blue-500 px-1.5 py-0.5 rounded">服务启动时</span>
          <code v-else-if="task.schedule" class="text-xs bg-muted/40 px-1.5 py-0.5 rounded font-mono">{{ task.schedule }}</code>
        </div>
        <div class="flex items-start gap-2 text-[11px]">
          <span class="w-10 shrink-0 font-medium opacity-70">命令:</span>
          <span class="truncate flex-1 font-mono opacity-90">{{ task.command }}</span>
        </div>
      </div>

      <div class="flex items-center justify-between pt-1.5 border-t border-border/40 text-xs">
        <div class="flex items-center gap-1.5">
          <Button variant="ghost" size="sm" class="h-7 text-xs px-2" @click="$emit('run', task.id)" :disabled="executingTaskId === task.id">
            <Play class="h-3 w-3 mr-1" /> 执行
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
            <DropdownMenuItem @click="$emit('depInstall', task)">
              <span>补全依赖</span>
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
