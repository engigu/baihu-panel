<script setup lang="ts">
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Copy, Terminal, Key, FileJson, RefreshCw, Check, Hash, Info, AlertTriangle } from 'lucide-vue-next'
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
import type { NotifyChannel, ChannelType } from '@/api'
import { ref, computed } from 'vue'
import { toast } from 'vue-sonner'

const props = defineProps<{
  channels: NotifyChannel[]
  channelTypes: ChannelType[]
  apiToken: string
}>()

const emit = defineEmits<{
  generateToken: []
  copyToken: []
  copyExample: []
}>()

const copiedBlock = ref<string | null>(null)
const host = ref(window.location.host)
const showConfirmDialog = ref(false)

function onGenerateClick() {
  if (props.apiToken) {
    showConfirmDialog.value = true
  } else {
    emit('generateToken')
  }
}

function handleConfirmGenerate() {
  showConfirmDialog.value = false
  emit('generateToken')
}

function copyToClipboard(text: string, blockId: string) {
  navigator.clipboard.writeText(text).then(() => {
    copiedBlock.value = blockId
    toast.success('已复制到剪贴板')
    setTimeout(() => {
      copiedBlock.value = null
    }, 2000)
  })
}

const apiExample = `POST /api/v1/notify/send
Content-Type: application/json
notify-token: <你的API Token>

{
  "channel_id": "渠道ID",
  "title": "标题",
  "text": "内容"
}`

const shellExample = computed(() => `curl -s -X POST "http://${host.value}/api/v1/notify/send" \\
  -H "Content-Type: application/json" \\
  -H "notify-token: ${props.apiToken || 'YOUR_TOKEN'}" \\
  -d '{"channel_id":"YOUR_CHANNEL_ID","title":"标题","text":"通知内容"}'`)
</script>

