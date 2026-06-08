<script setup lang="ts">
import { ref, watch } from 'vue'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle, DialogFooter } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import { Shield } from 'lucide-vue-next'
import { api, type EnvVar } from '@/api'
import { toast } from 'vue-sonner'
import { ENV_TYPE } from '@/constants'
import LineNumberTextarea from '@/components/LineNumberTextarea.vue'
import EnvTagsConfig from './EnvTagsConfig.vue'

const emit = defineEmits<{
  (e: 'saved'): void
}>()

const isOpen = ref(false)
const isEdit = ref(false)
const editingEnv = ref<Partial<EnvVar>>({})

const textareaRef = ref<InstanceType<typeof LineNumberTextarea> | null>(null)

function openCreate(type: string) {
  editingEnv.value = { name: '', value: '', remark: '', type, hidden: true, enabled: true, tags: '' }
  isEdit.value = false
  isOpen.value = true
}

function openEdit(env: EnvVar) {
  editingEnv.value = { ...env, value: env.type === 'secret' ? '' : env.value }
  isEdit.value = true
  isOpen.value = true
}

watch(isOpen, (val) => {
  if (val) {
    // 延迟等待 DOM 渲染后更新行号
    setTimeout(() => {
      textareaRef.value?.updateVisualLineNumbers()
    }, 50)
  }
})

async function saveEnv() {
  try {
    if (isEdit.value && editingEnv.value.id) {
      await api.env.update(editingEnv.value.id, editingEnv.value)
      toast.success(editingEnv.value.type === ENV_TYPE.SECRET ? '机密已更新' : '变量已更新')
    } else {
      await api.env.create(editingEnv.value)
      toast.success(editingEnv.value.type === ENV_TYPE.SECRET ? '机密已创建' : '变量已创建')
    }
    isOpen.value = false
    emit('saved')
  } catch { toast.error('保存失败') }
}

defineExpose({
  openCreate,
  openEdit
})
</script>

<template>
  <Dialog v-model:open="isOpen">
    <DialogContent class="w-[calc(100vw-2rem)] max-w-md min-w-0">
      <DialogHeader>
        <DialogTitle>{{ isEdit ? (editingEnv.type === ENV_TYPE.SECRET ? '更新机密' : '编辑变量') : (editingEnv.type === ENV_TYPE.SECRET ? '新建机密' : '新建变量') }}</DialogTitle>
        <div v-if="editingEnv.type === ENV_TYPE.SECRET" class="flex items-center gap-2.5 p-3 mt-3 rounded-xl bg-amber-500/5 border border-amber-500/10 text-amber-600 dark:text-amber-400 text-xs leading-relaxed">
          <Shield class="h-4 w-4 shrink-0 text-amber-500" />
          <p>机密会在保存后得到<b class="text-amber-700 dark:text-amber-300">强加密保护</b>，且<b class="text-amber-700 dark:text-amber-300">仅在计划任务定时执行时才会被注入环境</b>。终端命令、调试运行、测试运行均无法获取机密内容。在执行日志内，机密文本会被自动打码。</p>
        </div>
        <DialogDescription v-else class="sr-only">编辑变量的名称、值、备注以及启用和隐藏状态。</DialogDescription>
      </DialogHeader>
      <div class="space-y-4 py-2 min-w-0">
        <div class="space-y-2 min-w-0">
          <Label class="text-sm">{{ editingEnv.type === ENV_TYPE.SECRET ? '机密名称' : '变量名' }}</Label>
          <Input v-model="editingEnv.name" class="h-9 w-full min-w-0 text-sm" :placeholder="editingEnv.type === ENV_TYPE.SECRET ? '例如：GITHUB_TOKEN' : 'MY_VAR'" />
        </div>
        <div class="space-y-2 min-w-0">
          <Label>
            {{ editingEnv.type === ENV_TYPE.SECRET ? '机密内容' : '变量值' }}
            <span v-if="editingEnv.type === ENV_TYPE.SECRET && isEdit" class="text-muted-foreground ml-1 font-normal text-xs">(输入新值即可覆盖)</span>
          </Label>
          
          <LineNumberTextarea 
            ref="textareaRef"
            v-model="editingEnv.value"
            :placeholder="editingEnv.type === ENV_TYPE.SECRET && isEdit ? '' : (editingEnv.type === ENV_TYPE.SECRET ? '输入机密内容...' : '输入变量内容...')"
            :rows="5"
            minHeightClass="min-h-16"
            maxHeightClass="max-h-40"
          />

        </div>
        <EnvTagsConfig v-model="editingEnv.tags" />
        <div class="space-y-2 min-w-0">
          <Label>备注</Label>
          <Textarea v-model="editingEnv.remark" class="w-full min-w-0 resize-none break-all text-sm" rows="3" :placeholder="editingEnv.type === ENV_TYPE.SECRET ? '机密用途说明...' : '变量用途说明...'" />
        </div>
        <div class="flex items-center justify-between space-x-2 pt-2" v-if="editingEnv.type !== ENV_TYPE.SECRET">
          <Label class="text-sm font-medium">隐藏变量值</Label>
          <Switch v-model="editingEnv.hidden" />
        </div>
        <div class="flex items-center justify-between space-x-2 pt-2">
          <Label class="text-sm font-medium">{{ editingEnv.type === ENV_TYPE.SECRET ? '启用机密' : '启用变量' }}</Label>
          <Switch v-model="editingEnv.enabled" />
        </div>
      </div>
      <DialogFooter>
        <Button variant="outline" @click="isOpen = false">取消</Button>
        <Button @click="saveEnv">保存</Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>
