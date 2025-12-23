<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { Folder, ChevronRight, ChevronDown, FolderOpen } from 'lucide-vue-next'
import { Button } from '@/components/ui/button'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'
import { api, type FileNode } from '@/api'

const props = defineProps<{
  modelValue?: string
  placeholder?: string
}>()

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()

const open = ref(false)
const loading = ref(false)
const fileTree = ref<FileNode[]>([])
const expandedDirs = ref<Set<string>>(new Set())

interface FlatDir {
  path: string
  name: string
  depth: number
  hasChildren: boolean
}

// 将树形结构扁平化，只保留目录
function flattenDirs(nodes: FileNode[], depth = 0): FlatDir[] {
  const result: FlatDir[] = []
  for (const node of nodes) {
    if (!node.isDir) continue
    const children = node.children?.filter(c => c.isDir) || []
    result.push({
      path: node.path,
      name: node.name,
      depth,
      hasChildren: children.length > 0
    })
    if (expandedDirs.value.has(node.path) && node.children) {
      result.push(...flattenDirs(node.children, depth + 1))
    }
  }
  return result
}

const flatDirs = computed(() => flattenDirs(fileTree.value))

async function loadTree() {
  if (fileTree.value.length > 0) return
  loading.value = true
  try {
    fileTree.value = await api.files.tree()
  } catch {
    fileTree.value = []
  } finally {
    loading.value = false
  }
}

function toggleDir(path: string, e: Event) {
  e.stopPropagation()
  if (expandedDirs.value.has(path)) {
    expandedDirs.value.delete(path)
  } else {
    expandedDirs.value.add(path)
  }
}

function selectDir(path: string) {
  emit('update:modelValue', path)
  open.value = false
}

function selectRoot() {
  emit('update:modelValue', '')
  open.value = false
}

watch(open, (val) => {
  if (val) loadTree()
})

const displayValue = computed(() => {
  if (!props.modelValue) return props.placeholder || 'scripts (默认)'
  return props.modelValue
})
</script>

<template>
  <Popover v-model:open="open">
    <PopoverTrigger as-child>
      <Button variant="outline" class="w-full justify-start font-mono text-sm h-9">
        <Folder class="h-4 w-4 mr-2 text-yellow-500 shrink-0" />
        <span class="truncate">{{ displayValue }}</span>
      </Button>
    </PopoverTrigger>
    <PopoverContent class="w-[300px] p-2" align="start">
      <div class="text-xs text-muted-foreground mb-2">选择工作目录</div>
      <div class="max-h-[240px] overflow-y-auto">
        <!-- 根目录选项 -->
        <div
          :class="[
            'flex items-center gap-1.5 py-1 px-2 rounded cursor-pointer text-sm',
            !modelValue ? 'bg-primary/10 text-primary' : 'hover:bg-muted'
          ]"
          @click="selectRoot"
        >
          <FolderOpen class="h-4 w-4 text-yellow-500" />
          <span>scripts (默认)</span>
        </div>
        
        <!-- 扁平化的目录列表 -->
        <div
          v-for="dir in flatDirs"
          :key="dir.path"
          :class="[
            'flex items-center gap-1 py-1 px-2 rounded cursor-pointer text-sm',
            modelValue === dir.path ? 'bg-primary/10 text-primary' : 'hover:bg-muted'
          ]"
          :style="{ paddingLeft: (dir.depth * 12 + 8) + 'px' }"
          @click="selectDir(dir.path)"
        >
          <span
            v-if="dir.hasChildren"
            class="shrink-0 cursor-pointer"
            @click="toggleDir(dir.path, $event)"
          >
            <ChevronDown v-if="expandedDirs.has(dir.path)" class="h-3 w-3" />
            <ChevronRight v-else class="h-3 w-3" />
          </span>
          <span v-else class="w-3 shrink-0" />
          <Folder class="h-4 w-4 text-yellow-500 shrink-0" />
          <span class="truncate">{{ dir.name }}</span>
        </div>
        
        <div v-if="loading" class="text-xs text-muted-foreground text-center py-4">
          加载中...
        </div>
        <div v-else-if="flatDirs.length === 0" class="text-xs text-muted-foreground text-center py-2">
          暂无子目录
        </div>
      </div>
    </PopoverContent>
  </Popover>
</template>
