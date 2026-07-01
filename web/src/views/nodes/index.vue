<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { api } from '@/api'
import { toast } from 'vue-sonner'
import { Plus, RefreshCw, Search, Server, Network, Ticket, Download } from 'lucide-vue-next'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import * as nodeApi from '@/api/node'

import RoleSelector from './components/RoleSelector.vue'
import MasterView from './components/MasterView.vue'
import ChildView from './components/ChildView.vue'
import MasterList from './components/MasterList.vue'
import TokenListTab from './components/TokenListTab.vue'
import DownloadAgentDialog from './components/DownloadAgentDialog.vue'
import EditTokenDialog from './components/EditTokenDialog.vue'

import { useEventBus } from '@vueuse/core'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog'

const interconnectRole = ref<'master' | 'child' | 'none'>('none')
const loadingRole = ref(true)
const roleBus = useEventBus<string>('interconnect-role-changed')

// Tabs & State for Child mode Runner management
const activeTab = ref('connection')
const searchQuery = ref('')
const loading = ref(false)
const nodes = ref<nodeApi.NodeDTO[]>([])
const tokens = ref<nodeApi.NodeToken[]>([])
const agentVersion = ref('')
const platforms = ref<{ os: string; arch: string; filename: string }[]>([])

const masterListRef = ref<InstanceType<typeof MasterList> | null>(null)
const downloadAgentDialogRef = ref<InstanceType<typeof DownloadAgentDialog> | null>(null)
const editTokenDialogRef = ref<InstanceType<typeof EditTokenDialog> | null>(null)

async function fetchRole() {
  loadingRole.value = true
  try {
    const role = await api.settings.get('interconnect', 'interconnect_role') as string
    interconnectRole.value = (role === 'master' || role === 'child') ? role : 'none'
    if (interconnectRole.value === 'child') {
      await fetchNodes()
    }
  } catch (error) {
    interconnectRole.value = 'none'
  } finally {
    loadingRole.value = false
  }
}

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
    toast.error('获取 Agent 数据失败')
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

async function setRole(role: 'master' | 'child' | 'none') {
  try {
    await api.settings.setSection('interconnect', { interconnect_role: role })
    interconnectRole.value = role
    roleBus.emit(role)
    
    if (role === 'none') {
      await api.settings.setSection('interconnect', {
        interconnect_parent_url: '',
        interconnect_parent_token: ''
      })
    } else if (role === 'child') {
      activeTab.value = 'connection'
      await fetchNodes()
    }
  } catch (error: any) {
    toast.error('角色设置失败')
  }
}

const showCancelConfirm = ref(false)

function handleCancelRole() {
  showCancelConfirm.value = true
}

async function confirmCancelRole() {
  showCancelConfirm.value = false
  await setRole('none')
}

onMounted(async () => {
  await fetchRole()
})
</script>

<template>
  <div class="space-y-6">
    <div v-if="loadingRole" class="h-[60vh] w-full flex flex-col items-center justify-center space-y-4 animate-in fade-in duration-300">
      <div class="relative flex items-center justify-center">
        <!-- 外圈渐变呼吸环 -->
        <div class="absolute h-10 w-10 rounded-full border border-primary/25 animate-ping"></div>
        <!-- 旋转环 -->
        <div class="h-10 w-10 rounded-full border-2 border-primary/10 border-t-primary animate-spin"></div>
      </div>
      <span class="text-xs font-medium text-muted-foreground/80 tracking-wider animate-pulse">正在载入配置...</span>
    </div>

    <!-- 状态一：未选择角色 -->
    <RoleSelector v-else-if="interconnectRole === 'none'" @select="setRole" />

    <!-- 状态二：主节点视图 -->
    <MasterView v-else-if="interconnectRole === 'master'" @cancel="handleCancelRole" />

    <!-- 状态三：子节点视图 -->
    <ChildView v-else-if="interconnectRole === 'child'" @cancel="handleCancelRole" />

    <AlertDialog :open="showCancelConfirm" @update:open="showCancelConfirm = $event">
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>确认取消角色？</AlertDialogTitle>
          <AlertDialogDescription>
            <template v-if="interconnectRole === 'master'">
              切换为子节点或取消角色将导致所有连接的子节点失去控制，是否继续？
            </template>
            <template v-else-if="interconnectRole === 'child'">
              取消配置将断开与主节点的连接，是否继续？
            </template>
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>取消</AlertDialogCancel>
          <AlertDialogAction @click="confirmCancelRole" class="bg-destructive text-destructive-foreground hover:bg-destructive/90">确认</AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  </div>
</template>
