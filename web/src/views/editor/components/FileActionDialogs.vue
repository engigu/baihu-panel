<script setup lang="ts">
import { ref } from 'vue'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from '@/components/ui/dialog'
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle } from '@/components/ui/alert-dialog'
import { Label } from '@/components/ui/label'
import { RadioGroup, RadioGroupItem } from '@/components/ui/radio-group'

// Create Dialog State
const showCreate = ref(false)
const newItemName = ref('')
const newItemType = ref<'file' | 'dir'>('file')
const createInDir = ref('')

// Delete Dialog State
const showDelete = ref(false)
const deletePath = ref('')

// Rename Dialog State
const showRename = ref(false)
const renamePath = ref('')
const newName = ref('')

const emit = defineEmits<{
  create: [name: string, type: 'file' | 'dir', parent: string]
  delete: [path: string]
  rename: [oldPath: string, newName: string]
}>()

function openCreate(parent = '') {
  newItemName.value = ''
  newItemType.value = 'file'
  createInDir.value = parent
  showCreate.value = true
}

function openDelete(path: string) {
  deletePath.value = path
  showDelete.value = true
}

function openRename(path: string) {
  renamePath.value = path
  newName.value = path.split('/').pop() || ''
  showRename.value = true
}

defineExpose({ openCreate, openDelete, openRename, closeCreate: () => showCreate.value = false, closeRename: () => showRename.value = false, closeDelete: () => showDelete.value = false })
</script>

<template>
  <!-- 新建对话框 -->
  <Dialog v-model:open="showCreate">
    <DialogContent class="max-w-xs" @openAutoFocus.prevent>
      <DialogHeader>
        <DialogTitle class="text-sm">新建</DialogTitle>
      </DialogHeader>
      <div class="space-y-3 py-2">
        <div class="text-xs text-muted-foreground">
          位置: {{ createInDir || '根目录' }}
        </div>
        <RadioGroup v-model="newItemType" class="flex gap-4">
          <div class="flex items-center gap-2">
            <RadioGroupItem value="file" id="file" />
            <Label for="file" class="text-xs">文件</Label>
          </div>
          <div class="flex items-center gap-2">
            <RadioGroupItem value="dir" id="dir" />
            <Label for="dir" class="text-xs">文件夹</Label>
          </div>
        </RadioGroup>
        <div class="space-y-1">
          <Label class="text-xs">名称</Label>
          <Input v-model="newItemName" class="h-8 text-xs" placeholder="script.sh" @keyup.enter="emit('create', newItemName, newItemType, createInDir)" />
        </div>
      </div>
      <DialogFooter>
        <Button variant="outline" size="sm" class="h-7 text-xs" @click="showCreate = false">取消</Button>
        <Button size="sm" class="h-7 text-xs" @click="emit('create', newItemName, newItemType, createInDir)">创建</Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>

  <!-- 删除确认 -->
  <AlertDialog v-model:open="showDelete">
    <AlertDialogContent>
      <AlertDialogHeader>
        <AlertDialogTitle class="text-sm">确认删除</AlertDialogTitle>
        <AlertDialogDescription class="text-xs">确定要删除 {{ deletePath }} 吗？</AlertDialogDescription>
      </AlertDialogHeader>
      <AlertDialogFooter>
        <AlertDialogCancel class="h-7 text-xs" @click="showDelete = false">取消</AlertDialogCancel>
        <AlertDialogAction class="h-7 text-xs bg-destructive text-white hover:bg-destructive/90" @click="emit('delete', deletePath)">
          删除
        </AlertDialogAction>
      </AlertDialogFooter>
    </AlertDialogContent>
  </AlertDialog>

  <!-- 重命名对话框 -->
  <Dialog v-model:open="showRename">
    <DialogContent class="max-w-xs" @openAutoFocus.prevent>
      <DialogHeader>
        <DialogTitle class="text-sm">重命名</DialogTitle>
      </DialogHeader>
      <div class="space-y-3 py-2">
        <div class="space-y-1">
          <Label class="text-xs">新名称</Label>
          <Input v-model="newName" class="h-8 text-xs" placeholder="new_name.sh" @keyup.enter="emit('rename', renamePath, newName)" />
          <p class="text-[10px] text-muted-foreground">注：仅支持修改名称，不可包含路径分隔符 /</p>
        </div>
      </div>
      <DialogFooter>
        <Button variant="outline" size="sm" class="h-7 text-xs" @click="showRename = false">取消</Button>
        <Button size="sm" class="h-7 text-xs" @click="emit('rename', renamePath, newName)">确定</Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>
