<script setup lang="ts">
import { ref, computed } from 'vue'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter } from '@/components/ui/dialog'
import { Label } from '@/components/ui/label'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import DateTimePicker from '@/components/ui/DateTimePicker.vue'
import * as nodeApi from '@/api/node'
import { toast } from 'vue-sonner'

const emit = defineEmits<{
  (e: 'updated'): void
}>()

const isOpen = ref(false)
const isEdit = ref(false)
const editingToken = ref<nodeApi.NodeToken | null>(null)
const formData = ref({ remark: '', max_uses: 0, expires_at: '' })

const title = computed(() => isEdit.value ? '编辑令牌' : '生成令牌')
const description = computed(() => isEdit.value ? '修改令牌的备注、使用次数和过期时间' : '创建一个新的注册令牌，用于 Agent 认证')

// 仅提供创建，因为后端未实现 Token 更新接口，但保留存根以防编译错误
function openCreate() {
  isEdit.value = false
  editingToken.value = null
  formData.value = { remark: '', max_uses: 0, expires_at: '' }
  isOpen.value = true
}

function openEdit(token: nodeApi.NodeToken) {
  isEdit.value = true
  editingToken.value = token
  const rawExpires = token.expires_at?.replace(' ', 'T').slice(0, 16) || ''
  formData.value = { remark: token.remark || '', max_uses: token.max_uses, expires_at: rawExpires }
  isOpen.value = true
}

async function handleSave() {
  try {
    let expiresAt = formData.value.expires_at
    if (expiresAt) {
      expiresAt = expiresAt.replace('T', ' ') + ':00'
    }

    if (isEdit.value && editingToken.value) {
      // 后端不支持更新，弹友好提示
      toast.error('当前版本不支持修改令牌')
    } else {
      await nodeApi.createToken({
        remark: formData.value.remark,
        max_uses: formData.value.max_uses,
        expires_at: expiresAt || undefined
      })
      toast.success('创建成功')
    }
    
    isOpen.value = false
    emit('updated')
  } catch (e: unknown) {
    toast.error((e as Error).message || (isEdit.value ? '更新失败' : '创建失败'))
  }
}

defineExpose({ openCreate, openEdit })
</script>

<template>
  <Dialog v-model:open="isOpen">
    <DialogContent>
      <DialogHeader>
        <DialogTitle>{{ title }}</DialogTitle>
        <DialogDescription class="sr-only">{{ description }}</DialogDescription>
      </DialogHeader>
      <div class="grid gap-4 py-2">
        <div class="grid gap-1.5">
          <Label for="remark" class="text-xs font-medium text-muted-foreground">备注</Label>
          <Input id="remark" v-model="formData.remark" placeholder="备注信息（可选，例如部署环境名称）" class="h-9 text-xs" />
        </div>
        <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
          <div class="grid gap-1.5">
            <Label for="max_uses" class="text-xs font-medium text-muted-foreground">最大使用次数</Label>
            <Input id="max_uses" v-model.number="formData.max_uses" type="number" min="0" placeholder="0" class="h-9 text-xs" />
            <span class="text-[10px] text-muted-foreground/80">0 表示此令牌无限制重复使用</span>
          </div>
          <div class="grid gap-1.5">
            <Label for="expires_at" class="text-xs font-medium text-muted-foreground">过期时间</Label>
            <DateTimePicker id="expires_at" v-model="formData.expires_at" placeholder="永久有效" />
            <span class="text-[10px] text-muted-foreground/80">不填写表示令牌永久有效</span>
          </div>
        </div>
      </div>
      <DialogFooter>
        <Button variant="outline" @click="isOpen = false">取消</Button>
        <Button @click="handleSave">{{ isEdit ? '保存' : '生成' }}</Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>
