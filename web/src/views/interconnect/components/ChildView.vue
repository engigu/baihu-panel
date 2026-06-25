<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { Network, RefreshCw, CheckCircle2, AlertTriangle, ChevronDown, ChevronUp } from 'lucide-vue-next'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { toast } from 'vue-sonner'
import { api } from '@/api'
import { getChildStatus } from '@/api/interconnect'
import { eventBus } from '@/utils/event-bus'
import { SYSTEM_EVENTS } from '@/constants'

const emit = defineEmits<{
  (e: 'cancel'): void
}>()

const parentConfig = ref({ url: '', token: '' })
const savingSetting = ref(false)
const connectionStatus = ref<{ parent_url: string; parent_token: string; connected: boolean; tunnel_url?: string; tx_bytes?: number; rx_bytes?: number } | null>(null)
const statusLoading = ref(false)
const configExpanded = ref(false)
let unsubEventBus: (() => void) | null = null

function formatBytes(bytes?: number): string {
  if (bytes === undefined || bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

async function fetchStatus() {
  statusLoading.value = true
  try {
    const res = await getChildStatus()
    connectionStatus.value = res
  } catch (error) {
    // Ignore error
  } finally {
    statusLoading.value = false
  }
}

onMounted(async () => {
  try {
    parentConfig.value.url = await api.settings.get('interconnect', 'interconnect_parent_url') || ''
    parentConfig.value.token = await api.settings.get('interconnect', 'interconnect_parent_token') || ''
    await fetchStatus()
    configExpanded.value = !connectionStatus.value?.parent_url

    unsubEventBus = eventBus.subscribe((msg) => {
      if (msg.type === SYSTEM_EVENTS.INTERCONNECT_CHILD_STATUS) {
        if (connectionStatus.value) {
          connectionStatus.value.connected = msg.payload.connected
        } else {
          fetchStatus()
        }
      }
    })
  } catch (error) {
    // Ignore error
  }
})

onUnmounted(() => {
  if (unsubEventBus) unsubEventBus()
})

async function handleSaveParentConfig() {
  if (!parentConfig.value.url || !parentConfig.value.token) {
    toast.error('请填写完整的主面板地址 and 互联密钥')
    return
  }
  savingSetting.value = true
  try {
    await api.settings.setSection('interconnect', {
      interconnect_parent_url: parentConfig.value.url,
      interconnect_parent_token: parentConfig.value.token
    })
    toast.success('配置已保存，正在主动建立反向安全隧道')
    await fetchStatus()
    configExpanded.value = false
  } catch (error: any) {
    toast.error(error.message || '保存失败')
  } finally {
    savingSetting.value = false
  }
}
</script>

<template>
  <div class="max-w-2xl mx-auto pt-2 space-y-4">
    <!-- 顶部状态标题 -->
    <div class="flex flex-col items-center justify-center text-center space-y-2 mb-2">
      <div class="inline-flex h-10 w-10 items-center justify-center rounded-full bg-green-500/10 text-green-500">
        <Network class="h-5 w-5" />
      </div>
      <div>
        <h2 class="text-lg font-bold tracking-tight">本机作为子节点运行</h2>
        <p class="text-muted-foreground text-xs mt-1 max-w-lg">本机正在受控模式下运行，将定期向主节点汇报状态，并允许主节点穿越到本面板。</p>
      </div>
    </div>

    <!-- 连接状态展示面板 (仅在配置存在时显示) -->
    <div v-if="connectionStatus?.parent_url" class="rounded-xl border p-5 md:p-6 bg-card shadow-sm space-y-3.5 animate-in fade-in slide-in-from-bottom-2 duration-300">
      <div class="flex items-center justify-between border-b pb-2">
        <h3 class="text-xs font-bold text-muted-foreground uppercase tracking-wider">连接状态</h3>
        <Button variant="ghost" size="icon" class="h-7 w-7 rounded-full" @click="fetchStatus" :disabled="statusLoading" title="刷新状态">
          <RefreshCw class="h-3.5 w-3.5" :class="{ 'animate-spin': statusLoading }" />
        </Button>
      </div>

      <!-- 在线状态 -->
      <div v-if="connectionStatus.connected" class="flex items-start gap-3 p-3.5 rounded-lg bg-green-500/5 border border-green-500/10 text-green-600 dark:text-green-400">
        <CheckCircle2 class="h-4.5 w-4.5 shrink-0 mt-0.5" />
        <div class="space-y-1">
          <div class="text-xs font-bold">已成功连接至主控端</div>
          <div class="text-[10px] opacity-90 leading-relaxed">
            与主控端的反向安全物理隧道已打通，网络路径畅通。主控端现可无缝“穿越”并集中监控本机。
          </div>
        </div>
      </div>

      <!-- 离线状态 -->
      <div v-else class="flex items-start gap-3 p-3.5 rounded-lg bg-amber-500/5 border border-amber-500/10 text-amber-600 dark:text-amber-400">
        <AlertTriangle class="h-4.5 w-4.5 shrink-0 mt-0.5" />
        <div class="space-y-1">
          <div class="text-xs font-bold">未建立与主控的物理连接</div>
          <div class="text-[10px] opacity-90 leading-relaxed">
            物理隧道连接中断，系统正在后台进行自动重连。可能原因为：主控地址配置错误、专属密钥与主控不匹配，或者主控节点未处于在线状态。
          </div>
        </div>
      </div>

      <!-- 信息详情 -->
      <div class="grid grid-cols-1 sm:grid-cols-2 gap-x-4 gap-y-2 text-xs pt-1">
        <div class="flex justify-between py-1 border-b border-border/40">
          <span class="text-muted-foreground">主控地址</span>
          <span class="font-medium truncate max-w-[180px]" :title="connectionStatus.parent_url">{{ connectionStatus.parent_url }}</span>
        </div>
        <div class="flex justify-between py-1 border-b border-border/40" v-if="connectionStatus.tunnel_url">
          <span class="text-muted-foreground">物理隧道地址</span>
          <span class="font-medium truncate max-w-[180px]" :title="connectionStatus.tunnel_url">{{ connectionStatus.tunnel_url }}</span>
        </div>
        <div class="flex justify-between py-1 border-b border-border/40">
          <span class="text-muted-foreground">物理隧道状态</span>
          <span class="font-bold flex items-center gap-1.5" :class="connectionStatus.connected ? 'text-green-500' : 'text-amber-500'">
            <span class="h-1.5 w-1.5 rounded-full" :class="connectionStatus.connected ? 'bg-green-500' : 'bg-amber-500 animate-pulse'"></span>
            {{ connectionStatus.connected ? '正常在线' : '离线 / 尝试重连' }}
          </span>
        </div>
        <div class="flex justify-between py-1 border-b border-border/40" v-if="connectionStatus.tx_bytes !== undefined || connectionStatus.rx_bytes !== undefined">
          <span class="text-muted-foreground">物理隧道流量</span>
          <span class="font-medium">
            <span class="text-green-500" title="发送 (TX)">{{ formatBytes(connectionStatus.tx_bytes) }}</span>
            <span class="mx-1 text-muted-foreground">/</span>
            <span class="text-blue-500" title="接收 (RX)">{{ formatBytes(connectionStatus.rx_bytes) }}</span>
          </span>
        </div>
      </div>
    </div>

    <!-- 连接配置面板 -->
    <div class="rounded-xl border bg-card p-5 md:p-6 shadow-sm space-y-4">
      <!-- 头部：如果已配置且未展开，显示简化的“修改配置”触发栏 -->
      <div v-if="connectionStatus?.parent_url && !configExpanded" class="flex items-center justify-between">
        <div class="flex flex-col">
          <span class="text-sm font-semibold">主控连接配置</span>
          <span class="text-[10px] text-muted-foreground mt-0.5">配置已保存，若需更新连接凭证请点击展开。</span>
        </div>
        <div class="flex items-center gap-2">
          <Button variant="outline" size="sm" class="text-xs h-8 px-3" @click="configExpanded = true">
            展开配置
            <ChevronDown class="h-3.5 w-3.5 ml-1" />
          </Button>
          <Button variant="outline" size="sm" class="text-destructive hover:bg-destructive/10 text-xs px-2.5 h-8" @click="emit('cancel')">取消子节点</Button>
        </div>
      </div>

      <!-- 展开状态的完整配置面板 -->
      <template v-else>
        <div class="flex items-center justify-between border-b pb-2.5">
          <h3 class="text-sm font-semibold">主控连接配置</h3>
          <div class="flex items-center gap-2">
            <Button v-if="connectionStatus?.parent_url" variant="ghost" size="sm" class="text-xs h-7 px-2.5" @click="configExpanded = false">
              收起
              <ChevronUp class="h-3.5 w-3.5 ml-1" />
            </Button>
            <Button variant="outline" size="sm" class="text-destructive hover:bg-destructive/10 text-xs px-2.5 h-7" @click="emit('cancel')">取消子节点角色</Button>
          </div>
        </div>

        <div class="space-y-3">
          <div class="grid gap-1.5">
            <Label for="parentUrl" class="text-xs">主面板地址 (URL) <span class="text-destructive">*</span></Label>
            <Input id="parentUrl" v-model="parentConfig.url" placeholder="例如：http://main-panel.com:8052" autocomplete="off" class="h-8 text-xs" />
            <p class="text-[10px] text-muted-foreground">填写主面板的访问地址（包含协议和端口）。</p>
          </div>
          <div class="grid gap-1.5">
            <Label for="parentToken" class="text-xs">专属接入密钥 (Token) <span class="text-destructive">*</span></Label>
            <Input id="parentToken" v-model="parentConfig.token" type="password" placeholder="粘贴从主面板生成的专属接入密钥" autocomplete="new-password" class="h-8 text-xs" />
            <p class="text-[10px] text-muted-foreground">主面板添加节点时自动生成的随机高强度密钥。</p>
          </div>
        </div>
        
        <div class="pt-3 border-t">
          <Button @click="handleSaveParentConfig" :disabled="savingSetting" class="w-full h-9 text-xs">
            <RefreshCw v-if="savingSetting" class="h-3.5 w-3.5 mr-1.5 animate-spin" />
            保存并连接主节点
          </Button>
        </div>
      </template>
    </div>
  </div>
</template>
