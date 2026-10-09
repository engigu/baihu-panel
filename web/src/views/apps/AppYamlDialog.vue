<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted } from 'vue'
import {
  Dialog,
  DialogContent,
  DialogTitle,
  DialogDescription
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import {
  FileCode,
  Copy,
  Check,
  Download,
  ExternalLink,
  Maximize2,
  Minimize2,
  Loader2,
  Sparkles,
  Package,
  X,
  RefreshCw,
  AlertCircle,
  WrapText
} from 'lucide-vue-next'
import { VueMonacoEditor } from '@guolao/vue-monaco-editor'
import { type MarketplaceApp } from '@/api'
import { toast } from 'vue-sonner'

const props = withDefaults(
  defineProps<{
    open: boolean
    app: MarketplaceApp | null
  }>(),
  {
    open: false,
    app: null
  }
)

const emit = defineEmits<{
  'update:open': [value: boolean]
  'deploy': [app: MarketplaceApp]
}>()

const loading = ref(false)
const loadError = ref('')
const yamlContent = ref('')
const copied = ref(false)
const isFullscreen = ref(false)
const isWordWrap = ref(false) // 默认关闭换行，保持代码行完整与整齐缩进
const imageLoadError = ref(false)
const editorRef = ref<any>(null)

// 动态处理应用图标链接
const displayIconUrl = computed(() => {
  if (!props.app?.icon || imageLoadError.value) return ''
  return props.app.icon
})

function handleImageError() {
  imageLoadError.value = true
}

// 快捷键支持 (Esc 退出全屏)
function handleKeyDown(e: KeyboardEvent) {
  if (e.key === 'Escape') {
    if (isFullscreen.value) {
      isFullscreen.value = false
      e.stopPropagation()
    }
  }
}

onMounted(() => {
  window.addEventListener('keydown', handleKeyDown)
})

onUnmounted(() => {
  window.removeEventListener('keydown', handleKeyDown)
})

// Monaco Editor 挂载完成
function handleEditorMount(editor: any) {
  editorRef.value = editor
  const model = editor.getModel()
  if (model) {
    model.setEOL(0) // LF
  }
}

// 统计信息
const lineCount = computed(() => {
  if (!yamlContent.value) return 0
  return yamlContent.value.split('\n').length
})

const fileSizeFormatted = computed(() => {
  if (!yamlContent.value) return '0 B'
  const bytes = new Blob([yamlContent.value]).size
  if (bytes < 1024) return `${bytes} B`
  return `${(bytes / 1024).toFixed(1)} KB`
})

// 获取 YAML 内容
async function loadYaml() {
  if (!props.app) {
    yamlContent.value = ''
    loadError.value = ''
    return
  }

  loadError.value = ''

  // 1. 若应用对象中已存在 manifest_raw，直接使用
  if (props.app.manifest_raw && props.app.manifest_raw.trim()) {
    yamlContent.value = props.app.manifest_raw
    loading.value = false
    return
  }

  // 2. 若无 manifest_raw，从远程候选源异步拉取
  loading.value = true
  const candidates: string[] = []

  if (props.app.manifest_url) {
    candidates.push(props.app.manifest_url)
  }

  const appId = props.app.id
  candidates.push(
    `https://fastly.jsdelivr.net/gh/engigu/baihu-appstore@main/apps/${appId}/app.yaml`,
    `https://raw.githubusercontent.com/engigu/baihu-appstore/main/apps/${appId}/app.yaml`,
    `https://ghproxy.net/https://raw.githubusercontent.com/engigu/baihu-appstore/main/apps/${appId}/app.yaml`
  )

  let fetchedText = ''
  let lastErr = ''

  for (const url of candidates) {
    try {
      const resp = await fetch(url)
      if (resp.ok) {
        fetchedText = await resp.text()
        if (fetchedText && fetchedText.trim()) {
          break
        }
      }
    } catch (e: any) {
      lastErr = e.message || '网络请求异常'
    }
  }

  if (fetchedText && fetchedText.trim()) {
    yamlContent.value = fetchedText
  } else {
    loadError.value = lastErr || '未获取到 YAML 清单内容，请检查网络连接或源仓库'
  }
  loading.value = false
}

watch(
  () => [props.open, props.app],
  ([isOpen]) => {
    if (isOpen) {
      copied.value = false
      isFullscreen.value = false
      imageLoadError.value = false
      loadYaml()
    } else {
      yamlContent.value = ''
      loadError.value = ''
    }
  },
  { immediate: true }
)

// 一键复制
async function handleCopy() {
  if (!yamlContent.value) return

  try {
    if (navigator.clipboard && window.isSecureContext) {
      await navigator.clipboard.writeText(yamlContent.value)
    } else {
      const textArea = document.createElement('textarea')
      textArea.value = yamlContent.value
      textArea.style.position = 'fixed'
      textArea.style.left = '-999999px'
      textArea.style.top = '-999999px'
      document.body.appendChild(textArea)
      textArea.focus()
      textArea.select()
      document.execCommand('copy')
      textArea.remove()
    }
    copied.value = true
    toast.success('已成功复制 YAML 清单至剪贴板')
    setTimeout(() => {
      copied.value = false
    }, 2500)
  } catch (err: any) {
    toast.error('复制失败: ' + (err.message || '剪贴板访问受限'))
  }
}

// 一键下载文件
function handleDownload() {
  if (!yamlContent.value || !props.app) return

  try {
    const blob = new Blob([yamlContent.value], { type: 'text/yaml;charset=utf-8' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `${props.app.id}.app.yaml`
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
    URL.revokeObjectURL(url)
    toast.success(`已保存文件: ${props.app.id}.app.yaml`)
  } catch (err: any) {
    toast.error('下载失败: ' + (err.message || '未知错误'))
  }
}

// 前往部署
function handleDeploy() {
  if (!props.app) return
  emit('update:open', false)
  emit('deploy', props.app)
}

function handleClose() {
  emit('update:open', false)
}
</script>

<template>
  <Dialog :open="open" @update:open="emit('update:open', $event)">
    <DialogContent
      :show-close-button="false"
      :class="[
        'p-0 gap-0 overflow-hidden bg-background border border-border/80 shadow-2xl flex flex-col focus:outline-none transition-all duration-300',
        isFullscreen
          ? '!fixed !inset-0 sm:!inset-2 !w-screen sm:!w-[calc(100vw-1rem)] !max-w-none !h-[100dvh] sm:!h-[calc(100vh-1rem)] !max-h-none !translate-x-0 !translate-y-0 !top-0 sm:!top-2 !left-0 sm:!left-2 rounded-none sm:rounded-xl z-50'
          : '!w-screen sm:!w-[92vw] sm:!max-w-[880px] !h-[100dvh] sm:!h-[86vh] max-h-none sm:max-h-[820px] rounded-none sm:rounded-2xl'
      ]"
      @openAutoFocus.prevent
      @pointerDownOutside.prevent
      @interactOutside.prevent
    >
      <!-- 头部栏：在小屏下分为两行，在大屏下一行对齐 -->
      <div class="px-3 sm:px-5 py-2.5 sm:py-3 border-b border-border/60 bg-muted/20 flex flex-col sm:flex-row sm:items-center justify-between shrink-0 gap-2 sm:gap-3">
        <!-- 头部第一行（移动端：应用标题与右上角关闭；桌面端：完整信息） -->
        <div class="flex items-center justify-between sm:justify-start gap-2.5 sm:gap-3 min-w-0 flex-1">
          <div v-if="app" class="flex items-center gap-2.5 sm:gap-3 min-w-0 flex-1">
            <div class="w-8 h-8 sm:w-9 sm:h-9 rounded-xl bg-primary/10 border border-primary/20 flex items-center justify-center shrink-0 overflow-hidden shadow-xs">
              <img
                v-if="displayIconUrl"
                :src="displayIconUrl"
                class="w-full h-full object-cover"
                :alt="app.name"
                @error="handleImageError"
              />
              <Package v-else class="w-4 h-4 sm:w-5 sm:h-5 text-primary" />
            </div>

            <div class="space-y-0.5 min-w-0 flex-1">
              <div class="flex items-center gap-1.5 sm:gap-2 flex-wrap">
                <DialogTitle class="text-xs sm:text-base font-bold text-foreground truncate leading-snug max-w-[200px] xs:max-w-[260px] sm:max-w-none">
                  {{ app.name }}
                </DialogTitle>
                <Badge variant="secondary" class="font-mono text-[9px] sm:text-[10px] px-1 sm:px-1.5 py-0 shrink-0">
                  v{{ app.version }}
                </Badge>
                <Badge variant="outline" class="hidden xs:inline-flex text-[9px] font-mono px-1.5 py-0 text-primary border-primary/30 bg-primary/5 shrink-0">
                  YAML
                </Badge>
              </div>
              <DialogDescription class="text-[10px] sm:text-xs text-muted-foreground font-mono truncate">
                <span>{{ app.id }}</span>
                <span v-if="app.category" class="opacity-75"> • {{ app.category }}</span>
                <span v-if="app.author" class="hidden sm:inline opacity-75"> • {{ app.author }}</span>
              </DialogDescription>
            </div>
          </div>

          <!-- 仅在移动端小屏显示的顶栏快捷关闭按钮 -->
          <Button
            size="icon"
            variant="ghost"
            class="h-7 w-7 text-muted-foreground hover:text-foreground rounded-lg shrink-0 sm:hidden -mr-1"
            @click="handleClose"
            title="关闭窗口 (Esc)"
          >
            <X class="w-4 h-4" />
          </Button>
        </div>

        <!-- 头部操作按钮工具栏（移动端第二行自适应流式排布，桌面端靠右） -->
        <div class="flex items-center justify-between sm:justify-end gap-1.5 shrink-0 pt-1 sm:pt-0 border-t border-border/30 sm:border-t-0">
          <div class="flex items-center gap-1.5 flex-1 sm:flex-initial">
            <!-- 复制按钮 -->
            <Button
              size="sm"
              variant="outline"
              class="h-7 sm:h-8 px-2 sm:px-2.5 text-xs gap-1 sm:gap-1.5 flex-1 sm:flex-initial justify-center transition-all"
              :class="{ 'border-emerald-500 text-emerald-500 bg-emerald-500/10': copied }"
              :disabled="loading || !yamlContent"
              @click="handleCopy"
              title="一键复制全部 YAML 清单"
            >
              <Check v-if="copied" class="w-3.5 h-3.5 text-emerald-500 animate-in zoom-in-50 duration-200" />
              <Copy v-else class="w-3.5 h-3.5 opacity-80" />
              <span class="font-medium text-[11px] sm:text-xs">{{ copied ? '已复制' : '复制' }}</span>
            </Button>

            <!-- 下载文件按钮 -->
            <Button
              size="sm"
              variant="outline"
              class="h-7 sm:h-8 px-2 sm:px-2.5 text-xs gap-1 sm:gap-1.5 flex-1 sm:flex-initial justify-center"
              :disabled="loading || !yamlContent"
              @click="handleDownload"
              title="下载为 .app.yaml 文件"
            >
              <Download class="w-3.5 h-3.5 opacity-80" />
              <span class="text-[11px] sm:text-xs">下载</span>
            </Button>

            <!-- 自动换行切换按钮 -->
            <Button
              size="sm"
              variant="outline"
              class="h-7 sm:h-8 px-2 sm:px-2.5 text-xs gap-1 sm:gap-1.5 flex-1 sm:flex-initial justify-center transition-colors"
              :class="{ 'bg-primary/10 text-primary border-primary/40': isWordWrap }"
              :disabled="loading || !yamlContent"
              @click="isWordWrap = !isWordWrap"
              :title="isWordWrap ? '已开启自动换行 (点击关闭)' : '已关闭换行，保持整齐缩进 (点击开启)'"
            >
              <WrapText class="w-3.5 h-3.5 opacity-80" />
              <span class="text-[11px] sm:text-xs">{{ isWordWrap ? '取消换行' : '换行' }}</span>
            </Button>
          </div>

          <div class="hidden sm:flex items-center gap-1.5 shrink-0">
            <!-- 全屏切换按钮（小屏手机已是全屏，大屏才展示） -->
            <Button
              size="icon"
              variant="ghost"
              class="h-8 w-8 text-muted-foreground hover:text-foreground"
              :title="isFullscreen ? '还原窗口' : '最大化全屏'"
              @click="isFullscreen = !isFullscreen"
            >
              <Minimize2 v-if="isFullscreen" class="w-4 h-4" />
              <Maximize2 v-else class="w-4 h-4" />
            </Button>

            <!-- 桌面端关闭按钮 -->
            <Button
              size="icon"
              variant="ghost"
              class="h-8 w-8 text-muted-foreground hover:text-foreground rounded-lg ml-0.5"
              @click="handleClose"
              title="关闭窗口 (Esc)"
            >
              <X class="w-4 h-4" />
            </Button>
          </div>
        </div>
      </div>

      <!-- 编辑器核心视口区 -->
      <div class="flex-1 relative overflow-hidden bg-[#1e1e1e] flex flex-col min-h-0 w-full">
        <!-- 加载动画 -->
        <div v-if="loading" class="absolute inset-0 z-10 flex flex-col items-center justify-center gap-3 bg-[#1e1e1e]/80 backdrop-blur-xs text-muted-foreground">
          <Loader2 class="w-7 h-7 animate-spin text-primary" />
          <span class="text-xs font-mono">正在检索并解析应用清单 YAML...</span>
        </div>

        <!-- 错误提示区 -->
        <div v-else-if="loadError" class="absolute inset-0 z-10 flex flex-col items-center justify-center gap-3 p-4 sm:p-6 text-center bg-[#1e1e1e]">
          <AlertCircle class="w-9 h-9 sm:w-10 sm:h-10 text-destructive/80" />
          <div class="space-y-1 max-w-md">
            <h4 class="text-xs sm:text-sm font-semibold text-foreground">加载应用 YAML 失败</h4>
            <p class="text-[11px] sm:text-xs text-muted-foreground leading-relaxed">{{ loadError }}</p>
          </div>
          <Button size="sm" variant="outline" class="gap-1.5 text-xs h-8" @click="loadYaml">
            <RefreshCw class="w-3.5 h-3.5" />
            <span>重试加载</span>
          </Button>
        </div>

        <!-- Monaco Editor 实例 (小屏优化：lineNumbers 留白减少，字体适配) -->
        <div v-show="!loading && !loadError" class="flex-1 w-full h-full">
          <VueMonacoEditor
            v-if="yamlContent"
            :value="yamlContent"
            language="yaml"
            theme="vs-dark"
            :options="{
              readOnly: true,
              fontSize: 12,
              lineNumbers: 'on',
              lineNumbersMinChars: 3,
              glyphMargin: false,
              scrollBeyondLastLine: false,
              automaticLayout: true,
              tabSize: 2,
              wordWrap: isWordWrap ? 'on' : 'off',
              renderWhitespace: 'selection',
              contextmenu: true,
              cursorBlinking: 'smooth',
              cursorStyle: 'line',
              minimap: { enabled: isFullscreen },
              fontFamily: `'JetBrains Mono Variable', 'JetBrains Mono', Consolas, 'Courier New', monospace`,
              scrollbar: {
                verticalScrollbarSize: 8,
                horizontalScrollbarSize: 8
              }
            }"
            style="height: 100%; width: 100%"
            @mount="handleEditorMount"
          />
        </div>
      </div>

      <!-- 底部状态与操作栏 (移动端自适应收纳与按键排布) -->
      <div class="px-3 sm:px-4 py-2 sm:py-2.5 border-t border-border/60 bg-muted/20 flex flex-row items-center justify-between gap-2 shrink-0 text-xs">
        <!-- 左侧状态信息：小屏精简显示 -->
        <div class="flex items-center gap-2 sm:gap-3 text-muted-foreground font-mono text-[10px] sm:text-[11px] truncate min-w-0">
          <span class="flex items-center gap-1 shrink-0">
            <FileCode class="w-3 h-3 sm:w-3.5 sm:h-3.5 opacity-70" />
            <span>{{ lineCount }}行</span>
          </span>
          <span class="opacity-50">•</span>
          <span class="shrink-0">{{ fileSizeFormatted }}</span>
          <template v-if="app?.homepage">
            <span class="hidden md:inline opacity-50">|</span>
            <a
              :href="app.homepage"
              target="_blank"
              class="hidden md:inline-flex items-center gap-1 text-muted-foreground hover:text-primary transition-colors underline-offset-2 hover:underline shrink-0"
            >
              <span>项目主页</span>
              <ExternalLink class="w-3 h-3 opacity-60" />
            </a>
          </template>
        </div>

        <!-- 右侧操作按钮组 -->
        <div class="flex items-center gap-2 shrink-0 ml-auto">
          <Button size="sm" variant="ghost" class="hidden sm:inline-flex h-7 sm:h-8 px-2.5 sm:px-3 text-xs" @click="handleClose">
            关闭
          </Button>
          <Button
            size="sm"
            class="h-7 sm:h-8 px-3 text-xs gap-1.5 shadow-sm font-medium"
            @click="handleDeploy"
          >
            <Sparkles class="w-3 h-3 sm:w-3.5 sm:h-3.5 shrink-0" />
            <span>部署此应用</span>
          </Button>
        </div>
      </div>
    </DialogContent>
  </Dialog>
</template>
