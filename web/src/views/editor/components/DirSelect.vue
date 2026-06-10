<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { Button } from '@/components/ui/button'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'
import { Folder, ChevronsUpDown, ChevronDown, ChevronRight } from 'lucide-vue-next'
import type { FileNode } from '@/api'

const props = defineProps<{
  fileTree?: FileNode[]
  modelValue: string
  defaultExpand?: string
}>()

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()

const isPopoverOpen = ref(false)
const expandedDirs = ref<Set<string>>(new Set())

watch(() => props.defaultExpand, (newVal) => {
  if (newVal) {
    const parts = newVal.split('/')
    let current = ''
    for (const part of parts) {
      current = current ? `${current}/${part}` : part
      expandedDirs.value.add(current)
    }
  }
}, { immediate: true })

function toggleExpand(path: string) {
  if (expandedDirs.value.has(path)) {
    expandedDirs.value.delete(path)
  } else {
    expandedDirs.value.add(path)
  }
}

const visibleDirs = computed(() => {
  const result: { path: string; name: string; depth: number; hasChildren: boolean }[] = []
  
  function traverse(nodes: FileNode[], depth: number) {
    for (const node of nodes) {
      if (node.isDir) {
        const hasChildren = !!node.children && node.children.some(c => c.isDir)
        result.push({ path: node.path, name: node.name, depth, hasChildren })
        if (expandedDirs.value.has(node.path) && node.children) {
          traverse(node.children, depth + 1)
        }
      }
    }
  }
  
  if (props.fileTree) {
    traverse(props.fileTree, 0)
  }
  return result
})

function selectDir(path: string) {
  emit('update:modelValue', path)
  isPopoverOpen.value = false
}
</script>

<template>
  <Popover v-model:open="isPopoverOpen">
    <PopoverTrigger as-child>
      <Button variant="outline" role="combobox" class="w-full justify-between h-8 text-xs font-normal">
        {{ modelValue || '根目录' }}
        <ChevronsUpDown class="ml-2 h-4 w-4 shrink-0 opacity-50" />
      </Button>
    </PopoverTrigger>
    <PopoverContent class="w-[280px] p-0" align="start">
      <div class="max-h-[300px] overflow-auto py-1">
        <div 
          class="flex items-center gap-1 py-1.5 px-2 hover:bg-muted cursor-pointer text-xs" 
          :class="{ 'bg-accent text-accent-foreground': modelValue === '/' || modelValue === '' }"
          @click="selectDir('/')"
        >
          <Folder class="h-3 w-3 text-yellow-500" />
          根目录
        </div>
        <div 
          v-for="item in visibleDirs" 
          :key="item.path"
          class="flex items-center gap-1 py-1.5 px-2 hover:bg-muted cursor-pointer text-xs group"
          :class="{ 'bg-accent text-accent-foreground': modelValue === item.path }"
          :style="{ paddingLeft: (item.depth * 12 + 8) + 'px' }"
          @click="selectDir(item.path)"
        >
          <div @click.stop="toggleExpand(item.path)" class="w-4 h-4 flex items-center justify-center -ml-1 rounded hover:bg-muted-foreground/20">
            <ChevronDown v-if="expandedDirs.has(item.path)" class="h-3 w-3 shrink-0" />
            <ChevronRight v-else-if="item.hasChildren" class="h-3 w-3 shrink-0" />
            <span v-else class="w-3 h-3"></span>
          </div>
          <Folder class="h-3 w-3 text-yellow-500 shrink-0" />
          <span class="truncate">{{ item.name }}</span>
        </div>
      </div>
    </PopoverContent>
  </Popover>
</template>
