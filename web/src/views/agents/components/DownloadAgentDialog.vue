<script setup lang="ts">
import { ref, computed } from 'vue'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import {
  Download,
  Terminal,
  Monitor,
  Command,
  Smartphone,
  Info,
  ExternalLink,
  Copy,
  Check,
  Cpu
} from 'lucide-vue-next'
import { api } from '@/api'
import { copyToClipboard } from '@/utils/clipboard'
import { toast } from 'vue-sonner'

const isOpen = ref(false)

const props = defineProps<{
  agentVersion: string
  platforms: { os: string; arch: string; filename: string }[]
}>()

const defaultMockPlatforms = [
  { os: 'windows', arch: 'amd64', filename: 'baihu-agent-windows-amd64.zip' },
  { os: 'linux', arch: 'amd64', filename: 'baihu-agent-linux-amd64.tar.gz' },
  { os: 'linux', arch: 'arm64', filename: 'baihu-agent-linux-arm64.tar.gz' },
  { os: 'darwin', arch: 'amd64', filename: 'baihu-agent-darwin-amd64.tar.gz' },
  { os: 'darwin', arch: 'arm64', filename: 'baihu-agent-darwin-arm64.tar.gz' }
]

const displayPlatforms = computed(() => {
  if (props.platforms && props.platforms.length > 0) {
    return props.platforms
  }
  return defaultMockPlatforms
})

function openDialog() {
  isOpen.value = true
}

function downloadAgent(os: string, arch: string) {
  window.open(api.agents.downloadUrl(os, arch), '_blank')
}

function getPlatformLabel(os: string, arch: string) {
  const osLabels: Record<string, string> = { linux: 'Linux', windows: 'Windows', darwin: 'macOS', android: 'Android' }
  const archLabels: Record<string, string> = { amd64: 'x64', arm64: 'ARM64', '386': 'x86' }
  return `${osLabels[os] || os} ${archLabels[arch] || arch}`
}

function getPlatformIcon(os: string) {
  switch (os.toLowerCase()) {
    case 'windows':
      return Monitor
    case 'darwin':
    case 'macos':
      return Command
    case 'android':
      return Smartphone
    default:
      return Terminal
  }
}

function getPlatformFormat(filename: string) {
  if (filename?.endsWith('.zip')) return '.zip'
  if (filename?.endsWith('.tar.gz')) return '.tar.gz'
  return ''
}

const copiedCmd = ref<string | null>(null)

async function handleCopy(code: string) {
  const success = await copyToClipboard(code)
  if (success) {
    copiedCmd.value = code
    toast.success('命令已复制到剪贴板')
    setTimeout(() => {
      if (copiedCmd.value === code) copiedCmd.value = null
    }, 2000)
  }
}

defineExpose({ openDialog })
</script>

