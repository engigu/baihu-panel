<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { api, type WebUI } from '@/api'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { CheckCircle2, Trash2, MonitorPlay } from 'lucide-vue-next'
import { toast } from 'vue-sonner'


const webuis = ref<WebUI[]>([])
const activeWebUI = ref<string>('default')
const loading = ref(true)
const uploading = ref(false)
const fileInput = ref<HTMLInputElement | null>(null)

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
  const nameLower = file.name.toLowerCase()
  if (!nameLower.endsWith('.zip') && !nameLower.endsWith('.tar.gz') && !nameLower.endsWith('.tgz')) {
    toast.error('仅支持上传 .zip 或 .tar.gz 格式的前端包')
    return
  }

  uploading.value = true
  try {
    await api.webui.upload(file)
    toast.success('上传成功', { description: '新的前端包已安装' })
    await loadData()
  } catch (err: any) {
    toast.error('上传失败', { description: err.message })
  } finally {
    uploading.value = false
    if (fileInput.value) {
      fileInput.value.value = ''
    }
  }
}

const triggerUpload = () => {
  fileInput.value?.click()
}

const activateWebUI = async (name: string) => {
  if (name === activeWebUI.value) return
  
  try {
    await api.webui.setActive(name)
    toast.success('切换成功', { description: '正在重载前端界面...' })
    activeWebUI.value = name
    // Reload page after a short delay to apply the new UI
    setTimeout(() => {
      window.location.reload()
    }, 1000)
  } catch (err: any) {
    toast.error('切换失败', { description: err.message })
  }
}

