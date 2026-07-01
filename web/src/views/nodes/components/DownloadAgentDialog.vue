<script setup lang="ts">
import { ref } from 'vue'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription } from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Download } from 'lucide-vue-next'
import { api } from '@/api'

const isOpen = ref(false)

const props = defineProps<{
  agentVersion: string
  platforms: { os: string; arch: string; filename: string }[]
}>()

function openDialog() {
  isOpen.value = true
}

function downloadAgent(os: string, arch: string) {
  window.open(api.agents.downloadUrl(os, arch), '_blank')
}

function getPlatformLabel(os: string, arch: string) {
  const osLabels: Record<string, string> = { linux: 'Linux', windows: 'Windows', darwin: 'macOS' }
  const archLabels: Record<string, string> = { amd64: 'x64', arm64: 'ARM64', '386': 'x86' }
  return `${osLabels[os] || os} ${archLabels[arch] || arch}`
}

defineExpose({ openDialog })
</script>

<template>
  <Dialog v-model:open="isOpen">
    <DialogContent class="sm:max-w-lg">
      <DialogHeader>
        <DialogTitle>下载 Agent</DialogTitle>
        <DialogDescription>当前版本: {{ agentVersion }}</DialogDescription>
      </DialogHeader>
      <div class="space-y-4">
        <div class="bg-blue-500/10 text-blue-600 dark:text-blue-400 p-3 rounded-md text-sm border border-blue-500/20">
          <p class="font-medium mb-1">💡 下载说明：</p>
          <ul class="list-disc list-inside space-y-1 text-xs opacity-90">
            <li>若主程序为 <strong class="font-semibold">Docker 部署</strong>：支持直接在此处下载包含配置的自动打包程序。</li>
            <li>若主程序为 <strong class="font-semibold">单文件二进制部署</strong>：面板无法直接提供完整打包下载，请前往 <a href="https://github.com/engigu/baihu-panel/releases" target="_blank" class="underline font-medium hover:text-blue-500 transition-colors">GitHub Releases</a> 手动下载对应的 Agent。</li>
          </ul>
        </div>
        <div class="space-y-2">
          <div v-for="platform in platforms" :key="`${platform.os}-${platform.arch}`"
            class="flex items-center justify-between p-3 border rounded-lg hover:bg-muted/50 transition-colors">
            <span class="font-medium">{{ getPlatformLabel(platform.os, platform.arch) }}</span>
            <Button size="sm" @click="downloadAgent(platform.os, platform.arch)">
              <Download class="h-4 w-4 mr-1.5" />下载
            </Button>
          </div>
        </div>
        <div class="border-t pt-4">
          <h4 class="font-medium mb-2">使用说明</h4>
          <ol class="text-sm text-muted-foreground space-y-1.5 list-decimal list-inside">
            <li>下载对应平台的 Agent 压缩包并解压</li>
            <li>复制 <code class="bg-muted px-1.5 py-0.5 rounded text-foreground">config.example.ini</code> 为 <code
                class="bg-muted px-1.5 py-0.5 rounded text-foreground">config.ini</code></li>
            <li>编辑 <code class="bg-muted px-1.5 py-0.5 rounded text-foreground">config.ini</code>，填写服务器地址和注册令牌</li>
            <li>运行 <code class="bg-muted px-1.5 py-0.5 rounded text-foreground">./baihu-agent start</code> 启动（后台运行）
            </li>
          </ol>
          <div class="mt-3 text-sm text-muted-foreground">
            <p class="font-medium text-foreground mb-1.5">常用命令：</p>
            <div class="space-y-1">
              <div><code class="bg-muted px-1.5 py-0.5 rounded text-foreground text-xs">baihu-agent start</code> <span
                  class="text-xs">- 后台启动</span></div>
              <div><code class="bg-muted px-1.5 py-0.5 rounded text-foreground text-xs">baihu-agent stop</code> <span
                  class="text-xs">- 停止运行</span></div>
              <div><code class="bg-muted px-1.5 py-0.5 rounded text-foreground text-xs">baihu-agent status</code>
                <span class="text-xs">- 查看状态</span>
              </div>
              <div><code class="bg-muted px-1.5 py-0.5 rounded text-foreground text-xs">baihu-agent logs</code> <span
                  class="text-xs">- 查看日志</span></div>
              <div><code class="bg-muted px-1.5 py-0.5 rounded text-foreground text-xs">baihu-agent run</code> <span
                  class="text-xs">- 前台运行</span></div>
            </div>
          </div>
        </div>
      </div>
    </DialogContent>
  </Dialog>
</template>
