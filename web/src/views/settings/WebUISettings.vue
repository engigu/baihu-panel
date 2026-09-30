<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { api, type WebUI } from '@/api'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import {
  CheckCircle2,
  Trash2,
  MonitorPlay,
  Palette,
  UploadCloud,
  ExternalLink,
  Terminal,
  Copy,
  Check,
  User,
  Sparkles
} from 'lucide-vue-next'
import { toast } from 'vue-sonner'

const webuis = ref<WebUI[]>([])
const activeWebUI = ref<string>('default')
const loading = ref(true)
const uploading = ref(false)
const activatingName = ref<string | null>(null)
const fileInput = ref<HTMLInputElement | null>(null)
const copied = ref(false)

// Pagination
const currentPage = ref(1)
const pageSize = 6
const totalPages = computed(() => Math.ceil(webuis.value.length / pageSize))
const paginatedWebuis = computed(() => {
  const start = (currentPage.value - 1) * pageSize
  const end = start + pageSize
  return webuis.value.slice(start, end)
})

const loadData = async () => {
  loading.value = true
  try {
    const [listRes, siteRes] = await Promise.all([
      api.webui.list(),
      api.settings.getSite()
    ])
    webuis.value = listRes
    activeWebUI.value = siteRes.active_webui || 'default'
  } catch (err: any) {
    toast.error('加载前端包列表失败', { description: err.message })
  } finally {
    loading.value = false
  }
}

const handleFileUpload = async (event: Event) => {
  const target = event.target as HTMLInputElement
  if (!target.files || target.files.length === 0) return

  const file = target.files[0]
  if (!file) return
  await processFile(file)
  if (fileInput.value) {
    fileInput.value.value = ''
  }
}

const handleDrop = async (e: DragEvent) => {
  e.preventDefault()
  if (uploading.value) return
  const file = e.dataTransfer?.files?.[0]
  if (file) {
    await processFile(file)
  }
}

const processFile = async (file: File) => {
  const nameLower = file.name.toLowerCase()
  if (!nameLower.endsWith('.zip') && !nameLower.endsWith('.tar.gz') && !nameLower.endsWith('.tgz')) {
    toast.error('仅支持上传 .zip 或 .tar.gz 格式的前端包')
    return
  }

  uploading.value = true
  try {
    await api.webui.upload(file)
    toast.success('上传成功', { description: '新的前端包已安装至系统' })
    await loadData()
  } catch (err: any) {
    toast.error('上传失败', { description: err.message })
  } finally {
    uploading.value = false
  }
}

const triggerUpload = () => {
  fileInput.value?.click()
}

const activateWebUI = async (name: string) => {
  if (name === activeWebUI.value || activatingName.value) return
  
  activatingName.value = name
  try {
    await api.webui.setActive(name)
    toast.success('前端界面切换成功', { description: '页面即将自动重载...' })
    activeWebUI.value = name
    setTimeout(() => {
      window.location.reload()
    }, 1000)
  } catch (err: any) {
    toast.error('切换失败', { description: err.message })
  } finally {
    activatingName.value = null
  }
}

const deleteWebUI = async (name: string) => {
  if (!confirm(`确定要彻底删除前端包 "${name}" 吗？此操作不可恢复。`)) return
  
  try {
    await api.webui.delete(name)
    toast.success('删除成功')
    
    if (paginatedWebuis.value.length === 1 && currentPage.value > 1) {
      currentPage.value--
    }
    await loadData()
  } catch (err: any) {
    toast.error('删除失败', { description: err.message })
  }
}

const copyResetCommand = () => {
  navigator.clipboard.writeText('baihu webui reset')
  copied.value = true
  toast.success('重置命令已复制到剪贴板')
  setTimeout(() => {
    copied.value = false
  }, 2000)
}

defineExpose({
  triggerUpload,
  uploading
})

onMounted(() => {
  loadData()
})
</script>

