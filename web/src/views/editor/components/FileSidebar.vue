<script setup lang="ts">
import { ref, computed } from 'vue'
import { Button } from '@/components/ui/button'
import { RefreshCw, FileUp, FileArchive, Plus, AlertCircle } from 'lucide-vue-next'
import FileTreeNode from '@/components/FileTreeNode.vue'
import BaihuDialog from '@/components/ui/BaihuDialog.vue'
import { type FileNode } from '@/api'

const props = defineProps<{
  fileTree: FileNode[]
  expandedDirs: Set<string>
  selectedPath: string | null
  isRefreshing?: boolean
}>()

const emit = defineEmits<{
  refresh: []
  create: [path: string]
  select: [node: FileNode]
  delete: [path: string]
  download: [path: string]
  downloadZip: [path: string]
  move: [oldPath: string, newPath: string]
  rename: [path: string]
  duplicate: [path: string]
  uploadArchive: [file: File, target: string]
  uploadFiles: [files: FileList, paths: string[], target: string]
}>()

const archiveInputRef = ref<HTMLInputElement | null>(null)
const filesInputRef = ref<HTMLInputElement | null>(null)
const uploadTargetDir = ref('')

// 确认框状态
const confirmUpload = ref<{
  show: boolean
  title: string
  message: string
  onConfirm: () => void
}>({
  show: false,
  title: '',
  message: '',
  onConfirm: () => {}
})

const currentTargetDir = computed(() => {
  if (!props.selectedPath) return ''
  
  // 深度优先搜索以确定选中的是目录还是文件
  const findNode = (nodes: FileNode[], path: string): FileNode | null => {
    for (const n of nodes) {
      if (n.path === path) return n
      if (n.children) {
        const found = findNode(n.children, path)
        if (found) return found
      }
    }
    return null
  }
  
  const node = findNode(props.fileTree, props.selectedPath)
  if (node && node.isDir) {
    return node.path
  }
  
  // 如果选中的是文件，则返回它所在的目录
  const parts = props.selectedPath.split('/')
  parts.pop() // 移除文件名
  return parts.join('/')
})

function triggerArchiveUpload(targetDir = '') {
  uploadTargetDir.value = targetDir
  if (archiveInputRef.value) {
    archiveInputRef.value.value = ''
    archiveInputRef.value.click()
  }
}

function triggerFilesUpload(targetDir = '') {
  uploadTargetDir.value = targetDir
  if (filesInputRef.value) {
    filesInputRef.value.value = ''
    filesInputRef.value.click()
  }
}

function handleArchiveUpload(e: Event) {
  const input = e.target as HTMLInputElement
  const file = input.files?.[0]
  if (file) {
    const dirName = uploadTargetDir.value || '根目录'
    confirmUpload.value = {
      show: true,
      title: '确认导入压缩包',
      message: `即将把压缩包 [${file.name}] 导入到目录 [ ${dirName} ]。确认导入吗？`,
      onConfirm: () => {
        emit('uploadArchive', file, uploadTargetDir.value)
      }
    }
  }
}

function handleFilesUpload(e: Event) {
  const input = e.target as HTMLInputElement
  const fileList = input.files
  if (fileList && fileList.length > 0) {
    const files: File[] = []
    const paths: string[] = []
    for (let i = 0; i < fileList.length; i++) {
      const f = fileList.item(i)
      if (f) {
        files.push(f)
        paths.push((f as any).webkitRelativePath || f.name)
      }
    }
    const dirName = uploadTargetDir.value || '根目录'
    confirmUpload.value = {
      show: true,
      title: '确认上传文件',
      message: `即将把 ${files.length} 个文件/文件夹上传到目录 [ ${dirName} ]。确认上传吗？`,
      onConfirm: () => {
        emit('uploadFiles', files as unknown as FileList, paths, uploadTargetDir.value)
      }
    }
  }
}
</script>

<template>
  <div class="w-full lg:w-56 h-48 lg:h-auto flex-shrink-0 border rounded-md flex flex-col">
    <div class="flex items-center justify-between p-2 border-b">
      <span class="text-xs font-medium">脚本文件</span>
      <div class="flex gap-1">
        <Button variant="ghost" size="icon" class="h-6 w-6" @click="emit('refresh')" :disabled="isRefreshing" title="刷新">
          <RefreshCw class="h-3 w-3" :class="{ 'animate-spin': isRefreshing }" />
        </Button>
        <Button variant="ghost" size="icon" class="h-6 w-6" @click="triggerFilesUpload(currentTargetDir)" :title="currentTargetDir ? `上传文件到: ${currentTargetDir}` : '上传文件到根目录'">
          <FileUp class="h-3 w-3" />
        </Button>
        <Button variant="ghost" size="icon" class="h-6 w-6" @click="triggerArchiveUpload(currentTargetDir)" :title="currentTargetDir ? `导入压缩包到: ${currentTargetDir}` : '导入压缩包到根目录'">
          <FileArchive class="h-3 w-3" />
        </Button>
        <Button variant="ghost" size="icon" class="h-6 w-6" @click="emit('create', currentTargetDir)" :title="currentTargetDir ? `在 ${currentTargetDir} 中新建` : '在根目录新建'">
          <Plus class="h-3 w-3" />
        </Button>
      </div>
      <input ref="archiveInputRef" type="file" accept=".zip,.tar,.gz,.tgz" class="hidden" @change="handleArchiveUpload" />
      <input ref="filesInputRef" type="file" multiple class="hidden" @change="handleFilesUpload" />
    </div>
    <div class="flex-1 overflow-auto p-1 text-[13px]">
      <div v-if="fileTree.length === 0" class="text-xs text-muted-foreground text-center py-4">
        暂无文件
      </div>
      <FileTreeNode v-for="node in fileTree" :key="node.path" :node="node" :expanded-dirs="expandedDirs"
        :selected-path="selectedPath" 
        @select="n => emit('select', n)" 
        @delete="p => emit('delete', p)" 
        @create="p => emit('create', p)"
        @download-file="p => emit('download', p)" 
        @download-zip="p => emit('downloadZip', p)"
        @move="(o, n) => emit('move', o, n)" 
        @rename="p => emit('rename', p)" 
        @duplicate="p => emit('duplicate', p)" />
    </div>

    <!-- 上传确认对话框 -->
    <BaihuDialog v-model:open="confirmUpload.show" :title="confirmUpload.title" icon="Upload" size="sm">
      <div class="flex flex-col sm:flex-row items-center sm:items-start gap-4 p-1">
        <div class="h-12 w-12 rounded-full bg-primary/10 flex items-center justify-center shrink-0">
          <AlertCircle class="h-6 w-6 text-primary" />
        </div>
        <div class="flex-1 text-center sm:text-left">
          <p class="text-sm text-foreground/90 leading-relaxed font-medium">确认操作？</p>
          <p class="text-[13px] text-muted-foreground mt-1 break-all">
            {{ confirmUpload.message }}
          </p>
        </div>
      </div>
      <template #footer>
        <div class="flex flex-col-reverse sm:flex-row gap-2 w-full sm:w-auto mt-2 sm:mt-0">
          <Button variant="outline" @click="confirmUpload.show = false" class="w-full sm:w-24">取消</Button>
          <Button @click="confirmUpload.show = false; confirmUpload.onConfirm()" class="w-full sm:w-auto px-6">立即开始</Button>
        </div>
      </template>
    </BaihuDialog>
  </div>
</template>
