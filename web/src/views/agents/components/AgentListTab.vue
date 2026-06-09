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
  Trash2, Pencil, Eye, ListTodo,
  Zap, ZapOff, RotateCw, MoreHorizontal, Server
} from 'lucide-vue-next'
import StatusDot from '@/components/StatusDot.vue'
import { type Agent, api } from '@/api'
import { AGENT_STATUS } from '@/constants'
import { toast } from 'vue-sonner'
import { useRouter } from 'vue-router'

const router = useRouter()

const props = defineProps<{
  agents: Agent[]
  searchQuery: string
}>()

const emit = defineEmits<{
  (e: 'view-detail', agent: Agent): void
  (e: 'edit-agent', agent: Agent): void
  (e: 'delete-agent', agent: Agent): void
  (e: 'refresh'): void
}>()

function isOnline(agent: Agent): boolean {
  return agent.status === AGENT_STATUS.ONLINE
}

function viewDetail(agent: Agent) {
  emit('view-detail', agent)
}

function openEditDialog(agent: Agent) {
  emit('edit-agent', agent)
}

function confirmDelete(agent: Agent) {
  emit('delete-agent', agent)
}

function viewTasks(agent: Agent) {
  router.push({ path: '/tasks', query: { agent_id: String(agent.id) } })
}

async function toggleEnabled(agent: Agent) {
  try {
    const newEnabled = !agent.enabled
    await api.agents.update(agent.id, {
      name: agent.name,
      description: agent.description,
      enabled: newEnabled,
      scheduler_config: agent.scheduler_config
    })
    emit('refresh')
    toast.success(`${agent.name} 已${newEnabled ? '启用' : '禁用'}`)
  } catch (e: unknown) {
    toast.error((e as Error).message || '操作失败')
  }
}

async function forceUpdate(agent: Agent) {
  try {
    await api.agents.forceUpdate(agent.id)
    toast.success('已标记强制更新')
  } catch (e: unknown) {
    toast.error((e as Error).message || '操作失败')
  }
}
</script>

