<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { Plus, RefreshCw, Search } from 'lucide-vue-next'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
// import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { toast } from 'vue-sonner'
import * as interconnectApi from '@/api/interconnect'

// import SyncPanel from './SyncPanel.vue'
import MasterList from './MasterList.vue'

const emit = defineEmits<{
  (e: 'cancel'): void
}>()

const nodes = ref<interconnectApi.InterconnectNode[]>([])
const loading = ref(false)
const activeTab = ref('nodes')
const searchQuery = ref('')
const masterListRef = ref<InstanceType<typeof MasterList> | null>(null)

async function fetchNodes() {
  loading.value = true
  try {
    nodes.value = await interconnectApi.getNodes()
  } catch (error: any) {
    toast.error('获取节点列表失败')
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  fetchNodes()
})
</script>

<template>
  <div class="space-y-4">
    <div class="flex flex-col md:flex-row md:items-center justify-between gap-4">
      <div class="flex flex-col shrink-0">
        <div class="flex items-center gap-2">
          <h2 class="text-xl sm:text-2xl font-bold tracking-tight">互联管理</h2>
          <span class="px-2 py-0.5 rounded text-xs font-medium bg-amber-500/10 text-amber-500 border border-amber-500/20">主节点 (Master)</span>
        </div>
        <p class="text-muted-foreground text-xs mt-0.5 ml-0.5">集中监控其他面板的状态，并可无缝穿越到子节点进行管理</p>
      </div>

      <div class="flex flex-col sm:flex-row items-center gap-2 w-full md:w-auto md:ml-auto md:justify-end">
        <!-- 搜索框 (仅节点管理时显示) -->
        <div class="relative w-full sm:w-[200px] md:w-[240px] group shrink-0" v-if="activeTab === 'nodes'">
          <Search class="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground group-focus-within:text-primary transition-colors" />
          <Input v-model="searchQuery" placeholder="搜索节点..." class="h-9 pl-9 w-full bg-muted/20 border-muted-foreground/10 focus:bg-background text-sm" />
        </div>
        
        <!-- 按钮控制组 -->
        <div class="flex items-center gap-2 w-full sm:w-auto sm:justify-end overflow-x-auto scrollbar-none pb-1 -mb-1 sm:pb-0 sm:mb-0">
          <!-- 刷新按钮 -->
          <Button variant="outline" size="icon" class="h-9 w-9 shrink-0" @click="fetchNodes" :disabled="loading" title="刷新">
            <RefreshCw class="h-4 w-4" :class="{ 'animate-spin': loading }" />
          </Button>

          <!-- 取消主控角色按钮 -->
          <Button variant="outline" class="flex-1 sm:flex-none shrink-0 px-2 md:px-3 h-9 shadow-sm text-destructive border-destructive/20 hover:bg-destructive/10" @click="emit('cancel')">
            取消主控
          </Button>

          <!-- 添加子节点按钮 -->
          <Button v-if="activeTab === 'nodes'" @click="masterListRef?.openAddDialog()" class="flex-1 sm:flex-none shrink-0 px-2 md:px-3 h-9 shadow-sm font-medium" title="添加子节点">
            <Plus class="h-4 w-4 md:mr-1.5" /> <span class="hidden md:inline">添加子节点</span>
          </Button>
          
          <!-- Tabs 切换 (暂时隐藏同步功能) -->
          <!--
          <Tabs :model-value="activeTab" @update:model-value="(v: string | number) => activeTab = String(v)" class="shrink-0 flex-1 sm:flex-none min-w-[120px]">
             <TabsList class="h-9 p-0.5 bg-muted/20 border border-border/40 rounded-lg w-full flex">
                <TabsTrigger value="nodes" class="flex-1 sm:flex-none px-3 h-8 text-xs gap-1.5 font-medium transition-all">
                   <Server class="w-3.5 h-3.5 opacity-70" />
                   <span>节点</span>
                </TabsTrigger>
                <TabsTrigger value="sync" class="flex-1 sm:flex-none px-3 h-8 text-xs gap-1.5 font-medium transition-all">
                   <ArrowRightLeft class="w-3.5 h-3.5 opacity-70" />
                   <span>同步</span>
                </TabsTrigger>
             </TabsList>
          </Tabs>
          -->
        </div>
      </div>
    </div>

    <!-- 内容区域 -->
    <MasterList v-if="activeTab === 'nodes'" ref="masterListRef" :nodes="nodes" :loading="loading" :search-query="searchQuery" @refresh="fetchNodes" />
    <!-- <SyncPanel v-if="activeTab === 'sync'" :nodes="nodes" /> -->
  </div>
</template>
