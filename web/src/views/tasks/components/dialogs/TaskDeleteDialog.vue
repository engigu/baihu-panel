<script setup lang="ts">
import { ref } from 'vue'
import BaihuDialog from '@/components/ui/BaihuDialog.vue'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import { Label } from '@/components/ui/label'
import { AlertTriangle } from 'lucide-vue-next'
import type { Task } from '@/api'
import { TASK_TYPE } from '@/constants'

const props = defineProps<{
  openDelete: boolean
  openBatchDelete: boolean
  targetTask?: Task | null
}>()

const emit = defineEmits<{
  'update:openDelete': [value: boolean]
  'update:openBatchDelete': [value: boolean]
  'confirmDelete': [deleteFiles: boolean]
  'confirmBatchDelete': []
}>()

const deleteFiles = ref(false)

function handleDeleteSingle() {
  emit('confirmDelete', deleteFiles.value)
  deleteFiles.value = false
}
</script>

<template>
  <!-- 单个任务/应用删除 -->
  <BaihuDialog :open="openDelete" :title="targetTask?.type === TASK_TYPE.APP ? '确认卸载应用' : '确认删除任务'" @update:open="$emit('update:openDelete', $event)">
    <div class="space-y-3 py-1">
      <div class="text-xs text-muted-foreground leading-relaxed">
        确定要{{ targetTask?.type === TASK_TYPE.APP ? '卸载应用' : '删除任务' }} <b class="text-foreground font-semibold">{{ targetTask?.name }}</b> 吗？
      </div>

      <div v-if="targetTask?.type === TASK_TYPE.APP" class="p-3 rounded-lg bg-destructive/10 border border-destructive/20 flex items-start gap-2 text-xs text-destructive">
        <AlertTriangle class="w-4 h-4 shrink-0 mt-0.5" />
        <div class="leading-relaxed">
          卸载应用将彻底删除<b>本地代码文件夹</b>以及<b>所有关联的受控任务与配置</b>，该操作无法撤销。
        </div>
      </div>
      <div v-else class="text-xs text-destructive font-medium">
        ⚠️ 此操作无法撤销。
      </div>
    </div>
    <template #footer>
      <div class="flex items-center justify-between w-full gap-4">
        <div v-if="targetTask?.type === TASK_TYPE.REPO" class="flex items-center gap-2 mr-auto">
          <Checkbox id="delete-files" v-model="deleteFiles" />
          <Label for="delete-files" class="text-sm font-medium text-destructive cursor-pointer select-none">
            同时物理删除仓库文件夹
          </Label>
        </div>
        <div class="flex justify-end gap-2 ml-auto">
          <Button variant="ghost" size="sm" @click="$emit('update:openDelete', false)">取消</Button>
          <Button variant="destructive" size="sm" @click="handleDeleteSingle">
            {{ targetTask?.type === TASK_TYPE.APP ? '确认彻底卸载' : '确定删除' }}
          </Button>
        </div>
      </div>
    </template>
  </BaihuDialog>

  <!-- 批量删除确认 -->
  <BaihuDialog :open="openBatchDelete" title="确认批量删除" @update:open="$emit('update:openBatchDelete', $event)">
    <div class="text-sm text-muted-foreground leading-relaxed py-2">
      确定要批量删除当前筛选出来的所有任务吗？
      <p class="mt-2 text-destructive font-medium">⚠️ 操作不可撤销，请谨慎操作。</p>
    </div>
    <template #footer>
      <Button variant="ghost" @click="$emit('update:openBatchDelete', false)">取消</Button>
      <Button variant="destructive" class="shadow-lg shadow-destructive/20" @click="$emit('confirmBatchDelete')">确认批量删除</Button>
    </template>
  </BaihuDialog>
</template>
