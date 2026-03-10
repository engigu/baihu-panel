<script setup lang="ts">
import { ref, computed } from 'vue'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { X, Plus, Shield, Terminal, Search } from 'lucide-vue-next'
import type { NotifyChannel, ChannelType, EventType, NotifyBinding, Task } from '@/api'

const props = defineProps<{
  channels: NotifyChannel[]
  channelTypes: ChannelType[]
  eventTypes: EventType[]
  bindings: NotifyBinding[]
  tasks: Task[]
}>()

const emit = defineEmits<{
  save: [bindings: Partial<NotifyBinding>[]]
  delete: [id: string]
}>()

// --- 状态管理 ---
const selectedTaskId = ref<string>('') // 默认不选中任何任务，强制用户选择
const selectedChannels = ref<Record<string, string>>({})
const taskSearchQuery = ref('')
const isTaskDropdownOpen = ref(false)

// 任务过滤逻辑
const filteredTasks = computed(() => {
  if (!taskSearchQuery.value) return props.tasks
  const q = taskSearchQuery.value.toLowerCase()
  return props.tasks.filter(t => t.name.toLowerCase().includes(q))
})

// 获取当前选择的任务显示名称
const selectedTaskDisplay = computed(() => {
  const task = props.tasks.find(t => t.id === selectedTaskId.value)
  return task ? task.name : '请选择任务'
})

function selectTask(id: string) {
  selectedTaskId.value = id
  isTaskDropdownOpen.value = false
  taskSearchQuery.value = '' // 清空搜索词
}

// 分组事件
const systemEvents = computed(() => props.eventTypes.filter(e => e.binding_type === 'system'))
const taskEvents = computed(() => props.eventTypes.filter(e => e.binding_type === 'task'))

// 获取事件的绑定渠道
function getBindings(eventType: string, isSystem: boolean): NotifyBinding[] {
  if (isSystem) {
    return props.bindings.filter(b => b.event === eventType && b.type === 'system')
  }
  return props.bindings.filter(b => b.event === eventType && b.type === 'task' && b.data_id === selectedTaskId.value)
}

// 获取渠道名称与类型标签
function getChannelName(wayId: string): string {
  const channel = props.channels.find(c => c.id === wayId)
  return channel ? channel.name : '未知渠道'
}

function getChannelTypeLabel(wayId: string): string {
  const channel = props.channels.find(c => c.id === wayId)
  if (!channel) return ''
  const type = props.channelTypes.find(t => t.type === channel.type)
  return type ? type.label : channel.type
}

// 添加渠道到事件
function addChannelToEvent(eventType: string, bindingType: 'system' | 'task') {
  const dataId = bindingType === 'task' ? selectedTaskId.value : ''
  const key = `${eventType}-${bindingType}-${dataId}`
  const channelId = selectedChannels.value[key]
  if (!channelId) return

  const newBinding: Partial<NotifyBinding> = {
    type: bindingType,
    event: eventType,
    way_id: channelId,
    data_id: dataId
  }

  emit('save', [newBinding])
  selectedChannels.value[key] = ''
}

// 获取可选的渠道
function getAvailableChannels(eventType: string, bindingType: 'system' | 'task') {
  const isSystem = bindingType === 'system'
  const boundChannelIds = getBindings(eventType, isSystem).map(b => b.way_id)
  return props.channels.filter(c => !boundChannelIds.includes(c.id) && c.enabled)
}

// 删除绑定
function removeBinding(binding: NotifyBinding) {
  emit('delete', binding.id)
}
</script>

