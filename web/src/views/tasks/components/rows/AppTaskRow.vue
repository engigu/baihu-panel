<script setup lang="ts">
import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
  DropdownMenuSeparator,
} from '@/components/ui/dropdown-menu'
import { Package, Pencil, Trash2, ListTodo, Zap, ZapOff, Play, ScrollText, Loader2, MoreHorizontal } from 'lucide-vue-next'
import StatusDot from '@/components/StatusDot.vue'
import type { Task, Agent } from '@/api'

const props = defineProps<{
  task: Task
  index: number
  total: number
  currentPage: number
  pageSize: number
  executingTaskId?: string | null
  agentMap?: Record<string, Agent>
}>()

const emit = defineEmits<{
  'run': [id: string]
  'viewLogs': [id: string]
  'filterByApp': [appId: string]
  'editApp': [task: Task]
  'uninstallApp': [task: Task]
  'toggleTask': [task: Task, enabled: boolean]
}>()

function getAppConfig(task: Task) {
  if (!task || !task.unified_config) return {}
  try {
    const unified = JSON.parse(task.unified_config)
    return unified.app || {}
  } catch (e) {
    return {}
  }
}
</script>

<template>
  <div>
    <!-- ========== 1. 大屏/中屏 (Desktop/Tablet >= 640px) ========== -->
    <div class="hidden sm:flex items-center gap-2 px-4 py-2 hover:bg-muted/30 transition-colors bg-emerald-500/5 border-l-2 border-emerald-500">
      <StatusDot
        :state="task.running_status === 'running' ? 'running' : (task.running_status === 'queued' || task.running_status === 'pending' ? 'pending' : 'none')"
        :title="task.running_status === 'running' ? '运行中' : (task.running_status === 'queued' || task.running_status === 'pending' ? '排队中' : '')"
      />
      
      <div class="w-12 shrink-0 text-muted-foreground tabular-nums text-[11px]">
        #{{ total - (currentPage - 1) * pageSize - index }}
      </div>

      <span class="w-8 shrink-0 flex justify-center" title="白虎应用">
        <Package class="h-4 w-4 text-emerald-500" />
      </span>

      <div class="w-44 lg:w-56 shrink-0 flex flex-col justify-center gap-0.5 overflow-hidden">
        <div class="flex items-center gap-1.5 overflow-hidden">
          <span
            class="font-bold text-foreground truncate cursor-pointer hover:text-primary transition-colors flex-1 min-w-0"
            :title="task.name"
            @click="$emit('filterByApp', task.id)"
          >
            {{ task.name }}
          </span>
        </div>
        <div class="text-[10px] text-muted-foreground truncate">
          {{ getAppConfig(task).category || '应用实体' }} • {{ getAppConfig(task).author || '白虎应用' }}
        </div>
      </div>

      <!-- 应用场景 -->
      <div class="flex-1 min-w-0 text-xs text-muted-foreground flex flex-col justify-center gap-1 items-start overflow-hidden">
        <span
          v-if="getAppConfig(task).version"
          class="shrink-0 px-1.5 py-0.5 rounded bg-blue-500/10 text-blue-500 border border-blue-500/20 font-mono text-[10px] font-medium leading-none"
        >
          v{{ getAppConfig(task).version }}
        </span>
        <span
          class="px-2 py-0.5 rounded bg-emerald-500/10 text-emerald-500 border border-emerald-500/20 font-mono text-[11px] shrink-0 font-medium cursor-pointer hover:bg-emerald-500/20 transition-colors"
          :title="getAppConfig(task).description || '点击筛选该应用的任务'"
          @click="$emit('filterByApp', task.id)"
        >
          场景: {{ getAppConfig(task).current_scenario || 'standard' }}
        </span>
      </div>

      <!-- 定时规则 -->
      <div class="hidden lg:block w-24 lg:w-28 shrink-0 text-xs text-muted-foreground">
        <code v-if="task.schedule" class="text-muted-foreground text-xs bg-muted/40 px-1.5 py-0.5 rounded truncate font-mono tabular-nums">{{ task.schedule }}</code>
        <span v-else class="font-mono text-[11px] text-muted-foreground/60">主应用</span>
      </div>

      <!-- 执行 / 部署时间 -->
      <div class="hidden md:flex w-28 lg:w-36 shrink-0 flex-col justify-center gap-0.5 text-[11px] text-muted-foreground tabular-nums">
        <template v-if="task.schedule">
          <span class="truncate">上: {{ task.last_run || '-' }}</span>
          <span class="truncate">下: {{ task.next_run || '-' }}</span>
        </template>
        <template v-else>
          <span class="truncate" :title="'部署于 ' + (task.created_at || '')">部署于 {{ task.created_at ? task.created_at.split(' ')[0] : '-' }}</span>
        </template>
      </div>

      <!-- 状态 Toggle 开关 -->
      <span class="w-12 lg:w-14 flex justify-center shrink-0 cursor-pointer group" @click="$emit('toggleTask', task, !task.enabled)">
        <div v-if="task.enabled" class="h-6 w-6 rounded-md bg-emerald-500/10 flex items-center justify-center group-hover:bg-emerald-500/20">
          <Zap class="h-3.5 w-3.5 text-emerald-500 fill-emerald-500" />
        </div>
        <div v-else class="h-6 w-6 rounded-md bg-muted flex items-center justify-center group-hover:bg-muted/80">
          <ZapOff class="h-3.5 w-3.5 text-muted-foreground" />
        </div>
      </span>

      <!-- 操作区：运行、日志、编辑、更多折叠菜单 -->
      <span class="w-24 shrink-0 flex justify-center">
        <Button
          variant="ghost"
          size="icon"
          class="h-6 w-6"
          title="立即运行应用"
          :disabled="executingTaskId === task.id"
          @click="$emit('run', task.id)"
        >
          <Loader2 v-if="executingTaskId === task.id" class="h-3 w-3 animate-spin" />
          <Play v-else class="h-3 w-3" />
        </Button>
        <Button
          variant="ghost"
          size="icon"
          class="h-6 w-6"
          title="查看应用日志"
          @click="$emit('viewLogs', task.id)"
        >
          <ScrollText class="h-3 w-3" />
        </Button>
        <Button
          variant="ghost"
          size="icon"
          class="h-6 w-6 text-primary hover:bg-primary/10"
          title="编辑应用调度配置"
          @click="$emit('editApp', task)"
        >
          <Pencil class="h-3 w-3" />
        </Button>

        <DropdownMenu>
          <DropdownMenuTrigger as-child>
            <Button variant="ghost" size="icon" class="h-6 w-6" title="更多操作">
              <MoreHorizontal class="h-3 w-3" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end" class="w-32">
            <DropdownMenuItem @click="$emit('filterByApp', task.id)">
              <ListTodo class="h-3.5 w-3.5 mr-2 text-emerald-500" />
              <span>受控任务</span>
            </DropdownMenuItem>
            <DropdownMenuSeparator />
            <DropdownMenuItem class="text-destructive focus:text-destructive" @click="$emit('uninstallApp', task)">
              <Trash2 class="h-3.5 w-3.5 mr-2" />
              <span>卸载应用</span>
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </span>
    </div>

    <!-- ========== 2. 小屏 (Mobile < 640px) 卡片式布局 ========== -->
    <div class="sm:hidden p-3 hover:bg-muted/30 transition-colors border-l-2 border-emerald-500 bg-emerald-500/5 space-y-2.5">
      <div class="flex items-center justify-between gap-2 border-b border-border/40 pb-2">
        <div class="flex items-center gap-1.5 min-w-0 flex-1">
          <span class="text-[10px] text-muted-foreground tabular-nums shrink-0">#{{ total - (currentPage - 1) * pageSize - index }}</span>
          <Package class="h-3.5 w-3.5 text-emerald-500 shrink-0" />
          <span class="font-bold text-xs text-foreground truncate cursor-pointer" @click="$emit('filterByApp', task.id)">{{ task.name }}</span>
        </div>
        <div class="flex items-center gap-1 shrink-0">
          <span v-if="getAppConfig(task).version" class="text-[9px] font-mono px-1 py-px bg-blue-500/10 text-blue-500 border border-blue-500/20 rounded">
            v{{ getAppConfig(task).version }}
          </span>
          <span class="text-[9px] font-mono px-1.5 py-px bg-emerald-500/10 text-emerald-500 border border-emerald-500/20 rounded">
            {{ getAppConfig(task).current_scenario || 'standard' }}
          </span>
        </div>
      </div>

      <div class="text-[10px] text-muted-foreground/80 flex items-center justify-between px-0.5 gap-2">
        <div class="flex items-center gap-1.5 min-w-0">
          <span class="shrink-0 text-muted-foreground/60">{{ getAppConfig(task).author || '白虎应用' }}</span>
          <code v-if="task.schedule" class="text-[10px] bg-muted/60 px-1.5 py-0.5 rounded font-mono truncate text-foreground/80" :title="'定时规则: ' + task.schedule">
            {{ task.schedule }}
          </code>
          <span v-else class="text-[10px] text-muted-foreground/50 font-mono">主应用</span>
        </div>
        <span class="shrink-0 text-muted-foreground/60">{{ task.created_at ? task.created_at.split(' ')[0] : '-' }}</span>
      </div>

      <!-- 操作按钮 (与任务页小屏布局保持一致) -->
      <div class="flex items-center justify-between pt-1.5 border-t border-border/40 text-xs">
        <div class="flex items-center gap-1.5">
          <Button variant="ghost" size="sm" class="h-7 text-xs px-2" @click="$emit('run', task.id)" :disabled="executingTaskId === task.id">
            <Loader2 v-if="executingTaskId === task.id" class="h-3 w-3 mr-1 animate-spin" />
            <Play v-else class="h-3 w-3 mr-1" /> 运行
          </Button>
          <Button variant="ghost" size="sm" class="h-7 text-xs px-2" @click="$emit('viewLogs', task.id)">
            <ScrollText class="h-3 w-3 mr-1" /> 日志
          </Button>
          <Button variant="ghost" size="sm" class="h-7 text-xs px-2 text-primary" @click="$emit('editApp', task)">
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
            <DropdownMenuItem @click="$emit('filterByApp', task.id)">
              <ListTodo class="h-3.5 w-3.5 mr-2 text-emerald-500" />
              <span>受控任务</span>
            </DropdownMenuItem>
            <DropdownMenuSeparator />
            <DropdownMenuItem class="text-destructive focus:text-destructive" @click="$emit('uninstallApp', task)">
              <Trash2 class="h-3.5 w-3.5 mr-2" />
              <span>卸载应用</span>
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
    </div>
  </div>
</template>
