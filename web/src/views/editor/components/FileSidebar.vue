<script setup lang="ts">
import { ref } from 'vue'
import { Button } from '@/components/ui/button'
import { RefreshCw, FolderUp, Upload, Plus } from 'lucide-vue-next'
import FileTreeNode from '@/components/FileTreeNode.vue'
import { type FileNode } from '@/api'

defineProps<{
  fileTree: FileNode[]
  expandedDirs: Set<string>
  selectedPath: string | null
}>()

const emit = defineEmits<{
  refresh: []
  create: [path: string]
  select: [node: FileNode]
  delete: [path: string]
  download: [path: string]
  move: [oldPath: string, newPath: string]
  rename: [path: string]
  duplicate: [path: string]
  uploadArchive: [file: File, target: string]
  uploadFiles: [files: FileList, paths: string[], target: string]
}>()

const archiveInputRef = ref<HTMLInputElement | null>(null)
const filesInputRef = ref<HTMLInputElement | null>(null)
const uploadTargetDir = ref('')

function triggerArchiveUpload(targetDir = '') {
  uploadTargetDir.value = targetDir
  if (archiveInputRef.value) archiveInputRef.value.click()
}

function triggerFilesUpload(targetDir = '') {
  uploadTargetDir.value = targetDir
  if (filesInputRef.value) filesInputRef.value.click()
}

function handleArchiveUpload(e: Event) {
  const input = e.target as HTMLInputElement
  const file = input.files?.[0]
  if (file) emit('uploadArchive', file, uploadTargetDir.value)
  input.value = ''
}

function handleFilesUpload(e: Event) {
  const input = e.target as HTMLInputElement
  const files = input.files
  if (files && files.length > 0) {
    const paths: string[] = []
    for (let i = 0; i < files.length; i++) {
       const f = files.item(i)
       if (f) {
         paths.push((f as any).webkitRelativePath || f.name)
       }
    }
    emit('uploadFiles', files, paths, uploadTargetDir.value)
  }
  input.value = ''
}
</script>

<template>
  <div class="w-full lg:w-56 h-48 lg:h-auto flex-shrink-0 border rounded-md flex flex-col">
    <div class="flex items-center justify-between p-2 border-b">
      <span class="text-xs font-medium">脚本文件</span>
      <div class="flex gap-1">
        <Button variant="ghost" size="icon" class="h-6 w-6" @click="emit('refresh')" title="刷新">
          <RefreshCw class="h-3 w-3" />
        </Button>
        <Button variant="ghost" size="icon" class="h-6 w-6" @click="triggerFilesUpload('')" title="上传文件/文件夹(放在根目录)">
          <FolderUp class="h-3 w-3" />
        </Button>
        <Button variant="ghost" size="icon" class="h-6 w-6" @click="triggerArchiveUpload('')" title="导入压缩包(放在根目录)">
          <Upload class="h-3 w-3" />
        </Button>
        <Button variant="ghost" size="icon" class="h-6 w-6" @click="emit('create', '')" title="新建">
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
        @move="(o, n) => emit('move', o, n)" 
        @rename="p => emit('rename', p)" 
        @duplicate="p => emit('duplicate', p)" />
    </div>
  </div>
</template>
