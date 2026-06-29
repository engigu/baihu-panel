<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { Plus, RefreshCw, Search, Server, ArrowRightLeft, Ticket, Download, Network } from 'lucide-vue-next'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { toast } from 'vue-sonner'
import * as nodeApi from '@/api/node'

import SyncPanel from './SyncPanel.vue'
import MasterList from './MasterList.vue'
import TokenListTab from './TokenListTab.vue'
import DownloadAgentDialog from './DownloadAgentDialog.vue'
import EditTokenDialog from './EditTokenDialog.vue'

const emit = defineEmits<{
  (e: 'cancel'): void
}>()

const nodes = ref<nodeApi.NodeDTO[]>([])
const tokens = ref<nodeApi.NodeToken[]>([])
const agentVersion = ref('')
const platforms = ref<{ os: string; arch: string; filename: string }[]>([])
const loading = ref(false)

// Level 1 and Level 2 tabs
const mainTab = ref<'interconnect' | 'runner'>('interconnect')
const subTab = ref<string>('panels')

const searchQuery = ref('')
const masterListRef = ref<InstanceType<typeof MasterList> | null>(null)

// Dialog refs
const downloadAgentDialogRef = ref<InstanceType<typeof DownloadAgentDialog> | null>(null)
const editTokenDialogRef = ref<InstanceType<typeof EditTokenDialog> | null>(null)

async function fetchNodes() {
  loading.value = true
  try {
    const [nodeList, tokenList, versionInfo] = await Promise.all([
      nodeApi.getNodes(),
      nodeApi.getTokens(),
      nodeApi.getVersion().catch(() => ({ version: '', platforms: [] }))
    ])
    nodes.value = nodeList
    tokens.value = tokenList
    agentVersion.value = versionInfo.version || ''
    platforms.value = versionInfo.platforms || []
  } catch (error: any) {
    toast.error('获取节点列表失败')
  } finally {
    loading.value = false
  }
}

function openDownloadDialog() {
  downloadAgentDialogRef.value?.openDialog()
}

function openCreateToken() {
  editTokenDialogRef.value?.openCreate()
}

function openEditToken(token: nodeApi.NodeToken) {
  editTokenDialogRef.value?.openEdit(token)
}

onMounted(() => {
  fetchNodes()
})
</script>

