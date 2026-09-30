<script setup lang="ts">
import { ref, watch } from 'vue'
import BaihuDialog from '@/components/ui/BaihuDialog.vue'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import { Label } from '@/components/ui/label'
import { AlertTriangle, Loader2 } from 'lucide-vue-next'
import type { Task } from '@/api'
import { TASK_TYPE } from '@/constants'

const props = defineProps<{
  openDelete: boolean
  openBatchDelete: boolean
  targetTask?: Task | null
  loading?: boolean
}>()

const emit = defineEmits<{
  'update:openDelete': [value: boolean]
  'update:openBatchDelete': [value: boolean]
  'confirmDelete': [deleteFiles: boolean, cleanEnvs: boolean]
  'confirmBatchDelete': []
}>()

const deleteFiles = ref(false)
const cleanAppData = ref(true)
const cleanAppEnvs = ref(true)

watch(() => props.openDelete, (open) => {
  if (!open) {
    cleanAppData.value = true
    cleanAppEnvs.value = true
    deleteFiles.value = false
  }
})

function handleDeleteSingle() {
  if (props.loading) return
  if (props.targetTask?.type === TASK_TYPE.APP) {
    emit('confirmDelete', cleanAppData.value, cleanAppEnvs.value)
  } else {
    emit('confirmDelete', deleteFiles.value, false)
  }
}
</script>

<template>
  <!-- 单个任务/应用删除 -->
  <BaihuDialog :open="openDelete" :title="targetTask?.type === TASK_TYPE.APP ? '确认卸载应用' : '确认删除任务'" @update:open="!loading && $emit('update:openDelete', $event)">
    <div class="space-y-3 py-1">
      <div class="text-xs text-muted-foreground leading-relaxed">
        确定要{{ targetTask?.type === TASK_TYPE.APP ? '卸载应用' : '删除任务' }} <b class="text-foreground font-semibold">{{ targetTask?.name }}</b> 吗？
      </div>

      <template v-if="targetTask?.type === TASK_TYPE.APP">
        <div class="p-3 rounded-lg bg-destructive/10 border border-destructive/20 flex items-start gap-2 text-xs text-destructive">
          <AlertTriangle class="w-4 h-4 shrink-0 mt-0.5" />
          <div class="leading-relaxed">
            卸载应用将移除<b>主应用与所有受控子任务</b>，并自动回收无其他引用的专属标签，该操作无法撤销。
          </div>
        </div>

        <div class="space-y-2.5 pt-1">
          <div class="flex items-center gap-2">
            <Checkbox id="clean-app-data" v-model:checked="cleanAppData" :disabled="loading" />
            <Label for="clean-app-data" class="text-xs font-medium text-foreground cursor-pointer select-none">
              同时删除本地代码与产物文件夹
            </Label>
          </div>
          <div class="flex items-center gap-2">
            <Checkbox id="clean-app-envs" v-model:checked="cleanAppEnvs" :disabled="loading" />
            <Label for="clean-app-envs" class="text-xs font-medium text-foreground cursor-pointer select-none">
              同时删除应用关联的环境变量与凭证
            </Label>
          </div>
        </div>
      </template>
      <div v-else class="text-xs text-destructive font-medium">
        ⚠️ 此操作无法撤销。
      </div>
    </div>
    <template #footer>
      <div class="flex items-center justify-between w-full gap-4">
        <div v-if="targetTask?.type === TASK_TYPE.REPO" class="flex items-center gap-2 mr-auto">
          <Checkbox id="delete-files" v-model:checked="deleteFiles" :disabled="loading" />
          <Label for="delete-files" class="text-sm font-medium text-destructive cursor-pointer select-none">
            同时物理删除仓库文件夹
          </Label>
        </div>
        <div class="flex justify-end gap-2 ml-auto">
          <Button variant="ghost" size="sm" :disabled="loading" @click="$emit('update:openDelete', false)">取消</Button>
          <Button variant="destructive" size="sm" :disabled="loading" @click="handleDeleteSingle">
            <Loader2 v-if="loading" class="w-3.5 h-3.5 mr-1.5 animate-spin" />
            <span>{{ loading ? (targetTask?.type === TASK_TYPE.APP ? '正在卸载...' : '正在删除...') : (targetTask?.type === TASK_TYPE.APP ? '确认彻底卸载' : '确定删除') }}</span>
          </Button>
        </div>
      </div>
    </template>
  </BaihuDialog>

  <!-- 批量删除确认 -->
  <BaihuDialog :open="openBatchDelete" title="确认批量删除" @update:open="!loading && $emit('update:openBatchDelete', $event)">
    <div class="text-sm text-muted-foreground leading-relaxed py-2">
      确定要批量删除当前筛选出来的所有任务吗？
      <p class="mt-2 text-destructive font-medium">⚠️ 操作不可撤销，请谨慎操作。</p>
    </div>
    <template #footer>
      <Button variant="ghost" :disabled="loading" @click="$emit('update:openBatchDelete', false)">取消</Button>
      <Button variant="destructive" class="shadow-lg shadow-destructive/20" :disabled="loading" @click="$emit('confirmBatchDelete')">
        <Loader2 v-if="loading" class="w-3.5 h-3.5 mr-1.5 animate-spin" />
        <span>{{ loading ? '正在删除...' : '确认批量删除' }}</span>
      </Button>
    </template>
  </BaihuDialog>
</template>