<template>
  <div class="space-y-6">
    <!-- 系统事件卡片 -->
    <Card>
      <CardHeader>
        <div class="flex items-center gap-2">
          <Shield class="w-5 h-5 text-primary" />
          <div>
            <CardTitle>系统事件</CardTitle>
            <CardDescription>配置帐号登录、安全告警等系统级事件的通知推送</CardDescription>
          </div>
        </div>
      </CardHeader>
      <CardContent class="space-y-4">
        <div v-if="channels.length === 0"
          class="text-center py-8 text-muted-foreground border-2 border-dashed rounded-lg">
          <p class="text-sm">请先在“渠道管理”中添加通知渠道</p>
        </div>
        <div v-else class="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div v-for="event in systemEvents" :key="event.type"
            class="flex flex-col p-4 rounded-lg border bg-card hover:bg-accent/10 transition-colors">
            <div class="flex items-center justify-between mb-3">
              <div class="flex items-center gap-2">
                <span class="font-bold text-sm">{{ event.label }}</span>
                <Badge variant="outline" class="text-[10px] font-mono opacity-50">{{ event.type }}</Badge>
              </div>
            </div>

            <!-- 已绑定渠道 -->
            <div class="flex flex-wrap gap-2 mb-3 min-h-[32px] items-center">
              <template v-if="getBindings(event.type, true).length > 0">
                <div v-for="binding in getBindings(event.type, true)" :key="binding.id"
                  class="inline-flex items-center gap-1.5 px-2 py-1 rounded bg-secondary/50 border text-[11px] font-medium">
                  <span class="truncate max-w-[100px]">{{ getChannelName(binding.way_id) }}</span>
                  <button @click="removeBinding(binding)"
                    class="hover:text-destructive p-0.5 rounded-sm transition-colors">
                    <X class="w-3 h-3" />
                  </button>
                </div>
              </template>
              <span v-else class="text-[11px] text-muted-foreground italic">未绑定渠道</span>
            </div>

            <!-- 添加绑定 -->
            <div class="flex gap-2">
              <Select v-model="selectedChannels[`${event.type}-system-`]">
                <SelectTrigger class="h-8 text-xs flex-1">
                  <SelectValue placeholder="添加通知渠道" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem v-for="ch in getAvailableChannels(event.type, 'system')" :key="ch.id" :value="ch.id">
                    <div class="flex items-center gap-2">
                      <span class="text-xs">{{ ch.name }}</span>
                      <Badge variant="outline" class="text-[9px]">{{ getChannelTypeLabel(ch.id) }}</Badge>
                    </div>
                  </SelectItem>
                </SelectContent>
              </Select>
              <Button size="sm" variant="outline" class="h-8 px-2" @click="addChannelToEvent(event.type, 'system')"
                :disabled="!selectedChannels[`${event.type}-system-`]">
                <Plus class="w-3.5 h-3.5" />
              </Button>
            </div>
          </div>
        </div>
      </CardContent>
    </Card>

    <!-- 任务事件卡片 -->
    <Card>
      <CardHeader>
        <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
          <div class="flex items-center gap-2">
            <Terminal class="w-5 h-5 text-primary" />
            <div>
              <CardTitle>任务事件</CardTitle>
              <CardDescription>配置定时任务执行状态的通知行为（需选择具体任务）</CardDescription>
            </div>
          </div>

          <div class="relative w-full sm:w-auto min-w-[240px]">
            <div
              class="flex items-center gap-2 bg-transparent border border-input rounded-md px-3 h-9 focus-within:ring-1 focus-within:ring-ring/30 transition-all cursor-pointer"
              @click="isTaskDropdownOpen = !isTaskDropdownOpen">
              <Search class="w-4 h-4 text-muted-foreground shrink-0" />
              <div class="flex-1 text-xs truncate">
                <span v-if="!isTaskDropdownOpen" class="font-medium"
                  :class="{ 'text-muted-foreground': !selectedTaskId }">{{ selectedTaskDisplay }}</span>
                <input v-else v-model="taskSearchQuery"
                  class="w-full bg-transparent border-none outline-none p-0 text-xs" placeholder="搜索任务名..." @click.stop
                  autofocus />
              </div>
              <X v-if="taskSearchQuery || isTaskDropdownOpen"
                class="w-3 h-3 text-muted-foreground hover:text-destructive"
                @click.stop="taskSearchQuery = ''; isTaskDropdownOpen = false" />
            </div>

            <!-- 自定义下拉列表 -->
            <div v-if="isTaskDropdownOpen"
              class="absolute top-full left-0 w-full mt-1 bg-card border border-border rounded-md shadow-xl z-50 max-h-[250px] overflow-auto py-1 animate-in fade-in zoom-in-95 duration-150">
              <div v-if="filteredTasks.length === 0"
                class="px-3 py-4 text-center text-[10px] text-muted-foreground italic">
                未找到相关任务
              </div>
              <div v-for="task in filteredTasks" :key="task.id"
                class="px-3 py-1.5 text-xs hover:bg-accent cursor-pointer flex items-center justify-between"
                :class="{ 'bg-accent/50 font-medium': selectedTaskId === task.id }" @click="selectTask(task.id)">
                <span class="truncate">{{ task.name }}</span>
                <Badge v-if="selectedTaskId === task.id" variant="outline" class="h-4 p-0 px-1 text-[8px]">当前</Badge>
              </div>
            </div>
          </div>
        </div>
      </CardHeader>
      <CardContent class="space-y-4">
        <!-- 未选择任务时的提示 -->
        <div v-if="!selectedTaskId"
          class="flex flex-col items-center justify-center py-12 text-muted-foreground border-2 border-dashed rounded-lg bg-accent/5">
          <Terminal class="w-10 h-10 mb-3 opacity-20" />
          <p class="text-sm font-medium">请在右上角选择需要配置的任务</p>
          <p class="text-[11px] opacity-70 mt-1">每个任务可以独立配置通知渠道</p>
        </div>

        <template v-else>
          <!-- 任务特定模式提示 -->
          <div class="flex items-center justify-between p-3 rounded-lg bg-primary/5 border border-primary/20">
            <div class="flex items-center gap-2">
              <Badge class="bg-primary/20 text-primary border-none text-[10px]">当前任务</Badge>
              <span class="text-xs font-bold truncate max-w-[200px]">{{tasks.find(t => t.id === selectedTaskId)?.name
                }}</span>
            </div>
          </div>

          <div v-if="channels.length === 0"
            class="text-center py-8 text-muted-foreground border-2 border-dashed rounded-lg">
            <p class="text-sm">请先添加通知渠道</p>
          </div>
          <div v-else class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            <div v-for="event in taskEvents" :key="event.type"
              class="flex flex-col p-4 rounded-lg border bg-card hover:bg-accent/10 transition-colors">
              <div class="flex items-center gap-2 mb-3">
                <span class="font-bold text-sm">{{ event.label }}</span>
                <Badge variant="outline" class="text-[10px] font-mono opacity-50">{{ event.type }}</Badge>
              </div>

              <!-- 已绑定渠道 -->
              <div class="flex flex-wrap gap-2 mb-3 min-h-[32px] items-center">
                <template v-if="getBindings(event.type, false).length > 0">
                  <div v-for="binding in getBindings(event.type, false)" :key="binding.id"
                    class="inline-flex items-center gap-1.5 px-2 py-1 rounded bg-secondary/50 border text-[11px] font-medium">
                    <span class="truncate max-w-[100px]">{{ getChannelName(binding.way_id) }}</span>
                    <button @click="removeBinding(binding)"
                      class="hover:text-destructive p-0.5 rounded-sm transition-colors">
                      <X class="w-3 h-3" />
                    </button>
                  </div>
                </template>
                <span v-else class="text-[11px] text-muted-foreground italic">未绑定渠道</span>
              </div>

              <!-- 添加绑定 -->
              <div class="flex gap-2 font-sans">
                <Select v-model="selectedChannels[`${event.type}-task-${selectedTaskId}`]">
                  <SelectTrigger class="h-8 text-xs flex-1">
                    <SelectValue placeholder="添加通知渠道" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem v-for="ch in getAvailableChannels(event.type, 'task')" :key="ch.id" :value="ch.id">
                      <div class="flex items-center gap-2">
                        <span class="text-xs">{{ ch.name }}</span>
                        <Badge variant="outline" class="text-[9px]">{{ getChannelTypeLabel(ch.id) }}</Badge>
                      </div>
                    </SelectItem>
                  </SelectContent>
                </Select>
                <Button size="sm" variant="outline" class="h-8 px-2" @click="addChannelToEvent(event.type, 'task')"
                  :disabled="!selectedChannels[`${event.type}-task-${selectedTaskId}`]">
                  <Plus class="w-3.5 h-3.5" />
                </Button>
              </div>
            </div>
          </div>
        </template>
      </CardContent>
    </Card>
  </div>
</template>
