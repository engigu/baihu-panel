<script setup lang="ts">
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Copy, Terminal, Key, FileJson, RefreshCw, Check, Hash, Info, AlertTriangle, Code2 } from 'lucide-vue-next'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
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

const activeLang = ref('shell')
const testTitle = ref('系统通知')
const testText = ref('这是一条测试消息内容')
const testChannel = ref(props.channels[0]?.id || 'ID')

const shellExample = computed(() => `# 1. 设置环境变量 (可选)
# export BH_NOTIFY_TOKEN="${props.apiToken || 'YOUR_TOKEN'}"

# 2. 调用 (优先使用环境变量)
TOKEN="\${BH_NOTIFY_TOKEN:-${props.apiToken || 'YOUR_TOKEN'}}"
curl -s -X POST "http://${host.value}/api/v1/notify/send" \\
  -H "Content-Type: application/json" \\
  -H "notify-token: $TOKEN" \\
  -d '{"channel_id":"${testChannel.value}","title":"${testTitle.value}","text":"${testText.value}"}'`)

const pythonExample = computed(() => `import requests
import os

def send_notify(text, title="${testTitle.value}", channel="${testChannel.value}"):
    token = os.getenv("BH_NOTIFY_TOKEN", "${props.apiToken || 'YOUR_TOKEN'}")
    url = f"http://${host.value}/api/v1/notify/send"
    data = {"channel_id": channel, "title": title, "text": text}
    return requests.post(url, json=data, headers={"notify-token": token}).json()

# 调用示例
print(send_notify("${testText.value}"))`)

const javascriptExample = computed(() => `/**
 * 发送通知 (优先使用环境变量 BH_NOTIFY_TOKEN)
 * @param {string} text 消息内容
 */
const sendNotify = async (text) => {
  const token = (typeof process !== 'undefined' ? process.env.BH_NOTIFY_TOKEN : null) || "${props.apiToken || 'YOUR_TOKEN'}";
  const res = await fetch("http://${host.value}/api/v1/notify/send", {
    body: JSON.stringify({
      channel_id: "${testChannel.value}",
      title: "${testTitle.value}",
      text: text
    })
  });
  return await res.json();
};

sendNotify("${testText.value}");`)

const goExample = computed(() => `package main
import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
)

func sendNotify(title, text string) {
	token := os.Getenv("BH_NOTIFY_TOKEN")
	if token == "" {
		token = "${props.apiToken || 'YOUR_TOKEN'}"
	}
	payload, _ := json.Marshal(map[string]string{
		"channel_id": "${testChannel.value}",
		"title":      title,
		"text":       text,
	})
	req, _ := http.NewRequest("POST", "http://${host.value}/api/v1/notify/send", bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("notify-token", token)
	http.DefaultClient.Do(req)
}

func main() {
	sendNotify("${testTitle.value}", "${testText.value}")
}`)

const phpExample = computed(() => `<?php
function sendNotify($text, $title = "${testTitle.value}", $channel = "${testChannel.value}") {
    $token = getenv("BH_NOTIFY_TOKEN") ?: "${props.apiToken || 'YOUR_TOKEN'}";
    $curl = curl_init("http://${host.value}/api/v1/notify/send");
    curl_setopt($curl, CURLOPT_POST, true);
    curl_setopt($curl, CURLOPT_HTTPHEADER, [
        "Content-Type: application/json",
        "notify-token: " . $token
    ]);
    curl_setopt($curl, CURLOPT_POSTFIELDS, json_encode([
        "channel_id" => $channel,
        "title"      => $title,
        "text"       => $text
    ]));
    curl_exec($curl);
}
sendNotify("${testText.value}");
?>`)

const examples = [
  { id: 'shell', name: 'Shell', icon: Terminal, code: shellExample },
  { id: 'python', name: 'Python', icon: Code2, code: pythonExample },
  { id: 'javascript', name: 'JavaScript', icon: Code2, code: javascriptExample },
  { id: 'go', name: 'Go', icon: Code2, code: goExample },
  { id: 'php', name: 'PHP', icon: Code2, code: phpExample }
]