<template>
  <div class="space-y-6">
    <!-- API Token 管理卡片 -->
    <Card class="border bg-card shadow-sm overflow-hidden">
      <CardHeader class="pb-4">
        <div class="flex items-center gap-2 mb-1">
          <div class="p-1.5 rounded-md bg-primary/10 text-primary">
            <Key class="w-4 h-4" />
          </div>
          <CardTitle class="text-base font-semibold">身份验证 (API Token)</CardTitle>
        </div>
        <CardDescription>用于外部脚本或工具调用 API 时的安全凭证</CardDescription>
      </CardHeader>
      <CardContent class="space-y-4">
        <div class="flex items-center gap-3">
          <div class="relative flex-1 group">
            <Input :model-value="apiToken" readonly placeholder="尚未生成 Token"
              class="h-10 pr-10 bg-muted/30 border-muted-foreground/20 focus-visible:ring-primary/30 font-code text-sm tracking-tight" />
            <div v-if="apiToken" @click="copyToClipboard(apiToken, 'token')"
              class="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-primary cursor-pointer transition-colors p-1 rounded-md hover:bg-muted"
              title="复制 Token">
              <Check v-if="copiedBlock === 'token'" class="w-4 h-4 text-emerald-500 animate-in zoom-in" />
              <Copy v-else class="w-4 h-4" />
            </div>
          </div>
          <Button variant="default" @click="onGenerateClick" class="h-10 px-4 shrink-0 transition-all active:scale-95">
            <RefreshCw class="w-3.5 h-3.5 mr-2" />
            {{ apiToken ? '重新生成' : '生成 Token' }}
          </Button>
        </div>
        <div
          class="flex items-start gap-2 p-3 rounded-lg bg-amber-500/5 border border-amber-500/10 text-[13px] text-amber-700 dark:text-amber-400">
          <Info class="w-4 h-4 mt-0.5 shrink-0" />
          <p>请妥善保管您的 Token，一旦丢失需通过上方按钮重新生成。令牌将作为请求头中的 <code>notify-token</code> 字段发送。</p>
        </div>
      </CardContent>
    </Card>

    <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
      <!-- API 接口规格 -->
      <Card class="border bg-card shadow-sm flex flex-col overflow-hidden">
        <CardHeader class="pb-3 shrink-0">
          <div class="flex items-center justify-between">
            <div class="flex items-center gap-2">
              <div class="p-1.5 rounded-md bg-emerald-500/10 text-emerald-600">
                <FileJson class="w-4 h-4" />
              </div>
              <CardTitle class="text-sm font-bold uppercase tracking-wider">API 接口说明</CardTitle>
            </div>
          </div>
        </CardHeader>
        <CardContent class="p-0 flex-1">
          <div
            class="bg-zinc-50 dark:bg-zinc-950/50 p-5 font-code text-xs sm:text-sm leading-relaxed text-zinc-800 dark:text-zinc-300 relative group h-full">
            <div class="flex items-center justify-between mb-6 border-b border-zinc-200 dark:border-zinc-800/50 pb-3">
              <div class="flex items-center gap-2">
                <Badge class="bg-emerald-600 text-white border-none py-0 px-2 text-[10px]">POST</Badge>
                <code
                  class="px-2 py-0.5 bg-zinc-200/50 dark:bg-zinc-800/60 border border-zinc-300/50 dark:border-zinc-700/50 rounded-md text-[13px] text-zinc-700 dark:text-zinc-300 font-mono shadow-sm tracking-tight">/api/v1/notify/send</code>
              </div>
              <Button variant="ghost" size="icon"
                class="h-7 w-7 text-zinc-500 hover:text-zinc-900 dark:hover:text-white hover:bg-zinc-200 dark:hover:bg-zinc-800 transition-all rounded-md"
                @click="copyToClipboard(apiExample, 'api')">
                <Check v-if="copiedBlock === 'api'" class="w-3.5 h-3.5 text-emerald-500" />
                <Copy v-else class="w-3.5 h-3.5" />
              </Button>
            </div>

            <div class="space-y-6">
              <div>
                <span class="block mb-2 text-zinc-500 uppercase text-[10px] font-bold tracking-widest">Headers</span>
                <div class="space-y-2">
                  <div class="flex items-center gap-2">
                    <span class="text-zinc-500">Content-Type:</span>
                    <span>application/json</span>
                  </div>
                  <div class="flex items-center gap-2">
                    <span class="text-zinc-500">notify-token:</span>
                    <span class="text-primary">&lt;TOKEN&gt;</span>
                  </div>
                </div>
              </div>

              <div>
                <span class="block mb-2 text-zinc-500 uppercase text-[10px] font-bold tracking-widest">Body
                  (JSON)</span>
                <div class="pl-2 font-mono">
                  <p class="text-zinc-600 dark:text-zinc-400 mb-1">{</p>
                  <div class="pl-4 space-y-2">
                    <div class="flex flex-wrap items-center gap-x-2">
                      <span class="text-orange-600 dark:text-orange-400">"channel_id":</span>
                      <span class="text-emerald-600 dark:text-emerald-400">"ID"</span><span
                        class="text-zinc-600 dark:text-zinc-400">,</span>
                      <span class="text-zinc-500 font-sans italic ml-auto whitespace-nowrap">//
                        渠道唯一标识</span>
                    </div>
                    <div class="flex flex-wrap items-center gap-x-2">
                      <span class="text-orange-600 dark:text-orange-400">"title":</span>
                      <span class="text-emerald-600 dark:text-emerald-400">"标题"</span><span
                        class="text-zinc-600 dark:text-zinc-400">,</span>
                      <span class="text-zinc-500 font-sans italic ml-auto whitespace-nowrap">// 可选</span>
                    </div>
                    <div class="flex flex-wrap items-center gap-x-2">
                      <span class="text-orange-600 dark:text-orange-400">"text":</span>
                      <span class="text-emerald-600 dark:text-emerald-400">"内容"</span>
                      <span class="text-zinc-500 font-sans italic ml-auto whitespace-nowrap">// 必填</span>
                    </div>
                  </div>
                  <p class="text-zinc-600 dark:text-zinc-400 mt-1">}</p>
                </div>
              </div>
            </div>
          </div>
        </CardContent>
      </Card>

      <!-- Shell 示例 -->
      <Card class="border bg-card shadow-sm flex flex-col overflow-hidden">
        <CardHeader class="pb-3 shrink-0">
          <div class="flex items-center justify-between">
            <div class="flex items-center gap-2">
              <div class="p-1.5 rounded-md bg-sky-500/10 text-sky-600">
                <Terminal class="w-4 h-4" />
              </div>
              <CardTitle class="text-sm font-bold uppercase tracking-wider">Shell 脚本示例</CardTitle>
            </div>
            <Button variant="outline" size="sm"
              class="h-7 px-2 text-[10px] border-muted-foreground/30 hover:bg-muted transition-all"
              @click="copyToClipboard(shellExample, 'shell')">
              <Check v-if="copiedBlock === 'shell'" class="w-3 h-3 text-emerald-500 mr-1.5" />
              <Copy v-else class="w-3 h-3 mr-1.5" />
              一键复制
            </Button>
          </div>
        </CardHeader>
        <CardContent class="p-0 flex-1">
          <div
            class="bg-zinc-50 dark:bg-zinc-950/50 p-5 font-code text-[12px] sm:text-[13px] leading-relaxed text-zinc-800 dark:text-zinc-300 h-full">
            <div class="space-y-1">
              <p><span class="text-zinc-500"># 使用 CURL 调用推送接口</span></p>
              <p>curl -s -X POST <span class="text-emerald-600 dark:text-emerald-400">"http://{{ host
                  }}/api/v1/notify/send"</span> \</p>
              <p class="pl-4"> -H <span class="text-orange-600 dark:text-orange-400">"Content-Type:
                  application/json"</span> \</p>
              <p class="pl-4"> -H <span class="text-orange-600 dark:text-orange-400">"notify-token: {{ apiToken ||
                'YOUR_TOKEN' }}"</span> \
              </p>
              <p class="pl-4"> -d <span
                  class="text-orange-600 dark:text-orange-400">'{"channel_id":"ID","title":"任务完成","text":"脚本执行完毕"}'</span>
              </p>
            </div>

            <div class="mt-8 pt-4 border-t border-zinc-200 dark:border-zinc-800">
              <span
                class="text-zinc-500 block mb-2 uppercase text-[10px] font-bold tracking-widest flex items-center gap-1.5">
                <Hash class="w-3 h-3" /> 渠道 ID 快速查找
              </span>
              <div v-if="channels.length === 0" class="text-xs text-zinc-500 italic">暂无活跃渠道</div>
              <div v-else class="grid grid-cols-1 sm:grid-cols-2 gap-2">
                <div v-for="ch in channels" :key="ch.id"
                  class="flex items-center gap-2 text-xs bg-zinc-100/80 dark:bg-zinc-900 px-2 py-1.5 rounded border border-zinc-200 dark:border-zinc-800 hover:border-zinc-300 dark:hover:border-zinc-700 transition-colors group">
                  <code class="text-primary font-bold tracking-tighter font-code"
                    :title="ch.id">{{ ch.id.slice(0, 8) }}...</code>
                  <span class="text-zinc-600 dark:text-zinc-500 truncate max-w-[100px]">{{ ch.name }}</span>
                  <Button variant="ghost" size="icon"
                    class="h-5 w-5 ml-auto text-zinc-400 hover:text-zinc-800 dark:hover:text-zinc-200 hover:bg-zinc-200 dark:hover:bg-zinc-800 opacity-0 group-hover:opacity-100 transition-all rounded"
                    @click="copyToClipboard(ch.id, 'channel-' + ch.id)">
                    <Check v-if="copiedBlock === 'channel-' + ch.id" class="w-2.5 h-2.5 text-emerald-500" />
                    <Copy v-else class="w-2.5 h-2.5" />
                  </Button>
                </div>
              </div>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>

    <!-- 重新生成确认弹窗 -->
    <AlertDialog :open="showConfirmDialog" @update:open="showConfirmDialog = $event">
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle class="flex items-center gap-2">
            <AlertTriangle class="w-5 h-5 text-amber-500" />
            确认重新生成 Token？
          </AlertDialogTitle>
          <AlertDialogDescription>
            此操作将立刻覆盖当前 Token，旧的 Token 将会永久失效。确认要继续吗？
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>取消</AlertDialogCancel>
          <AlertDialogAction @click="handleConfirmGenerate">确认重新生成</AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  </div>
</template>

<style scoped>
.font-code {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace;
  position: relative;
}

.font-code ::selection {
  background-color: rgba(59, 130, 246, 0.4);
  /* 鲜亮的蓝色选中背景 */
  color: #fff;
  /* 选中时文字为白色 */
}

/* 兼容 Safari */
.font-code ::-moz-selection {
  background-color: rgba(59, 130, 246, 0.4);
  color: #fff;
}

/* 优化滚动条样式 if needed */
.font-code::-webkit-scrollbar {
  width: 6px;
  height: 6px;
}

.font-code::-webkit-scrollbar-thumb {
  background: rgba(255, 255, 255, 0.1);
  border-radius: 3px;
}

.font-code::-webkit-scrollbar-thumb:hover {
  background: rgba(255, 255, 255, 0.2);
}
</style>
