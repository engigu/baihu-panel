<script setup lang="ts">
import { ref } from 'vue'
import { toast } from 'vue-sonner'
import { api } from '@/api'
import { Download } from 'lucide-vue-next'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'

const loading = ref(false)

const importPassword = ref('')
const importScript = ref(true)
const importEnv = ref(true)
const importTask = ref(true)
const importDep = ref(true)

const fileInput = ref<HTMLInputElement | null>(null)
const selectedFile = ref<File | null>(null)

const emit = defineEmits<{
  (e: 'imported'): void
}>()

const handleFileChange = (event: Event) => {
  const target = event.target as HTMLInputElement
  if (target.files && target.files.length > 0) {
    selectedFile.value = target.files[0] || null
  }
}

const handleImport = async () => {
  if (!selectedFile.value) {
    toast.error('请先选择要导入的压缩包')
    return
  }
  if (!importTask.value && !importEnv.value && !importScript.value && !importDep.value) {
    toast.error('请至少选择一项要导入的类别')
    return
  }
  loading.value = true
  try {
    await api.exim.importData(selectedFile.value, {
      password: importPassword.value || undefined,
      import_task: importTask.value,
      import_env: importEnv.value,
      import_script: importScript.value,
      import_dep: importDep.value
    })
    toast.success('恢复导入成功')
    selectedFile.value = null
    importPassword.value = ''
    if (fileInput.value) fileInput.value.value = ''
    emit('imported')
  } catch (error) {
    toast.error(error instanceof Error ? error.message : '导入失败')
  } finally {
    loading.value = false
  }
}
</script>

<template>
    <div class="space-y-6 flex justify-center">
       <div class="w-full max-w-2xl rounded-2xl border bg-card/60 backdrop-blur text-card-foreground shadow-lg p-8 space-y-8">
          
          <div class="space-y-3">
            <label class="block text-sm font-semibold text-foreground/90">选择备份/分享文件 (.zip)</label>
            <div class="flex items-center justify-center w-full relative">
              <label for="dropzone-file" class="flex flex-col items-center justify-center w-full h-32 border-2 border-dashed rounded-xl cursor-pointer bg-muted/30 border-muted-foreground/30 hover:bg-muted/50 hover:border-primary/50 transition-all">
                  <div class="flex flex-col items-center justify-center pt-5 pb-6">
                      <Download class="w-8 h-8 mb-3 text-muted-foreground" />
                      <p class="mb-1 text-sm font-medium text-foreground"><span class="font-bold underline decoration-primary decoration-2">点击选择</span> 压缩包文件</p>
                      <p class="text-xs text-muted-foreground truncate max-w-[200px]" v-if="selectedFile">{{ selectedFile.name }}</p>
                      <p class="text-xs text-muted-foreground" v-else>支持白虎系统的ZIP导出格式</p>
                  </div>
                  <input id="dropzone-file" type="file" ref="fileInput" accept=".zip" @change="handleFileChange" class="hidden" />
              </label>
            </div>
          </div>
          
          <div class="space-y-3">
            <label class="block text-sm font-semibold text-foreground/90">提取密码 (非必填)</label>
            <Input v-model="importPassword" type="password" placeholder="若导出时设置了密码，请在此输入密码" class="w-full transition-shadow focus-visible:ring-primary/50" />
            <p class="text-xs text-muted-foreground mt-1">留空如果当时没有设定加密</p>
          </div>

          <div class="space-y-4 pt-2 border-t border-border/50">
             <label class="block text-sm font-semibold text-foreground/90">选择要恢复的数据种类</label>
             <div class="flex items-center flex-wrap gap-8 p-4 bg-muted/20 rounded-xl border">
                <label class="flex items-center cursor-pointer group">
                  <input type="checkbox" v-model="importTask" class="rounded border-gray-300 w-4 h-4 text-primary focus:ring-primary/50 transition-all" />
                  <span class="ml-2 text-sm font-medium group-hover:text-primary transition-colors">定时任务</span>
                </label>
                <label class="flex items-center cursor-pointer group">
                  <input type="checkbox" v-model="importEnv" class="rounded border-gray-300 w-4 h-4 text-primary focus:ring-primary/50 transition-all" />
                  <span class="ml-2 text-sm font-medium group-hover:text-primary transition-colors">环境变量</span>
                </label>
                <label class="flex items-center cursor-pointer group">
                  <input type="checkbox" v-model="importScript" class="rounded border-gray-300 w-4 h-4 text-primary focus:ring-primary/50 transition-all" />
                  <span class="ml-2 text-sm font-medium group-hover:text-primary transition-colors">本地脚本</span>
                </label>
                <label class="flex items-center cursor-pointer group">
                  <input type="checkbox" v-model="importDep" class="rounded border-gray-300 w-4 h-4 text-primary focus:ring-primary/50 transition-all" />
                  <span class="ml-2 text-sm font-medium group-hover:text-primary transition-colors">语言依赖</span>
                </label>
             </div>
          </div>
          
          <Button @click="handleImport" :disabled="loading || !selectedFile" class="w-full h-12 rounded-xl text-base font-semibold shadow-md hover:shadow-lg transition-all" size="lg">
             <Download class="w-5 h-5 mr-2" />
             {{ loading ? '恢复导入中...' : '开始安全导入' }}
          </Button>
       </div>
    </div>
</template>