<template>
  <div class="rounded-lg border bg-card overflow-hidden">
    <!-- ========== 1. 大屏布局 (Large >= 1280px) ========== -->
    <div class="hidden xl:block">
      <!-- 表头 -->
      <div class="flex items-center gap-4 px-4 py-1.5 border-b bg-muted/20 text-xs text-muted-foreground font-medium">
        <span class="w-12 shrink-0 pl-1">序号</span>
        <span class="w-48 shrink-0">名称</span>
        <span class="w-32 shrink-0">IP 地址</span>
        <span class="w-32 shrink-0">主机名</span>
        <span class="w-28 shrink-0">版本</span>
        <span class="flex-1 min-w-0">心跳时间</span>
        <span class="w-24 shrink-0 text-center">操作</span>
      </div>
      <!-- 列表 -->
      <div class="divide-y text-sm">
        <div v-if="agents.length === 0" class="text-center py-12 text-muted-foreground">
          <Server class="h-8 w-8 mx-auto mb-2 opacity-50" />
          {{ searchQuery ? '无匹配结果' : '暂无 Agent' }}
        </div>
        <div v-for="(agent, index) in agents" :key="`large-${agent.id}`"
          class="flex items-center gap-2 px-4 py-1.5 hover:bg-muted/30 transition-colors">
          <StatusDot :state="isOnline(agent) ? 'online' : 'offline'" :title="isOnline(agent) ? '在线' : '离线'" />
          <div class="w-12 shrink-0 pl-1 text-muted-foreground tabular-nums text-[11px]">#{{ agents.length - index }}</div>
          <div class="w-48 shrink-0 flex flex-col justify-center gap-0.5 overflow-hidden">
            <span class="font-medium truncate cursor-pointer hover:text-primary transition-colors" @click="viewDetail(agent)">{{ agent.name }}</span>
            <div v-if="agent.description" class="text-[10px] text-muted-foreground truncate">{{ agent.description }}</div>
          </div>
          <span class="w-32 shrink-0 text-xs text-muted-foreground truncate">{{ agent.ip || '-' }}</span>
          <span class="w-32 shrink-0 text-xs text-muted-foreground truncate">{{ agent.hostname || '-' }}</span>
          <span class="w-28 shrink-0 text-xs text-muted-foreground truncate">{{ agent.version || '-' }}</span>
          <span class="flex-1 min-w-0 text-[11px] text-muted-foreground tabular-nums truncate">
            {{ agent.last_seen || '-' }}
          </span>
          <span class="w-24 shrink-0 flex justify-center items-center">
            <span class="cursor-pointer group mr-1" @click="toggleEnabled(agent)" :title="agent.enabled ? '点击禁用' : '点击启用'">
              <div v-if="agent.enabled" class="h-6 w-6 rounded-md bg-green-500/5 flex items-center justify-center group-hover:bg-green-500/10">
                <Zap class="h-3 w-3 text-green-500 fill-green-500" />
              </div>
              <div v-else class="h-6 w-6 rounded-md bg-muted flex items-center justify-center group-hover:bg-muted/80">
                <ZapOff class="h-3 w-3 text-muted-foreground" />
              </div>
            </span>
            <Button variant="ghost" size="icon" class="h-6 w-6" @click="viewDetail(agent)" title="详情"><Eye class="h-3 w-3" /></Button>
            <Button variant="ghost" size="icon" class="h-6 w-6" @click="viewTasks(agent)" title="查看任务"><ListTodo class="h-3 w-3" /></Button>
            <Button variant="ghost" size="icon" class="h-6 w-6" @click="openEditDialog(agent)" title="编辑"><Pencil class="h-3 w-3" /></Button>
            
            <DropdownMenu>
              <DropdownMenuTrigger as-child>
                <Button variant="ghost" size="icon" class="h-6 w-6"><MoreHorizontal class="h-3 w-3" /></Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end" class="w-32">
                <DropdownMenuItem @click="forceUpdate(agent)">
                  <RotateCw class="h-3.5 w-3.5 mr-2" />
                  <span>强制更新</span>
                </DropdownMenuItem>
                <DropdownMenuSeparator />
                <DropdownMenuItem class="text-destructive focus:text-destructive" @click="confirmDelete(agent)">
                  <Trash2 class="h-3.5 w-3.5 mr-2" />
                  <span>删除 Agent</span>
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </span>
        </div>
      </div>
    </div>

    <!-- ========== 2. 中屏布局 (Medium 640px - 1280px) ========== -->
    <div class="hidden sm:block xl:hidden">
      <!-- 表头 -->
      <div class="flex items-center gap-4 px-4 py-1.5 border-b bg-muted/20 text-xs text-muted-foreground font-medium">
        <span class="w-12 shrink-0 pl-1">序号</span>
        <span class="w-48 shrink-0">名称</span>
        <span class="flex-1 min-w-0">IP 地址</span>
        <span class="w-28 shrink-0 text-center">操作</span>
      </div>
      <!-- 列表 -->
      <div class="divide-y text-sm">
        <div v-for="(agent, index) in agents" :key="`medium-${agent.id}`"
          class="flex items-center gap-2 px-4 py-2.5 hover:bg-muted/30 transition-colors">
          <StatusDot :state="isOnline(agent) ? 'online' : 'offline'" :title="isOnline(agent) ? '在线' : '离线'" />
          <div class="w-12 shrink-0 pl-1 text-muted-foreground tabular-nums text-[10px]">#{{ agents.length - index }}</div>
          <div class="w-48 shrink-0 flex flex-col justify-center gap-0.5 overflow-hidden">
            <span class="font-medium truncate">{{ agent.name }}</span>
            <div v-if="agent.description" class="text-[10px] text-muted-foreground truncate">{{ agent.description }}</div>
          </div>
          <span class="flex-1 min-w-0 text-xs text-muted-foreground truncate">{{ agent.ip || '-' }}</span>
          <div class="w-28 shrink-0 flex justify-center items-center">
            <span class="cursor-pointer group mr-1" @click="toggleEnabled(agent)" :title="agent.enabled ? '点击禁用' : '点击启用'">
              <div v-if="agent.enabled" class="h-6 w-6 rounded-md bg-green-500/5 flex items-center justify-center group-hover:bg-green-500/10">
                <Zap class="h-3 w-3 text-green-500 fill-green-500" />
              </div>
              <div v-else class="h-6 w-6 rounded-md bg-muted flex items-center justify-center group-hover:bg-muted/80">
                <ZapOff class="h-3 w-3 text-muted-foreground" />
              </div>
            </span>
            <Button variant="ghost" size="icon" class="h-6 w-6" @click="viewDetail(agent)" title="详情"><Eye class="h-3 w-3" /></Button>
            <Button variant="ghost" size="icon" class="h-6 w-6" @click="viewTasks(agent)" title="查看任务"><ListTodo class="h-3 w-3" /></Button>
            <Button variant="ghost" size="icon" class="h-6 w-6" @click="openEditDialog(agent)" title="编辑"><Pencil class="h-3 w-3" /></Button>
            <DropdownMenu>
              <DropdownMenuTrigger as-child>
                <Button variant="ghost" size="icon" class="h-6 w-6"><MoreHorizontal class="h-3 w-3" /></Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                <DropdownMenuItem @click="forceUpdate(agent)">
                  <RotateCw class="h-3.5 w-3.5 mr-2" />更新
                </DropdownMenuItem>
                <DropdownMenuSeparator />
                <DropdownMenuItem class="text-destructive" @click="confirmDelete(agent)">
                  <Trash2 class="h-3.5 w-3.5 mr-2" />删除
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
        </div>
      </div>
    </div>

    <!-- ========== 3. 小屏布局 (Small < 640px) ========== -->
    <div class="divide-y sm:hidden">
      <div v-if="agents.length === 0" class="text-sm text-muted-foreground text-center py-12">暂无 Agent</div>
      <div v-for="(agent, index) in agents" :key="`small-${agent.id}`" class="p-3 hover:bg-muted/50 transition-colors">
        <div class="flex items-start justify-between mb-3 border-b border-border/40 pb-2">
          <div class="flex items-center gap-2 flex-1 min-w-0 pr-2">
            <StatusDot :state="isOnline(agent) ? 'online' : 'offline'" :title="isOnline(agent) ? '在线' : '离线'" />
            <span class="text-[10px] text-muted-foreground tabular-nums flex-shrink-0">#{{ agents.length - index }}</span>
            <div class="flex items-center gap-1.5 min-w-0 flex-1">
              <span class="font-bold text-sm truncate" @click="viewDetail(agent)">{{ agent.name }}</span>
            </div>
          </div>
          <span @click="toggleEnabled(agent)" class="cursor-pointer">
            <div v-if="agent.enabled" class="h-6 w-6 rounded-md bg-green-500/10 flex items-center justify-center">
              <Zap class="h-3.5 w-3.5 text-green-500 fill-green-500" />
            </div>
            <div v-else class="h-6 w-6 rounded-md bg-muted flex items-center justify-center">
              <ZapOff class="h-3.5 w-3.5 text-muted-foreground" />
            </div>
          </span>
        </div>
        <!-- 详情信息 -->
        <div class="space-y-1.5 text-xs text-muted-foreground mb-3 px-1">
          <div class="flex items-center gap-3">
            <span class="w-10 shrink-0 font-medium opacity-70">IP:</span>
            <span class="flex-1 truncate text-foreground">{{ agent.ip || '-' }}</span>
          </div>
          <div class="flex items-center gap-3">
            <span class="w-10 shrink-0 font-medium opacity-70">主机:</span>
            <span class="flex-1 truncate">{{ agent.hostname || '-' }}</span>
          </div>
          <div v-if="agent.description" class="flex items-start gap-3">
            <span class="w-10 shrink-0 font-medium mt-0.5 opacity-70">描述:</span>
            <span class="flex-1 text-[11px] line-clamp-1">{{ agent.description }}</span>
          </div>
        </div>
        <div class="grid grid-cols-4 items-center pt-2 mt-2 border-t border-border/40 -mx-1">
          <Button variant="ghost" class="h-9 px-0 text-xs gap-1.5 hover:bg-primary/5 rounded-none" @click="viewDetail(agent)">
            <Eye class="h-3.5 w-3.5" />详情
          </Button>
          <Button variant="ghost" class="h-9 px-0 text-xs gap-1.5 hover:bg-primary/5 rounded-none border-l border-border/10" @click="viewTasks(agent)">
            <ListTodo class="h-3.5 w-3.5" />任务
          </Button>
          <Button variant="ghost" class="h-9 px-0 text-xs gap-1.5 hover:bg-primary/5 rounded-none border-l border-border/10" @click="openEditDialog(agent)">
            <Pencil class="h-3.5 w-3.5" />编辑
          </Button>
          <DropdownMenu>
            <DropdownMenuTrigger as-child>
              <Button variant="ghost" class="h-9 px-0 text-xs gap-1.5 hover:bg-primary/5 rounded-none border-l border-border/10 w-full">
                <MoreHorizontal class="h-3.5 w-3.5" />更多
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end" class="w-40">
              <DropdownMenuItem @click="forceUpdate(agent)">
                <RotateCw class="h-4 w-4 mr-2" />更新 Agent
              </DropdownMenuItem>
              <DropdownMenuSeparator />
              <DropdownMenuItem class="text-destructive" @click="confirmDelete(agent)">
                <Trash2 class="h-4 w-4 mr-2" />删除 Agent
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </div>
    </div>
  </div>
</template>
