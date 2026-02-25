<script setup lang="ts">
import { computed } from 'vue'
import { Folder, File, ChevronRight, ChevronDown } from 'lucide-vue-next'
import type { FileNode } from '@/api'

defineOptions({ name: 'ExportFileTreeNode' })

const props = defineProps<{
  node: FileNode
  expandedDirs: Set<string>
  selectedScripts: string[]
  depth?: number
}>()

const emit = defineEmits<{
  toggleExpand: [path: string]
  toggleSelect: [path: string]
  selectMultiple: [paths: string[], select: boolean]
}>()

const depth = computed(() => props.depth ?? 0)
const isExpanded = computed(() => props.expandedDirs.has(props.node.path))

const allFilePaths = computed(() => {
  if (!props.node.isDir) return [props.node.path]
  const files: string[] = []
  const dfs = (node: FileNode) => {
    if (!node.isDir) files.push(node.path)
    else if (node.children) node.children.forEach(dfs)
  }
  dfs(props.node)
  return files
})

const isSelected = computed(() => {
  if (!props.node.isDir) return props.selectedScripts.includes(props.node.path)
  const files = allFilePaths.value
  return files.length > 0 && files.every(f => props.selectedScripts.includes(f))
})

const isIndeterminate = computed(() => {
  if (!props.node.isDir) return false
  const files = allFilePaths.value
  if (files.length === 0) return false
  const selectedCount = files.filter(f => props.selectedScripts.includes(f)).length
  return selectedCount > 0 && selectedCount < files.length
})

function handleExpand() {
  if (props.node.isDir) {
    emit('toggleExpand', props.node.path)
  } else {
    emit('toggleSelect', props.node.path)
  }
}

function handleSelect() {
  if (!props.node.isDir) {
    emit('toggleSelect', props.node.path)
  } else {
    emit('selectMultiple', allFilePaths.value, !isSelected.value)
  }
}
</script>

<template>
  <div>
    <div :class="[
      'flex items-center justify-between py-1 px-2 rounded cursor-pointer text-sm hover:bg-muted/50 transition-colors',
      isSelected && 'bg-accent/50'
    ]">
      <div 
        class="flex items-center gap-2 flex-1 min-w-0" 
        :style="{ paddingLeft: depth * 16 + 'px' }"
        @click="handleExpand"
      >
        <template v-if="node.isDir">
          <ChevronDown v-if="isExpanded" class="h-4 w-4 text-muted-foreground flex-shrink-0" />
          <ChevronRight v-else class="h-4 w-4 text-muted-foreground flex-shrink-0" />
        </template>
        <span v-else class="w-4 flex-shrink-0" />
        <Folder v-if="node.isDir" class="h-4 w-4 text-yellow-500 flex-shrink-0" />
        <File v-else class="h-4 w-4 text-blue-500 flex-shrink-0" />
        <span class="truncate flex-1" :class="{'font-medium': node.isDir}">{{ node.name }}</span>
      </div>
      
      <input 
        type="checkbox" 
        :checked="isSelected" 
        :indeterminate.prop="isIndeterminate"
        @change="handleSelect" 
        class="rounded border-gray-300 ml-2 shadow-sm"
      />
    </div>
    <template v-if="node.isDir && isExpanded && node.children">
      <ExportFileTreeNode 
        v-for="child in node.children" 
        :key="child.path" 
        :node="child" 
        :expanded-dirs="expandedDirs"
        :selected-scripts="selectedScripts" 
        :depth="depth + 1" 
        @toggleExpand="$emit('toggleExpand', $event)"
        @toggleSelect="$emit('toggleSelect', $event)"
        @selectMultiple="(paths, select) => $emit('selectMultiple', paths, select)"
      />
    </template>
  </div>
</template>
