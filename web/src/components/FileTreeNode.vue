<script setup lang="ts">
import { computed, ref } from 'vue'
import { Folder, File, ChevronRight, ChevronDown, Trash2, Copy as CopyIcon } from 'lucide-vue-next'
import type { FileNode } from '@/api'

defineOptions({
  name: 'FileTreeNode'
})

const props = defineProps<{
  node: FileNode
  expandedDirs: Set<string>
  selectedPath: string | null
  depth?: number
}>()

const emit = defineEmits<{
  select: [node: FileNode]
  delete: [path: string]
  create: [parentDir: string]
  move: [oldPath: string, newPath: string]
  rename: [path: string]
  downloadFile: [path: string]
  duplicate: [path: string]
}>()

const depth = computed(() => props.depth ?? 0)
const isExpanded = computed(() => props.expandedDirs.has(props.node.path))
const isSelected = computed(() => props.selectedPath === props.node.path)
const isDragOver = ref(false)

function handleSelect() {
  emit('select', props.node)
}



function handleDragStart(e: DragEvent) {
  e.dataTransfer?.setData('text/plain', props.node.path)
  e.dataTransfer!.effectAllowed = 'move'
}

function handleDragOver(e: DragEvent) {
  if (!props.node.isDir) return
  e.preventDefault()
  e.dataTransfer!.dropEffect = 'move'
  isDragOver.value = true
}

function handleDragLeave() {
  isDragOver.value = false
}

function handleDrop(e: DragEvent) {
  e.preventDefault()
  isDragOver.value = false
  if (!props.node.isDir) return

  const sourcePath = e.dataTransfer?.getData('text/plain')
  if (!sourcePath || sourcePath === props.node.path) return

  // 不能移动到自己的子目录
  if (props.node.path.startsWith(sourcePath + '/')) return

  const fileName = sourcePath.split('/').pop()
  const newPath = props.node.path ? `${props.node.path}/${fileName}` : fileName

  if (newPath !== sourcePath) {
    emit('move', sourcePath, newPath!)
  }
}
</script>

<template>
  <div>
    <div :class="[
      'flex items-center gap-1 py-0.5 px-1 rounded cursor-pointer text-xs hover:bg-muted group',
      isSelected && 'bg-accent',
      isDragOver && 'bg-blue-500/20 ring-1 ring-blue-500'
    ]" :style="{ paddingLeft: depth * 12 + 4 + 'px' }" draggable="true" @click="handleSelect"
      @dragstart="handleDragStart" @dragover="handleDragOver" @dragleave="handleDragLeave" @drop="handleDrop">
      <template v-if="node.isDir">
        <ChevronDown v-if="isExpanded" class="h-3 w-3 flex-shrink-0" />
        <ChevronRight v-else class="h-3 w-3 flex-shrink-0" />
      </template>
      <span v-else class="w-3" />
      <Folder v-if="node.isDir" class="h-3 w-3 text-yellow-500 flex-shrink-0" />
      <File v-else class="h-3 w-3 text-blue-500 flex-shrink-0" />
      <span class="truncate">{{ node.name }}</span>
      <div v-if="!node.isDir"
        class="opacity-0 group-hover:opacity-100 flex items-center gap-1 ml-auto shrink-0 pr-1 transition-opacity">
        <span @click.stop="$emit('duplicate', node.path)" class="cursor-pointer text-blue-500 hover:text-blue-500/80"
          title="复制">
          <CopyIcon class="h-3 w-3" />
        </span>
        <span @click.stop="$emit('delete', node.path)" class="cursor-pointer text-destructive hover:text-destructive/80"
          title="删除">
          <Trash2 class="h-3 w-3" />
        </span>
      </div>
      <div v-else
        class="opacity-0 group-hover:opacity-100 flex items-center gap-1 ml-auto shrink-0 pr-1 transition-opacity">
        <span @click.stop="$emit('delete', node.path)" class="cursor-pointer text-destructive hover:text-destructive/80"
          title="删除">
          <Trash2 class="h-3 w-3" />
        </span>
      </div>
    </div>
    <template v-if="node.isDir && isExpanded && node.children">
      <FileTreeNode v-for="child in node.children" :key="child.path" :node="child" :expanded-dirs="expandedDirs"
        :selected-path="selectedPath" :depth="depth + 1" @select="$emit('select', $event)"
        @create="$emit('create', $event)" @move="(oldPath, newPath) => $emit('move', oldPath, newPath)"
        @rename="$emit('rename', $event)" @delete="$emit('delete', $event)"
        @download-file="$emit('downloadFile', $event)" @duplicate="$emit('duplicate', $event)" />
    </template>
  </div>
</template>