<template>
  <div class="space-y-6">
    <!-- 顶部概览与应急控制台 Card -->
    <Card class="shadow-sm border-border/80">
      <CardHeader class="pb-3 pt-5 px-5">
        <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
          <div class="space-y-1">
            <div class="flex items-center gap-2.5 flex-wrap">
              <CardTitle class="text-lg font-bold flex items-center gap-2">
                <Palette class="w-4 h-4 text-primary" />
                <span>自定义前端包 (WebUI)</span>
              </CardTitle>
              <a
                href="https://engigu.github.io/baihu-panel/guide/webui"
                target="_blank"
                class="inline-flex items-center gap-1 text-xs text-primary hover:underline font-normal"
              >
                <span>开发规范文档</span>
                <ExternalLink class="w-3 h-3" />
              </a>
            </div>
            <CardDescription class="text-xs leading-relaxed pt-0.5">
              完全接管系统控制台渲染，支持加载社区或编译自定义的前端静态资源包。
            </CardDescription>
          </div>

          <!-- 上传操作按钮 -->
          <div class="shrink-0 flex items-center gap-2">
            <input 
              type="file" 
              ref="fileInput" 
              class="hidden" 
              accept=".zip,.tar.gz,.tgz" 
              @change="handleFileUpload" 
            />
            <Button 
              @click="triggerUpload" 
              :disabled="uploading" 
              size="sm" 
              class="h-8.5 px-3.5 text-xs font-medium gap-1.5 shadow-sm"
              title="上传前端静态资源包"
            >
              <template v-if="uploading">
                <svg class="animate-spin h-3.5 w-3.5" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
                  <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                  <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                </svg>
                <span>正在上传解压...</span>
              </template>
              <template v-else>
                <UploadCloud class="w-3.5 h-3.5" />
                <span>上传前端资源包</span>
              </template>
            </Button>
          </div>
        </div>
      </CardHeader>

      <!-- 应急恢复终端救生微条 -->
      <CardContent class="px-5 pb-5 pt-1">
        <div class="p-3 rounded-xl bg-muted/30 border border-border/70 flex flex-col sm:flex-row sm:items-center justify-between gap-3 text-xs">
          <div class="flex items-center gap-2 text-muted-foreground">
            <Terminal class="w-4 h-4 text-amber-500 shrink-0" />
            <span>防白屏救生：若不慎应用异常前端包导致页面白屏，可在服务器终端执行一键重置：</span>
          </div>
          <div class="flex items-center gap-2 shrink-0 self-end sm:self-auto">
            <code class="px-2.5 py-1 rounded-md bg-background/80 border border-border/80 font-mono text-[11px] text-foreground font-semibold">
              baihu webui reset
            </code>
            <Button
              variant="outline"
              size="sm"
              class="h-7 px-2 text-[11px] gap-1 hover:bg-accent"
              @click="copyResetCommand"
              title="复制重置命令"
            >
              <Check v-if="copied" class="w-3 h-3 text-emerald-500" />
              <Copy v-else class="w-3 h-3" />
              <span>{{ copied ? '已复制' : '复制' }}</span>
            </Button>
          </div>
        </div>
      </CardContent>
    </Card>

    <!-- 前端包画廊网格 (Theme Grid) -->
    <div class="space-y-4">
      <div class="flex items-center justify-between px-0.5">
        <span class="text-xs font-medium text-muted-foreground flex items-center gap-1.5">
          <Sparkles class="w-3.5 h-3.5 text-primary" />
          <span>可用前端界面包 ({{ webuis.length }})</span>
        </span>
      </div>

      <!-- 加载中骨架屏 -->
      <div v-if="loading" class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4.5">
        <div v-for="i in 3" :key="i" class="p-5 rounded-xl border border-border/60 bg-muted/20 h-44 animate-pulse space-y-3">
          <div class="h-5 bg-muted rounded w-28"></div>
          <div class="h-4 bg-muted rounded w-3/4"></div>
          <div class="h-8 bg-muted rounded w-full mt-6"></div>
        </div>
      </div>

      <!-- 实际卡片网格 -->
      <div v-else class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4.5">
        <!-- 主题包卡片 -->
        <Card
          v-for="item in paginatedWebuis"
          :key="item.name"
          :class="[
            'shadow-sm transition-all rounded-xl flex flex-col justify-between p-4.5 border',
            activeWebUI === item.name
              ? 'border-primary/50 bg-primary/5 ring-1 ring-primary/20'
              : 'border-border/70 hover:border-border hover:shadow-md bg-card'
          ]"
        >
          <div class="space-y-3">
            <!-- 头部：图标、名称、版本、激活徽章 -->
            <div class="flex items-start justify-between gap-3">
              <div class="flex items-center gap-2.5 min-w-0">
                <div
                  :class="[
                    'w-9 h-9 rounded-lg flex items-center justify-center shrink-0 border',
                    item.name === 'default'
                      ? 'bg-primary/10 border-primary/20 text-primary'
                      : 'bg-muted/50 border-border text-foreground/80'
                  ]"
                >
                  <MonitorPlay v-if="item.name === 'default'" class="w-4 h-4" />
                  <Palette v-else class="w-4 h-4" />
                </div>
                <div class="min-w-0">
                  <div class="flex items-center gap-1.5">
                    <span class="font-bold text-sm text-foreground truncate" :title="item.name">
                      {{ item.name }}
                    </span>
                    <Badge variant="outline" class="font-mono text-[9px] px-1 py-0 h-4 border-border/80 shrink-0">
                      v{{ item.version || '1.0' }}
                    </Badge>
                  </div>
                  <div class="text-[11px] text-muted-foreground flex items-center gap-1 pt-0.5 truncate">
                    <User class="w-3 h-3 shrink-0" />
                    <span class="truncate">{{ item.author || 'Baihu' }}</span>
                  </div>
                </div>
              </div>

              <!-- 生效指示徽标 -->
              <Badge
                v-if="activeWebUI === item.name"
                class="bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border-emerald-500/20 text-[10px] px-1.5 py-0.5 font-normal shadow-none shrink-0 flex items-center gap-1"
              >
                <span class="w-1.5 h-1.5 rounded-full bg-emerald-500 animate-pulse"></span>
                <span>当前使用</span>
              </Badge>
            </div>

            <!-- 描述文字 -->
            <p class="text-xs text-muted-foreground line-clamp-2 leading-relaxed min-h-[2.25rem]">
              {{ item.description || '无详细描述说明' }}
            </p>
          </div>

          <!-- 底部操作栏 -->
          <div class="pt-4 border-t border-border/50 flex items-center justify-between gap-2 mt-2">
            <div class="text-[11px] text-muted-foreground">
              <span v-if="item.name === 'default'" class="text-muted-foreground/70">系统内置</span>
              <span v-else class="text-muted-foreground/70">自定义包</span>
            </div>

            <div class="flex items-center gap-2">
              <!-- 删除按钮（仅非内置且非当前激活的主题可删除） -->
              <Button
                v-if="item.name !== 'default' && activeWebUI !== item.name"
                variant="ghost"
                size="icon"
                class="h-7.5 w-7.5 text-muted-foreground hover:text-destructive hover:bg-destructive/10"
                @click="deleteWebUI(item.name)"
                title="删除此前端包"
              >
                <Trash2 class="w-3.5 h-3.5" />
              </Button>

              <!-- 切换/激活按钮 -->
              <Button
                v-if="activeWebUI !== item.name"
                variant="outline"
                size="sm"
                class="h-7.5 px-3 text-xs font-medium hover:bg-primary hover:text-primary-foreground transition-all"
                :disabled="activatingName === item.name"
                @click="activateWebUI(item.name)"
              >
                {{ activatingName === item.name ? '切换中...' : '启用' }}
              </Button>

              <Button
                v-else
                variant="secondary"
                size="sm"
                disabled
                class="h-7.5 px-3 text-xs font-medium opacity-80 cursor-default"
              >
                <CheckCircle2 class="w-3.5 h-3.5 mr-1 text-emerald-500" />
                已生效
              </Button>
            </div>
          </div>
        </Card>

        <!-- 虚线交互式上传引导卡片 -->
        <div
          @click="triggerUpload"
          @dragover.prevent
          @dragenter.prevent
          @drop="handleDrop"
          class="border-2 border-dashed border-border/70 hover:border-primary/50 bg-muted/15 hover:bg-muted/25 rounded-xl transition-all cursor-pointer flex flex-col items-center justify-center p-6 text-center group min-h-[170px]"
        >
          <div class="w-10 h-10 rounded-full bg-primary/10 text-primary flex items-center justify-center mb-2.5 group-hover:scale-105 transition-transform">
            <UploadCloud class="w-5 h-5" />
          </div>
          <div class="text-xs font-semibold text-foreground">
            上传并安装前端包
          </div>
          <p class="text-[11px] text-muted-foreground pt-1">
            点击或将 .zip / .tar.gz 拖拽至此处
          </p>
        </div>
      </div>
    </div>

    <!-- 分页控件 -->
    <div v-if="totalPages > 1" class="flex items-center justify-between pt-2">
      <p class="text-xs text-muted-foreground">
        共 {{ webuis.length }} 个前端包
      </p>
      <div class="flex items-center space-x-2">
        <Button 
          variant="outline" 
          size="sm" 
          :disabled="currentPage === 1" 
          @click="currentPage--"
          class="h-7.5 text-xs"
        >
          上一页
        </Button>
        <div class="text-xs font-medium">
          第 {{ currentPage }} / {{ totalPages }} 页
        </div>
        <Button 
          variant="outline" 
          size="sm" 
          :disabled="currentPage === totalPages" 
          @click="currentPage++"
          class="h-7.5 text-xs"
        >
          下一页
        </Button>
      </div>
    </div>
  </div>
</template>
