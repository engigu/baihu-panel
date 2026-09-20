<script setup lang="ts">
import BaihuDialog from '@/components/ui/BaihuDialog.vue'
import { Button } from '@/components/ui/button'
import { Copy, Download, Share2 } from 'lucide-vue-next'
import { copyToClipboard } from '@/utils/clipboard'
import { toast } from 'vue-sonner'

const props = defineProps<{
  open: boolean
  appName?: string
  yamlContent: string
}>()

const emit = defineEmits<{
  'update:open': [value: boolean]
}>()

function copyYaml() {
  if (!props.yamlContent) return
  copyToClipboard(props.yamlContent).then((success) => {
    if (success) {
      toast.success('YML 配置已复制到剪贴板')
    } else {
      toast.error('复制失败，请手动选择文本')
    }
  })
}

function downloadYaml() {
  if (!props.yamlContent) return
  try {
    const blob = new Blob([props.yamlContent], { type: 'text/yaml;charset=utf-8;' })
    const url = URL.createObjectURL(blob)
    const link = document.createElement('a')
    const fileName = props.appName ? `${props.appName.toLowerCase().replace(/\s+/g, '-')}.app.yaml` : 'app.yaml'
    link.setAttribute('href', url)
    link.setAttribute('download', fileName)
    document.body.appendChild(link)
    link.click()
    document.body.removeChild(link)
    URL.revokeObjectURL(url)
    toast.success(`已开始下载配置文件 ${fileName}`)
  } catch {
    toast.error('下载配置文件失败')
  }
}
</script>

<template>
  <BaihuDialog :open="open" title="导出应用配置" @update:open="$emit('update:open', $event)">
    <div class="space-y-4 py-2">
      <div class="text-xs text-muted-foreground leading-relaxed flex items-center gap-1.5">
        <Share2 class="h-3.5 w-3.5 text-blue-500 shrink-0" />
        <span>您可以复制 YML 文本或直接下载配置文件，以便导入到白虎应用市场或在其他机器部署：</span>
      </div>
      <div class="relative">
        <textarea
          readonly
          :value="yamlContent"
          rows="10"
          class="w-full p-4 font-mono text-xs leading-relaxed bg-muted/40 border border-muted-foreground/15 rounded-lg resize-none select-all focus:outline-none focus:ring-1 focus:ring-primary shadow-inner custom-scrollbar"
        />
      </div>
    </div>
    <template #footer>
      <Button variant="ghost" size="sm" @click="$emit('update:open', false)">关闭</Button>
      <Button variant="outline" size="sm" class="gap-1.5 font-medium" @click="downloadYaml">
        <Download class="h-3.5 w-3.5" />
        <span>下载 app.yaml</span>
      </Button>
      <Button size="sm" class="gap-1.5 shadow-md shadow-primary/20 font-medium" @click="copyYaml">
        <Copy class="h-3.5 w-3.5" />
        <span>一键复制 YML</span>
      </Button>
    </template>
  </BaihuDialog>
</template>
