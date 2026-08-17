<script setup lang="ts">
import { ref, computed } from 'vue'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter } from '@/components/ui/dialog'
import { Label } from '@/components/ui/label'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import { api, type AgentToken } from '@/api'
import { toast } from 'vue-sonner'

const emit = defineEmits<{
  (e: 'updated'): void
}>()

const isOpen = ref(false)
const isEdit = ref(false)
const editingToken = ref<AgentToken | null>(null)
const formData = ref({ remark: '', max_uses: 0, expires_at: '' })

const title = computed(() => isEdit.value ? '编辑令牌' : '生成令牌')
const description = computed(() => isEdit.value ? '修改令牌的备注、使用次数和过期时间' : '创建一个新的注册令牌，用于 Agent 认证')

function openCreate() {
  isEdit.value = false
  editingToken.value = null
  formData.value = { remark: '', max_uses: 0, expires_at: '' }
  isOpen.value = true
}

function openEdit(token: AgentToken) {
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
      await api.agents.updateToken(editingToken.value.id, {
        remark: formData.value.remark,
        max_uses: formData.value.max_uses,
        expires_at: expiresAt || undefined
      })
      toast.success('更新成功')
    } else {
      await api.agents.createToken({
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
      <div class="space-y-4">
        <div>
          <Label>备注</Label>
          <Input v-model="formData.remark" placeholder="备注信息（可选）" />
        </div>
        <div>
          <Label>最大使用次数</Label>
          <Input v-model.number="formData.max_uses" type="number" placeholder="0 表示无限制" />
        </div>
        <div>
          <Label>过期时间</Label>
          <Input v-model="formData.expires_at" type="datetime-local" />
        </div>
      </div>
      <DialogFooter>
        <Button variant="outline" @click="isOpen = false">取消</Button>
        <Button @click="handleSave">{{ isEdit ? '保存' : '生成' }}</Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>