<template>
  <div class="space-y-2">
    <!-- Level 1 Navigation Segment Control -->
    <div class="flex flex-col gap-3 sm:flex-row sm:items-center justify-between border-b border-border/40 pb-2">
      <div class="flex items-center justify-between sm:justify-start gap-3 w-full sm:w-auto">
        <div class="flex p-0.5 bg-muted/20 border border-border/30 rounded-lg gap-0.5 shadow-sm">
          <button 
            @click="mainTab = 'interconnect'; subTab = 'panels'"
            :class="['px-3 sm:px-4 py-1.5 text-xs font-semibold rounded-md transition-all flex items-center gap-1.5 cursor-pointer', 
              mainTab === 'interconnect' ? 'bg-background shadow-sm text-foreground border border-border/20' : 'text-muted-foreground hover:text-foreground']">
            <Network class="w-3.5 h-3.5" />
            <span>节点互联</span>
          </button>
          <button 
            @click="mainTab = 'runner'; subTab = 'runners'"
            :class="['px-3 sm:px-4 py-1.5 text-xs font-semibold rounded-md transition-all flex items-center gap-1.5 cursor-pointer', 
              mainTab === 'runner' ? 'bg-background shadow-sm text-foreground border border-border/20' : 'text-muted-foreground hover:text-foreground']">
            <Server class="w-3.5 h-3.5" />
            <span>Runner节点(原Agent)</span>
          </button>
        </div>
        <span class="px-2 py-0.5 rounded text-[10px] font-medium bg-amber-500/10 text-amber-500 border border-amber-500/20 shrink-0">主节点 (Master)</span>
      </div>
      
      <div class="flex items-center justify-end w-full sm:w-auto">
        <Button variant="outline" size="sm" class="h-8 shadow-sm text-destructive border-destructive/20 hover:bg-destructive/10 w-full sm:w-auto" @click="emit('cancel')">
          取消主控
        </Button>
      </div>
    </div>

    <!-- Level 2 Sub Tabs Row -->
    <div class="flex flex-col gap-2.5 sm:flex-row sm:items-center justify-between py-1 border-b border-border/20">
      <!-- Left: Tabs (Level 2 Navigation) -->
      <div class="flex items-center gap-2">
        <!-- Interconnect Sub Tabs -->
        <div v-if="mainTab === 'interconnect'" class="flex p-0.5 bg-muted/15 border border-border/10 rounded-lg gap-0.5 w-full sm:w-auto">
          <button @click="subTab = 'panels'" :class="['flex-1 sm:flex-none px-3 py-1 text-xs font-medium rounded-md transition-all cursor-pointer text-center', subTab === 'panels' ? 'bg-background shadow-sm text-foreground border border-border/10' : 'text-muted-foreground hover:text-foreground']">
            子面板列表
          </button>
          <button @click="subTab = 'sync'" :class="['flex-1 sm:flex-none px-3 py-1 text-xs font-medium rounded-md transition-all cursor-pointer text-center', subTab === 'sync' ? 'bg-background shadow-sm text-foreground border border-border/10' : 'text-muted-foreground hover:text-foreground']">
            配置同步
          </button>
        </div>
        
        <!-- Runner Sub Tabs -->
        <div v-if="mainTab === 'runner'" class="flex p-0.5 bg-muted/15 border border-border/10 rounded-lg gap-0.5 w-full sm:w-auto">
          <button @click="subTab = 'runners'" :class="['flex-1 sm:flex-none px-3 py-1 text-xs font-medium rounded-md transition-all cursor-pointer text-center', subTab === 'runners' ? 'bg-background shadow-sm text-foreground border border-border/10' : 'text-muted-foreground hover:text-foreground']">
            Runner 列表
          </button>
          <button @click="subTab = 'tokens'" :class="['px-3 py-1 text-xs font-medium rounded-md transition-all cursor-pointer text-center', subTab === 'tokens' ? 'bg-background shadow-sm text-foreground border border-border/10' : 'text-muted-foreground hover:text-foreground']">
            注册令牌
          </button>
        </div>
      </div>

      <!-- Right: Action Buttons and Search -->
      <div class="flex items-center gap-2 w-full sm:w-auto justify-between sm:justify-end">
        <!-- Search Box (only for lists) -->
        <div class="relative flex-1 min-w-[120px] sm:w-[180px] xl:w-[240px] group" v-if="subTab === 'panels' || subTab === 'runners'">
          <Search class="absolute left-2.5 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-muted-foreground group-focus-within:text-primary transition-colors" />
          <Input v-model="searchQuery" placeholder="搜索节点..." class="h-8 pl-8 w-full bg-muted/20 border-muted-foreground/10 focus:bg-background text-xs" />
        </div>

        <div class="flex items-center gap-1.5 shrink-0">
          <!-- Refresh button -->
          <Button variant="outline" size="icon" class="h-8 w-8" @click="fetchNodes" :disabled="loading" title="刷新">
            <RefreshCw class="h-3.5 w-3.5" :class="{ 'animate-spin': loading }" />
          </Button>

          <!-- Context-specific Actions -->
          <template v-if="mainTab === 'interconnect' && subTab === 'panels'">
            <Button @click="masterListRef?.openAddDialog()" class="h-8 px-3 text-xs font-medium gap-1" title="添加子面板">
              <Plus class="h-3.5 w-3.5" /><span>添加面板</span>
            </Button>
          </template>

          <template v-if="mainTab === 'runner'">
            <Button variant="outline" class="h-8 px-3 text-xs gap-1.5" @click="openDownloadDialog" v-if="subTab === 'runners'">
              <Download class="h-3.5 w-3.5" /><span>下载 Runner</span>
            </Button>
            <Button @click="openCreateToken" class="h-8 px-3 text-xs font-medium gap-1" title="生成令牌" v-if="subTab === 'tokens'">
              <Plus class="h-3.5 w-3.5" /><span>生成令牌</span>
            </Button>
          </template>
        </div>
      </div>
    </div>

    <!-- 内容区域 -->
    <div class="pt-1">
      <!-- panels list -->
      <MasterList 
        v-if="mainTab === 'interconnect' && subTab === 'panels'" 
        ref="masterListRef" 
        :nodes="nodes.filter(n => n.type === 'panel')" 
        :loading="loading" 
        :search-query="searchQuery" 
        @refresh="fetchNodes" 
      />

      <!-- config sync -->
      <SyncPanel 
        v-if="mainTab === 'interconnect' && subTab === 'sync'" 
        :nodes="nodes.filter(n => n.type === 'panel')" 
      />

      <!-- runner list -->
      <MasterList 
        v-if="mainTab === 'runner' && subTab === 'runners'" 
        ref="masterListRef" 
        :nodes="nodes.filter(n => n.type === 'runner')" 
        :loading="loading" 
        :search-query="searchQuery" 
        @refresh="fetchNodes" 
      />

      <!-- runner registration tokens -->
      <TokenListTab 
        v-if="mainTab === 'runner' && subTab === 'tokens'" 
        :tokens="tokens" 
        :search-query="searchQuery" 
        @create-token="openCreateToken" 
        @edit-token="openEditToken" 
        @refresh="fetchNodes" 
      />
    </div>

    <!-- 令牌相关弹窗 -->
    <DownloadAgentDialog ref="downloadAgentDialogRef" :agent-version="agentVersion" :platforms="platforms" />
    <EditTokenDialog ref="editTokenDialogRef" @refresh="fetchNodes" />
  </div>
</template>