const highlightCode = (code: string, lang: string) => {
  if (!code) return ''

  const colors = {
    keyword: 'text-violet-500 dark:text-violet-400 font-medium',
    string: 'text-emerald-600 dark:text-emerald-400',
    comment: 'text-zinc-400 dark:text-zinc-500 italic',
    type: 'text-amber-600 dark:text-amber-500',
    function: 'text-blue-500 dark:text-blue-400',
    number: 'text-orange-500',
    operator: 'text-zinc-400 dark:text-zinc-600'
  }

  // 基础转义
  let html = code
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')

  // 1. 处理字符串 (优先处理，防止内部匹配)
  html = html.replace(/("(?:\\.|[^"])*"|'(?:\\.|[^'])*')/g, `<span class="${colors.string}">$&</span>`)

  // 2. 处理注释 (注意排除 http:// 或 https:// 中的双斜杠)
  html = html.replace(/(^|[^\:])(\/\/.+)$|(#.+)$/gm, `$1<span class="${colors.comment}">$2$3</span>`)

  // 3. 语言配置
  const langConfig: Record<string, { keywords: string[], types: string[], functions: string[] }> = {
    shell: {
      keywords: ['curl', 'export'],
      types: [],
      functions: ['send_notification']
    },
    python: {
      keywords: ['import', 'def', 'return', 'as', 'from', 'print'],
      types: ['dict', 'list', 'str', 'int', 'float'],
      functions: ['send_notify', 'post', 'json', 'getenv']
    },
    javascript: {
      keywords: ['async', 'await', 'function', 'const', 'return', 'let', 'var', 'if', 'else'],
      types: ['JSON', 'Promise', 'fetch'],
      functions: ['sendNotify', 'stringify', 'json', 'post']
    },
    go: {
      keywords: ['package', 'import', 'func', 'return', 'map', 'defer', 'main'],
      types: ['string', 'error', 'byte', 'int'],
      functions: ['Marshal', 'NewRequest', 'Set', 'Do', 'Close', 'sendNotify']
    },
    php: {
      keywords: ['function', 'return', 'true'],
      types: [],
      functions: ['curl_init', 'curl_setopt', 'curl_exec', 'json_encode', 'sendNotify']
    }
  }

  const conf = langConfig[lang === 'javascript' ? 'javascript' : (lang === 'shell' ? 'shell' : lang)]
  if (conf) {
    if (conf.keywords.length) {
      const regex = new RegExp(`\\b(${conf.keywords.join('|')})\\b(?![^<]*>)`, 'g')
      html = html.replace(regex, `<span class="${colors.keyword}">$1</span>`)
    }
    if (conf.types.length) {
      const regex = new RegExp(`\\b(${conf.types.join('|')})\\b(?![^<]*>)`, 'g')
      html = html.replace(regex, `<span class="${colors.type}">$1</span>`)
    }
    if (conf.functions.length) {
      const regex = new RegExp(`\\b(${conf.functions.join('|')})\\b(?![^<]*>)`, 'g')
      html = html.replace(regex, `<span class="${colors.function}">$1</span>`)
    }
  }

  // 4. 操作符和括号 (可选，这里处理基础)
  // html = html.replace(/[:{}[\],]/g, `<span class="${colors.operator}">$&</span>`)

  return html
}

const currentExample = computed(() => {
  const code = examples.find(e => e.id === activeLang.value)?.code.value || ''
  return highlightCode(code, activeLang.value)
})
</script>

<template>
  <div class="space-y-6">
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
      <CardContent class="grid grid-cols-1 md:grid-cols-2 gap-6 pt-2">
        <!-- 左侧：Token 管理 -->
        <div class="space-y-4">
          <div class="space-y-2">
            <label class="text-[11px] font-bold text-muted-foreground uppercase tracking-widest">当前密钥</label>
            <div class="flex flex-col sm:flex-row items-stretch sm:items-center gap-3">
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
          </div>
          <div
            class="flex items-start gap-2 p-3 rounded-lg bg-amber-500/5 border border-amber-500/10 text-[12px] text-amber-700 dark:text-amber-400">
            <Info class="w-4 h-4 mt-0.5 shrink-0" />
            <p>请妥善保管您的 Token，令牌将作为请求头中的 <code>notify-token</code> 字段发送。</p>
          </div>
        </div>

        <!-- 右侧：测试参数预览 -->
        <div class="space-y-4 p-4 rounded-xl bg-zinc-50 dark:bg-zinc-900/50 border border-zinc-200 dark:border-zinc-800">
          <div class="flex items-center gap-2 mb-2">
            <Badge variant="outline" class="text-[10px] py-0 border-emerald-500/30 text-emerald-600 bg-emerald-500/5">参数预览</Badge>
            <span class="text-xs text-muted-foreground">修改下方参数实时生成代码</span>
          </div>
          <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <div class="space-y-1.5">
              <label class="text-[11px] font-medium text-zinc-500">标题 (Title)</label>
              <Input v-model="testTitle" class="h-8 text-xs" />
            </div>
            <div class="space-y-1.5">
              <label class="text-[11px] font-medium text-zinc-500">选择渠道 (Channel ID)</label>
              <Select v-model="testChannel">
                <SelectTrigger class="w-full h-8 text-xs bg-background">
                  <SelectValue placeholder="选择渠道" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem v-if="channels.length === 0" value="ID">无可用渠道</SelectItem>
                  <SelectItem v-for="ch in channels" :key="ch.id" :value="ch.id">
                    <div class="flex items-center gap-2">
                      <span class="font-medium">{{ ch.name }}</span>
                      <span class="text-[10px] text-muted-foreground font-mono">({{ ch.id.slice(0, 6) }})</span>
                    </div>
                  </SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>
          <div class="space-y-1.5">
            <label class="text-[11px] font-medium text-zinc-500">正文内容 (Text)</label>
            <Input v-model="testText" class="h-8 text-xs" />
          </div>
        </div>
      </CardContent>
    </Card>

    <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
      <!-- API 接口规格 -->
      <Card class="border bg-card shadow-sm flex flex-col overflow-hidden h-[520px]">
        <CardHeader class="pb-3 shrink-0">
          <div class="flex items-center justify-between">
            <div class="flex items-center gap-2">
              <div class="p-1.5 rounded-md bg-emerald-500/10 text-emerald-600">
                <FileJson class="w-4 h-4" />
              </div>
              <CardTitle class="text-sm font-bold uppercase tracking-wider whitespace-nowrap">API 接口说明</CardTitle>
            </div>
          </div>
        </CardHeader>
        <CardContent class="p-0 flex-1 overflow-y-auto">
          <div
            class="bg-zinc-50 dark:bg-zinc-950/50 p-5 font-code text-xs sm:text-sm leading-relaxed text-zinc-800 dark:text-zinc-300 relative group min-h-full">
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

      <!-- 调用示例 -->
      <Card class="border bg-card shadow-sm flex flex-col overflow-hidden h-[520px]">
        <Tabs v-model="activeLang" class="w-full flex flex-col h-full overflow-hidden">
          <CardHeader class="pb-3 shrink-0 border-b px-4">
            <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
              <div class="flex items-center gap-2">
                <div class="p-1.5 rounded-md bg-sky-500/10 text-sky-600">
                  <Terminal class="w-4 h-4" />
                </div>
                <CardTitle class="text-sm font-bold uppercase tracking-wider whitespace-nowrap">脚本调用示例</CardTitle>
              </div>
              <div class="flex items-center gap-2 sm:gap-3 justify-between sm:justify-end overflow-hidden">
                <div class="overflow-x-auto hide-scrollbar shrink-0">
                  <TabsList class="h-8 p-0.5 bg-muted/50 border flex-nowrap">
                    <TabsTrigger v-for="lang in examples" :key="lang.id" :value="lang.id"
                      class="h-7 px-2.5 text-[11px] data-[state=active]:bg-background data-[state=active]:shadow-sm whitespace-nowrap">
                      {{ lang.name }}
                    </TabsTrigger>
                  </TabsList>
                </div>
                <Button variant="outline" size="sm"
                  class="h-8 px-2.5 text-[11px] border-muted-foreground/30 hover:bg-muted transition-all shrink-0"
                  @click="copyToClipboard(currentExample, 'example')">
                  <Check v-if="copiedBlock === 'example'" class="w-3.5 h-3.5 text-emerald-500 sm:mr-1.5" />
                  <Copy v-else class="w-3.5 h-3.5 sm:mr-1.5" />
                  <span class="hidden sm:inline">复制代码</span>
                </Button>
              </div>
            </div>
          </CardHeader>
          <CardContent class="p-0 flex-1 flex flex-col overflow-hidden">
            <!-- 脚本区域：独立滚动 -->
            <div class="flex-1 overflow-y-auto p-5 font-code text-[12px] sm:text-[13px] leading-relaxed text-zinc-800 dark:text-zinc-300 bg-zinc-50 dark:bg-zinc-950/50">
              <pre class="whitespace-pre-wrap break-all" v-html="currentExample" />
            </div>

            <!-- 渠道列表区域：固定在底部，如有需要可独立滚动 -->
            <div class="shrink-0 p-5 pt-4 border-t border-zinc-200 dark:border-zinc-800 bg-zinc-100/20 dark:bg-zinc-900/10 max-h-[180px] overflow-y-auto">
              <span
                class="text-zinc-500 block mb-3 uppercase text-[10px] font-bold tracking-widest flex items-center gap-1.5">
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
          </CardContent>
        </Tabs>
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
