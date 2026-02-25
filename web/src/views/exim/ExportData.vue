<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { toast } from 'vue-sonner'
import { api, type Task, type EnvVar, type Script, type Dependency } from '@/api'
import { Upload, Box, Settings, CheckSquare, Package } from 'lucide-vue-next'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import ExportFileTreeNode from '@/components/ExportFileTreeNode.vue'

const loading = ref(false)

const tasks = ref<Task[]>([])
const envs = ref<EnvVar[]>([])
const scripts = ref<Script[]>([])
const deps = ref<Dependency[]>([])
const fileTree = ref<any[]>([])

const expandedDirs = ref<Set<string>>(new Set())

const selectedTasks = ref<number[]>([])
const selectedEnvs = ref<number[]>([])
const selectedDeps = ref<number[]>([])
const selectedScripts = ref<string[]>([])

const exportPassword = ref('')

async function fetchData() {
  loading.value = true
  try {
    const ts = await api.tasks.list({ page_size: 1000 })
    tasks.value = ts.data || []
    
    const ev = await api.env.all()
    envs.value = ev || []

    const dp = await api.deps.list()
    deps.value = dp || []
    
    const tree = await api.files.tree()
    const flatFiles: { id: string; name: string }[] = []
    const flatten = (nodes: any[]) => {
      for (const node of nodes) {
        if (!node.isDir) {
           flatFiles.push({ id: node.path, name: node.path })
        } else if (node.children) {
           flatten(node.children)
        }
      }
    }
    if (tree) {
      flatten(tree)
      fileTree.value = tree
      expandedDirs.value = new Set()
    }
    scripts.value = flatFiles as any
  } catch (error) {
    toast.error(error instanceof Error ? error.message : '获取数据失败')
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  fetchData()
})

const handleExport = async () => {
  if (!selectedTasks.value.length && !selectedEnvs.value.length && !selectedScripts.value.length && !selectedDeps.value.length) {
    toast.error('请至少选择一项需要导出的内容')
    return
  }
  loading.value = true
  try {
    const res = await api.exim.exportData({
      task_ids: selectedTasks.value,
      env_ids: selectedEnvs.value,
      dep_ids: selectedDeps.value,
      script_paths: selectedScripts.value,
      password: exportPassword.value || undefined
    })
    if (res.file_path) {
      toast.success('分享链接已生成。为确保安全，该压缩包将在 5 分钟后自动删除。')
      const url = api.exim.downloadUrl()
      window.location.href = url
    }
  } catch (error) {
    toast.error(error instanceof Error ? error.message : '导出失败')
  } finally {
    loading.value = false
  }
}

const toggleSelectionStr = (source: string[], id: string) => {
  const index = source.indexOf(id)
  if (index === -1) {
    source.push(id)
  } else {
    source.splice(index, 1)
  }
}
const toggleSelection = (source: number[], id: number) => {
  const index = source.indexOf(id)
  if (index === -1) {
    source.push(id)
  } else {
    source.splice(index, 1)
  }
}

const toggleExpandNode = (path: string) => {
  if (expandedDirs.value.has(path)) {
    expandedDirs.value.delete(path)
  } else {
    expandedDirs.value.add(path)
  }
  expandedDirs.value = new Set(expandedDirs.value)
}

const handleSelectMultipleStr = (paths: string[], select: boolean) => {
  if (select) {
    for (const p of paths) {
      if (!selectedScripts.value.includes(p)) {
        selectedScripts.value.push(p)
      }
    }
  } else {
    selectedScripts.value = selectedScripts.value.filter(p => !paths.includes(p))
  }
}

defineExpose({
  fetchData
})
</script>

