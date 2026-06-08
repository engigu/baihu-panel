<script setup lang="ts">
import { ref, onMounted, computed, onUnmounted } from 'vue'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { RefreshCw, Server, Search, Download, Ticket } from 'lucide-vue-next'
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle } from '@/components/ui/alert-dialog'
import { api, type Agent, type AgentToken } from '@/api'
import { toast } from 'vue-sonner'

import AgentListTab from './components/AgentListTab.vue'
import TokenListTab from './components/TokenListTab.vue'
import AgentDetailDialog from './components/AgentDetailDialog.vue'
import EditAgentDialog from './components/EditAgentDialog.vue'
import DownloadAgentDialog from './components/DownloadAgentDialog.vue'
import EditTokenDialog from './components/EditTokenDialog.vue'

const agents = ref<Agent[]>([])
const tokens = ref<AgentToken[]>([])
const loading = ref(false)
const searchQuery = ref('')
const activeTab = ref('agents')
const agentVersion = ref('')
const platforms = ref<{ os: string; arch: string; filename: string }[]>([])

// Dialog refs
const agentDetailDialogRef = ref<InstanceType<typeof AgentDetailDialog> | null>(null)
const editAgentDialogRef = ref<InstanceType<typeof EditAgentDialog> | null>(null)
const downloadAgentDialogRef = ref<InstanceType<typeof DownloadAgentDialog> | null>(null)
const editTokenDialogRef = ref<InstanceType<typeof EditTokenDialog> | null>(null)

// Delete Dialog state
const showDeleteDialog = ref(false)
const deletingAgent = ref<Agent | null>(null)

let refreshTimer: ReturnType<typeof setInterval> | null = null

const filteredAgents = computed(() => {
  if (!searchQuery.value) return agents.value
  const q = searchQuery.value.toLowerCase()
  return agents.value.filter(a =>
    a.name.toLowerCase().includes(q) ||
    a.hostname?.toLowerCase().includes(q) ||
    a.ip?.toLowerCase().includes(q)
  )
})

async function loadAgents() {
  loading.value = true
  try {
    const [agentList, versionInfo, tokenList] = await Promise.all([
      api.agents.list(),
      api.agents.getVersion(),
      api.agents.listTokens()
    ])
    agents.value = agentList
    agentVersion.value = versionInfo.version || ''
    platforms.value = versionInfo.platforms || []
    tokens.value = tokenList
  } catch {
    toast.error('加载失败')
  } finally {
    loading.value = false
  }
}

// Handler functions for Agent
function viewDetail(agent: Agent) {
  agentDetailDialogRef.value?.openDialog(agent)
}

function openEditDialog(agent: Agent) {
  editAgentDialogRef.value?.openDialog(agent)
}

function confirmDelete(agent: Agent) {
  deletingAgent.value = agent
  showDeleteDialog.value = true
}

async function deleteAgent() {
  if (!deletingAgent.value) return
  try {
    await api.agents.delete(deletingAgent.value.id)
    showDeleteDialog.value = false
    await loadAgents()
    toast.success('删除成功')
  } catch (e: unknown) {
    toast.error((e as Error).message || '删除失败')
  }
}

function openDownloadDialog() {
  downloadAgentDialogRef.value?.openDialog()
}

// Handler functions for Token
function openCreateToken() {
  editTokenDialogRef.value?.openCreate()
}

function openEditToken(token: AgentToken) {
  editTokenDialogRef.value?.openEdit(token)
}

onMounted(() => {
  loadAgents()
  refreshTimer = setInterval(loadAgents, 10000)
})

onUnmounted(() => {
  if (refreshTimer) clearInterval(refreshTimer)
})
</script>

<template>
  <Tabs v-model="activeTab" class="space-y-6">
    <div class="flex flex-col md:flex-row md:items-center justify-between gap-4">
      <div class="flex flex-col shrink-0">
        <h2 class="text-xl sm:text-2xl font-bold tracking-tight">Agent 管理</h2>
        <p class="text-muted-foreground text-xs mt-0.5 ml-0.5">管理远程执行代理</p>
      </div>

      <div class="flex flex-row items-center flex-wrap gap-2 w-full md:w-auto md:ml-auto md:justify-end">
        <!-- 搜索与操作 -->
        <div class="flex flex-row items-center gap-2 w-full sm:flex-1 md:flex-none md:w-auto text-sm">
          <div class="relative flex-1 md:flex-none md:w-[200px] group">
            <Search class="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground group-focus-within:text-primary transition-colors" />
            <Input v-model="searchQuery" placeholder="搜索 Agent..." class="h-9 pl-9 w-full bg-muted/20 border-muted-foreground/10 focus:bg-background text-sm" />
          </div>
          
          <Button variant="outline" size="icon" class="h-9 w-9 shrink-0" @click="loadAgents" :disabled="loading" title="刷新">
            <RefreshCw class="h-4 w-4" :class="{ 'animate-spin': loading }" />
          </Button>

          <Button variant="outline" class="h-9 px-3 shrink-0 shadow-sm" @click="openDownloadDialog">
            <Download class="h-4 w-4 md:mr-2" /> <span class="hidden md:inline">下载 Agent</span>
          </Button>
        </div>

        <TabsList class="h-9 p-0.5 bg-muted/20 border border-border/40 rounded-lg w-full sm:w-auto">
          <TabsTrigger value="agents" class="px-3 h-8 text-xs gap-1.5 font-medium transition-all flex-1 sm:flex-none">
            <Server class="w-3.5 h-3.5 opacity-70" />
            <span>列表</span>
          </TabsTrigger>
          <TabsTrigger value="regcodes" class="px-3 h-8 text-xs gap-1.5 font-medium transition-all flex-1 sm:flex-none">
            <Ticket class="w-3.5 h-3.5 opacity-70" />
            <span>令牌</span>
          </TabsTrigger>
        </TabsList>
      </div>
    </div>

    <TabsContent value="agents" class="mt-0">
      <AgentListTab 
        :agents="filteredAgents" 
        :searchQuery="searchQuery"
        @view-detail="viewDetail"
        @edit-agent="openEditDialog"
        @delete-agent="confirmDelete"
        @refresh="loadAgents"
      />
    </TabsContent>

    <TabsContent value="regcodes" class="mt-0">
      <TokenListTab 
        :tokens="tokens"
        @create-token="openCreateToken"
        @edit-token="openEditToken"
        @refresh="loadAgents"
      />
    </TabsContent>

    <!-- 弹窗组件 -->
    <AgentDetailDialog ref="agentDetailDialogRef" />
    <EditAgentDialog ref="editAgentDialogRef" @updated="loadAgents" />
    <DownloadAgentDialog ref="downloadAgentDialogRef" :agentVersion="agentVersion" :platforms="platforms" />
    <EditTokenDialog ref="editTokenDialogRef" @updated="loadAgents" />

    <!-- 删除确认对话框保持在主组件，因为它比较简单，且关联了直接的 API 删除请求 -->
    <AlertDialog v-model:open="showDeleteDialog">
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>确认删除</AlertDialogTitle>
          <AlertDialogDescription>
            确定要删除 Agent "{{ deletingAgent?.name }}" 吗？此操作无法撤销。
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>取消</AlertDialogCancel>
          <AlertDialogAction class="bg-destructive text-white hover:bg-destructive/90" @click="deleteAgent">删除
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  </Tabs>
</template>