<template>
  <Dialog v-model:open="isOpen">
    <DialogContent class="sm:max-w-xl max-h-[90vh] overflow-y-auto">
      <DialogHeader class="pb-2">
        <div class="flex items-center gap-2">
          <div class="p-2 rounded-xl bg-primary/10 text-primary shrink-0">
            <Cpu class="w-5 h-5" />
          </div>
          <div>
            <DialogTitle class="text-base sm:text-lg flex items-center gap-2 flex-wrap">
              <span>下载 Agent 节点执行器</span>
              <Badge variant="secondary" class="font-mono text-[11px] px-1.5 py-0 bg-primary/10 text-primary border-primary/20">
                {{ agentVersion || 'main' }}
              </Badge>
            </DialogTitle>
            <DialogDescription class="text-xs pt-0.5">
              为目标服务器选择对应的操作系统与架构独立二进制节点包
            </DialogDescription>
          </div>
        </div>
      </DialogHeader>

      <div class="space-y-4 pt-1">
        <!-- 下载说明提示框 -->
        <div class="p-3 rounded-xl bg-blue-500/10 border border-blue-500/20 text-xs text-blue-600 dark:text-blue-400 space-y-1.5">
          <div class="font-semibold flex items-center gap-1.5">
            <Info class="w-3.5 h-3.5 shrink-0" />
            <span>部署提示说明</span>
          </div>
          <ul class="space-y-1 text-[11px] opacity-90 leading-relaxed pl-4 list-disc">
            <li><strong class="font-semibold">Docker 镜像部署：</strong>支持直接在此下载包含配置的预置二进制打包程序。</li>
            <li><strong class="font-semibold">单文件二进制部署：</strong>请前往 <a href="https://github.com/engigu/baihu-panel/releases" target="_blank" class="underline font-medium hover:text-blue-500 inline-flex items-center gap-0.5">GitHub Releases <ExternalLink class="w-3 h-3" /></a> 下载对应 Release。</li>
          </ul>
        </div>

        <!-- 平台下载二列卡片网格 -->
        <div class="space-y-1.5">
          <div class="text-xs font-semibold text-foreground flex items-center justify-between">
            <span>支持的操作系统与架构</span>
            <span class="text-[11px] text-muted-foreground font-normal">点击右侧按钮直接下载</span>
          </div>

          <div v-if="displayPlatforms.length > 0" class="grid grid-cols-1 sm:grid-cols-2 gap-2.5">
            <div
              v-for="platform in displayPlatforms"
              :key="`${platform.os}-${platform.arch}`"
              class="p-2.5 rounded-xl border border-border/70 bg-muted/20 hover:bg-muted/40 transition-all flex items-center justify-between gap-2 group"
            >
              <div class="flex items-center gap-2.5 min-w-0">
                <div class="p-2 rounded-lg bg-background border border-border/60 text-foreground/80 group-hover:text-primary group-hover:border-primary/30 transition-colors shrink-0">
                  <component :is="getPlatformIcon(platform.os)" class="w-4 h-4" />
                </div>
                <div class="min-w-0">
                  <div class="text-xs font-bold text-foreground truncate">
                    {{ getPlatformLabel(platform.os, platform.arch) }}
                  </div>
                  <div class="text-[10px] text-muted-foreground font-mono truncate">
                    {{ platform.filename || `${platform.os}-${platform.arch}${getPlatformFormat(platform.filename)}` }}
                  </div>
                </div>
              </div>

              <Button
                size="sm"
                variant="outline"
                class="h-7.5 px-2.5 text-xs font-medium shadow-none shrink-0 group-hover:border-primary/40 group-hover:bg-primary/10 group-hover:text-primary transition-all gap-1"
                @click="downloadAgent(platform.os, platform.arch)"
              >
                <Download class="w-3.5 h-3.5" />
                <span>下载</span>
              </Button>
            </div>
          </div>

          <div v-else class="p-6 text-center rounded-xl border border-dashed text-xs text-muted-foreground">
            暂未发现打入镜像的本地 Agent 归档包，请前往 GitHub Releases 页面下载。
          </div>
        </div>

        <!-- 使用说明与快速启动命令 -->
        <div class="pt-2 border-t border-border/60 space-y-3">
          <div class="text-xs font-semibold text-foreground flex items-center gap-1.5">
            <Terminal class="w-3.5 h-3.5 text-primary" />
            <span>部署与快速启动指引</span>
          </div>

          <!-- 步骤导引清单 -->
          <ol class="text-xs text-muted-foreground space-y-1.5 leading-relaxed list-decimal pl-4">
            <li>下载目标平台的压缩包解压，复制配置模板：
              <code class="bg-muted px-1.5 py-0.5 rounded text-[11px] font-mono text-foreground select-all">cp config.example.ini config.ini</code>
            </li>
            <li>编辑 <code class="bg-muted px-1.5 py-0.5 rounded text-[11px] font-mono text-foreground">config.ini</code>，配置宿主面板通讯地址与节点注册 Token；</li>
            <li>执行后台守护服务启动：<code class="bg-muted px-1.5 py-0.5 rounded text-[11px] font-mono text-foreground">./baihu-agent start</code>。</li>
          </ol>

          <!-- 命令行快捷复制卡片 -->
          <div class="p-3 rounded-xl bg-muted/30 border border-border/70 space-y-2">
            <div class="text-[11px] font-medium text-foreground flex items-center justify-between">
              <span>常用控制命令集</span>
              <span class="text-[10px] text-muted-foreground">点击右侧按钮一键复制</span>
            </div>
            <div class="grid grid-cols-1 sm:grid-cols-2 gap-1.5">
              <div
                v-for="cmd in [
                  { label: '后台启动', code: 'baihu-agent start' },
                  { label: '前台运行', code: 'baihu-agent run' },
                  { label: '停止运行', code: 'baihu-agent stop' },
                  { label: '查看状态', code: 'baihu-agent status' },
                  { label: '下发任务', code: 'baihu-agent tasks' },
                  { label: '实时日志', code: 'baihu-agent logs' },
                  { label: '安装自启', code: 'baihu-agent install' },
                  { label: '卸载自启', code: 'baihu-agent uninstall' },
                  { label: '查看版本', code: 'baihu-agent version' }
                ]"
                :key="cmd.code"
                class="flex items-center justify-between p-1.5 px-2.5 rounded-lg bg-background/80 border border-border/50 text-[11px]"
              >
                <div class="flex items-center gap-2 min-w-0">
                  <span class="text-muted-foreground text-[10px] shrink-0">{{ cmd.label }}</span>
                  <code class="font-mono text-foreground truncate select-all">{{ cmd.code }}</code>
                </div>
                <button
                  type="button"
                  @click="handleCopy(cmd.code)"
                  class="text-muted-foreground hover:text-primary transition-colors p-1 shrink-0 ml-1"
                  :title="`复制 ${cmd.code}`"
                >
                  <Check v-if="copiedCmd === cmd.code" class="w-3 h-3 text-emerald-500" />
                  <Copy v-else class="w-3 h-3" />
                </button>
              </div>
            </div>
          </div>
        </div>
      </div>
    </DialogContent>
  </Dialog>
</template>