<template>
    <div class="space-y-6">
      <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        
        <!-- Tasks List -->
        <div class="rounded-lg border bg-card text-card-foreground shadow-sm flex flex-col h-[500px]">
          <div class="p-4 border-b font-semibold flex items-center justify-between">
            <div class="flex items-center gap-2"><CheckSquare class="w-4 h-4 text-blue-500"/> 定时任务 ({{ selectedTasks.length }} / {{ tasks.length }})</div>
            <Button variant="ghost" size="sm" @click="selectedTasks = selectedTasks.length === tasks.length ? [] : tasks.map(t => t.id)">全选</Button>
          </div>
          <div class="p-2 overflow-y-auto flex-1 space-y-1">
            <label v-for="t in tasks" :key="t.id" class="flex items-center justify-between hover:bg-muted/50 p-2 rounded cursor-pointer text-sm transition-colors">
               <span class="truncate pr-4 flex-1">{{ t.name }}</span>
               <input type="checkbox" :checked="selectedTasks.includes(t.id)" @change="toggleSelection(selectedTasks, t.id)" class="rounded border-gray-300 ml-2" />
            </label>
          </div>
        </div>
        
        <!-- Envs List -->
        <div class="rounded-lg border bg-card text-card-foreground shadow-sm flex flex-col h-[500px]">
          <div class="p-4 border-b font-semibold flex items-center justify-between">
            <div class="flex items-center gap-2"><Settings class="w-4 h-4 text-emerald-500"/> 环境变量 ({{ selectedEnvs.length }} / {{ envs.length }})</div>
            <Button variant="ghost" size="sm" @click="selectedEnvs = selectedEnvs.length === envs.length ? [] : envs.map(e => e.id)">全选</Button>
          </div>
          <div class="p-2 overflow-y-auto flex-1 space-y-1">
            <label v-for="e in envs" :key="e.id" class="flex items-center justify-between hover:bg-muted/50 p-2 rounded cursor-pointer text-sm transition-colors">
               <span class="truncate pr-4 flex-1">{{ e.name }}</span>
               <input type="checkbox" :checked="selectedEnvs.includes(e.id)" @change="toggleSelection(selectedEnvs, e.id)" class="rounded border-gray-300 ml-2" />
            </label>
          </div>
        </div>

        <!-- Deps List -->
        <div class="rounded-lg border bg-card text-card-foreground shadow-sm flex flex-col h-[500px]">
          <div class="p-4 border-b font-semibold flex items-center justify-between">
            <div class="flex items-center gap-2"><Package class="w-4 h-4 text-orange-500"/> 语言依赖 ({{ selectedDeps.length }} / {{ deps.length }})</div>
            <Button variant="ghost" size="sm" @click="selectedDeps = selectedDeps.length === deps.length ? [] : deps.map(d => d.id)">全选</Button>
          </div>
          <div class="p-2 overflow-y-auto flex-1 space-y-1">
            <label v-for="d in deps" :key="d.id" class="flex items-center justify-between hover:bg-muted/50 p-2 rounded cursor-pointer text-sm transition-colors">
               <span class="truncate pr-4 flex-1">{{ d.name }}<span class="text-xs text-muted-foreground ml-1">@{{ d.language }}</span></span>
               <input type="checkbox" :checked="selectedDeps.includes(d.id)" @change="toggleSelection(selectedDeps, d.id)" class="rounded border-gray-300 ml-2" />
            </label>
          </div>
        </div>

        <!-- Scripts List -->
        <div class="rounded-lg border bg-card text-card-foreground shadow-sm flex flex-col h-[500px]">
          <div class="p-4 border-b font-semibold flex items-center justify-between">
            <div class="flex items-center gap-2"><Box class="w-4 h-4 text-purple-500"/> 脚本文件 ({{ selectedScripts.length }} / {{ scripts.length }})</div>
            <Button variant="ghost" size="sm" @click="selectedScripts = selectedScripts.length === scripts.length ? [] : scripts.map(s => String(s.id))">全选</Button>
          </div>
          <div class="p-2 overflow-y-auto flex-1 space-y-1">
            <ExportFileTreeNode 
              v-for="node in fileTree" 
              :key="node.path" 
              :node="node" 
              :expanded-dirs="expandedDirs"
              :selected-scripts="selectedScripts" 
              @toggleExpand="toggleExpandNode"
              @toggleSelect="toggleSelectionStr(selectedScripts, $event)"
              @selectMultiple="handleSelectMultipleStr"
            />
          </div>
        </div>

      </div>
      
      <div class="flex items-center justify-between gap-4 bg-muted/20 p-6 rounded-lg border shadow-sm backdrop-blur-sm">
         <div class="flex-1 max-w-sm">
            <label class="block text-sm font-medium mb-2 text-foreground/80">压缩包密码 (选填)</label>
            <Input v-model="exportPassword" type="password" placeholder="留空则不加密" class="max-w-xs transition-shadow focus-visible:ring-primary/50" />
         </div>
         <Button @click="handleExport" :disabled="loading" size="lg" class="mt-4 shadow-md hover:shadow-lg transition-all h-11 px-8 rounded-full">
             <Upload class="w-4 h-4 mr-2" />
             {{ loading ? '导出中...' : '生成并下载包' }}
         </Button>
      </div>

    </div>
</template>
