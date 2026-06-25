<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { Select, SelectContent, SelectGroup, SelectItem, SelectTrigger } from '@/components/ui/select'
import { MonitorDot } from 'lucide-vue-next'
import { request, activeInterconnectNodeId, setActiveInterconnectNodeId, api } from '@/api'
import type { InterconnectNode } from '@/api/interconnect'
import { useEventBus } from '@vueuse/core'

const nodes = ref<InterconnectNode[]>([])
const selectedNodeId = ref<string>('local')
const isMaster = ref(false)

let timer: any = null

async function loadNodes() {
  try {
    const res = await request<InterconnectNode[]>('/interconnect/nodes')
    nodes.value = res || []
  } catch (e) {
    console.error('Failed to load interconnect nodes', e)
  }
}

function handleNodeChange(val: any) {
  if (!val) return
  const strVal = String(val)
  selectedNodeId.value = strVal
  if (strVal === 'local') {
    setActiveInterconnectNodeId('')
  } else {
    setActiveInterconnectNodeId(strVal)
  }
  // Refresh the current page to reload data from the new node
  window.location.reload()
}

function startPolling() {
  stopPolling()
  loadNodes()
  timer = setInterval(loadNodes, 30000)
}

function stopPolling() {
  if (timer) {
    clearInterval(timer)
    timer = null
  }
}

async function checkRoleAndInit() {
  try {
    const role = await api.settings.get('interconnect', 'interconnect_role')
    updateRoleState(role)
  } catch (error) {
    updateRoleState('none')
  }
}

function updateRoleState(role: string) {
  const master = role === 'master'
  isMaster.value = master
  if (master) {
    if (activeInterconnectNodeId) {
      selectedNodeId.value = activeInterconnectNodeId
    } else {
      selectedNodeId.value = 'local'
    }
    startPolling()
  } else {
    stopPolling()
    nodes.value = []
    selectedNodeId.value = 'local'
    // 只有在非“穿越状态”下，才可以因为角色改变而清除 LocalStorage/Cookie
    const isTraveling = !!document.cookie.match(new RegExp('(^| )active_interconnect_node_id=([^;]*)'))
    if (activeInterconnectNodeId && !isTraveling) {
      setActiveInterconnectNodeId('')
    }
  }
}

const roleBus = useEventBus<string>('interconnect-role-changed')
roleBus.on((newRole) => {
  updateRoleState(newRole)
})

onMounted(async () => {
  await checkRoleAndInit()
})

onUnmounted(() => {
  stopPolling()
})
</script>

<template>
  <div v-if="isMaster" class="flex items-center gap-2">
    <Select :model-value="selectedNodeId" @update:model-value="handleNodeChange">
      <SelectTrigger class="h-6 px-1.5 py-0 text-xs font-medium rounded-md border border-input/60 bg-background/50 hover:bg-accent hover:text-accent-foreground shadow-sm transition-all focus:ring-0 focus:ring-offset-0 w-auto gap-0.5 min-w-[70px] max-w-[120px]" :class="{'bg-amber-500/10 text-amber-600 dark:text-amber-400 border-amber-500/25 hover:bg-amber-500/15': selectedNodeId !== 'local'}">
        <div class="flex items-center gap-0.5 min-w-0">
          <MonitorDot class="h-2.5 w-2.5 shrink-0" />
          <span class="truncate pr-0.5 text-[10px] leading-none">
            {{ selectedNodeId === 'local' ? '本机节点' : (nodes.find(n => n.id === selectedNodeId)?.name || '未知子节点') }}
          </span>
        </div>
      </SelectTrigger>
      <SelectContent>
        <SelectGroup>
          <SelectItem value="local" class="text-xs">
            本机节点
          </SelectItem>
          <SelectItem v-for="node in nodes" :key="node.id" :value="node.id" class="text-xs">
            {{ node.name }}
          </SelectItem>
        </SelectGroup>
      </SelectContent>
    </Select>
  </div>
</template>