const deleteWebUI = async (name: string) => {
  if (!confirm(`确定要删除前端包 "${name}" 吗？此操作不可恢复。`)) return
  
  try {
    await api.webui.delete(name)
    toast.success('删除成功')
    
    // 检查删除后当前页是否为空
    if (paginatedWebuis.value.length === 1 && currentPage.value > 1) {
      currentPage.value--
    }
    
    await loadData()
  } catch (err: any) {
    toast.error('删除失败', { description: err.message })
  }
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
  <div class="space-y-4">
    <!-- Warning Tip -->
    <div class="rounded-md bg-yellow-500/10 border border-yellow-500/20 p-2.5 text-[10px] text-yellow-600 dark:text-yellow-400 leading-relaxed mt-2">
      <strong>风险提示：</strong>自定义前端可能导致界面无法访问。如果不慎应用了错误或不兼容的包导致白屏，请进入终端执行 <code class="bg-yellow-500/20 px-1 py-0.5 rounded mx-0.5 font-mono">baihu webui reset</code> 一键恢复默认内置界面。
    </div>

    <!-- Hidden file input for uploading -->
    <input 
      type="file" 
      ref="fileInput" 
      class="hidden" 
      accept=".zip,.tar.gz,.tgz" 
      @change="handleFileUpload" 
    />

    <!-- WebUI Table / Custom List Layout -->
    <div class="rounded-lg border bg-card overflow-hidden">
      <!-- 表头 (仅在大屏显示) -->
      <div class="hidden sm:flex items-center gap-4 px-4 py-1.5 border-b bg-muted/20 text-xs text-muted-foreground font-medium">
        <span class="w-32 shrink-0 pl-1">名称</span>
        <span class="flex-1 min-w-0">描述</span>
        <span class="w-20 shrink-0 text-center">版本</span>
        <span class="w-20 shrink-0">作者</span>
        <span class="w-20 shrink-0 text-center">状态</span>
        <span class="w-24 shrink-0 text-right pr-1">操作</span>
      </div>

      <!-- 列表内容 -->
      <div class="divide-y text-sm">
        <template v-if="loading">
          <!-- Skeleton rows -->
          <div v-for="i in 3" :key="i" class="flex flex-col sm:flex-row sm:items-center gap-3 sm:gap-4 px-4 py-3 sm:py-2 hover:bg-muted/30 transition-colors">
            <div class="w-full sm:w-32 shrink-0 sm:pl-1 flex justify-between"><div class="h-4 bg-muted rounded w-20 animate-pulse"></div></div>
            <div class="w-full sm:flex-1 min-w-0"><div class="h-4 bg-muted rounded w-full animate-pulse"></div></div>
            <div class="hidden sm:block w-20 shrink-0"><div class="h-4 bg-muted rounded w-10 mx-auto animate-pulse"></div></div>
            <div class="hidden sm:block w-20 shrink-0"><div class="h-4 bg-muted rounded w-12 animate-pulse"></div></div>
            <div class="hidden sm:block w-20 shrink-0"><div class="h-4 bg-muted rounded w-12 mx-auto animate-pulse"></div></div>
            <div class="w-full sm:w-24 shrink-0 sm:pr-1 flex justify-end"><div class="h-7 bg-muted rounded w-16 animate-pulse"></div></div>
          </div>
        </template>
        <template v-else>
          <div v-if="webuis.length === 0" class="text-center py-12 text-muted-foreground text-xs">
            暂无前端资源包
          </div>
          <div v-for="item in paginatedWebuis" :key="item.name"
            class="flex flex-col sm:flex-row sm:items-center gap-2 sm:gap-4 px-4 py-3 sm:py-1.5 hover:bg-muted/30 transition-colors">
            <!-- 名称与移动端状态 -->
            <div class="w-full sm:w-32 shrink-0 sm:pl-1 font-medium flex items-center justify-between sm:justify-start gap-2 overflow-hidden">
              <div class="flex items-center gap-2 overflow-hidden">
                <MonitorPlay class="w-3.5 h-3.5 text-muted-foreground shrink-0" />
                <span class="truncate" :title="item.name">{{ item.name }}</span>
              </div>
              <!-- 移动端状态展示 -->
              <div class="sm:hidden shrink-0 flex items-center">
                <Badge v-if="activeWebUI === item.name" variant="default" class="bg-primary/10 text-primary hover:bg-primary/20 border-primary/20 font-normal py-0 h-5 text-[10px]">
                  <CheckCircle2 class="w-3 h-3 mr-1" /> 使用中
                </Badge>
              </div>
            </div>
            <!-- 描述 -->
            <div class="w-full sm:flex-1 min-w-0 text-muted-foreground text-xs truncate" :title="item.description || '无描述'">
              {{ item.description || '无描述' }}
            </div>
            <!-- 移动端：版本与作者；PC端：分成两列 -->
            <div class="flex items-center justify-between sm:contents mt-1 sm:mt-0 text-xs text-muted-foreground">
              <div class="flex items-center gap-4 sm:contents">
                <div class="flex items-center gap-1 sm:w-20 shrink-0 sm:justify-center">
                  <span class="sm:hidden text-muted-foreground/70">版本:</span>
                  <Badge variant="outline" class="font-mono text-[9px] px-1 py-0 h-4">v{{ item.version || '1.0' }}</Badge>
                </div>
                <div class="flex items-center gap-1 sm:w-20 shrink-0 truncate" :title="item.author || 'Unknown'">
                  <span class="sm:hidden text-muted-foreground/70">作者:</span>
                  <span class="truncate">{{ item.author || 'Unknown' }}</span>
                </div>
              </div>
              
              <!-- 移动端的操作按钮（放在这里与版本同行） -->
              <div class="sm:hidden flex items-center gap-2">
                <Button 
                  v-if="activeWebUI !== item.name" 
                  variant="outline" 
                  size="sm"
                  class="h-6 text-[10px] px-2 py-0"
                  @click="activateWebUI(item.name)"
                >
                  启用
                </Button>
                <Button 
                  v-if="item.name !== 'default' && activeWebUI !== item.name" 
                  variant="ghost" 
                  size="icon" 
                  class="h-6 w-6 text-destructive hover:text-destructive hover:bg-destructive/10 shrink-0"
                  @click="deleteWebUI(item.name)"
                >
                  <Trash2 class="w-3 h-3" />
                </Button>
              </div>
            </div>
            <!-- PC端状态 -->
            <div class="hidden sm:flex w-20 shrink-0 justify-center">
              <Badge v-if="activeWebUI === item.name" variant="default" class="bg-primary/10 text-primary hover:bg-primary/20 border-primary/20 font-normal py-0 h-5 text-[10px]">
                <CheckCircle2 class="w-3 h-3 mr-1" /> 使用中
              </Badge>
              <span v-else class="text-muted-foreground text-xs">-</span>
            </div>
            <!-- PC端操作 -->
            <div class="hidden sm:flex w-24 shrink-0 pr-1 justify-end items-center gap-2">
              <Button 
                v-if="activeWebUI !== item.name" 
                variant="outline" 
                size="sm"
                class="h-7 text-xs px-2 py-0"
                @click="activateWebUI(item.name)"
              >
                启用
              </Button>
              <Button 
                v-else 
                variant="outline" 
                disabled 
                size="sm"
                class="h-7 text-xs px-2 py-0 opacity-50"
              >
                已激活
              </Button>
              
              <Button 
                v-if="item.name !== 'default' && activeWebUI !== item.name" 
                variant="ghost" 
                size="icon" 
                class="h-7 w-7 text-destructive hover:text-destructive hover:bg-destructive/10 shrink-0"
                @click="deleteWebUI(item.name)"
                title="删除此包"
              >
                <Trash2 class="w-3.5 h-3.5" />
              </Button>
              <div v-else class="w-7 shrink-0"></div>
            </div>
          </div>
        </template>
      </div>
    </div>

    <!-- Pagination Controls -->
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
        >
          下一页
        </Button>
      </div>
    </div>
  </div>
</template>
