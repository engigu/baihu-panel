<script setup lang="ts">
import { ref, watch } from 'vue'
import { Button } from '@/components/ui/button'
import BaihuDialog from '@/components/ui/BaihuDialog.vue'
import { AlertTriangle, Terminal } from 'lucide-vue-next'
import { api } from '@/api'
import { toast } from 'vue-sonner'
import { ENV_TYPE } from '@/constants'

const emit = defineEmits<{
  (e: 'deleted'): void
}>()

const isOpen = ref(false)
const isDeleting = ref(false)
const deleteEnvId = ref<string | null>(null)
const envType = ref<string>(ENV_TYPE.NORMAL)
const associatedTasks = ref<any[]>([])

async function confirmDelete(id: string, type: string) {
  deleteEnvId.value = id
  envType.value = type
  try {
    const res = await api.env.tasks(id)
    associatedTasks.value = res || []
    isOpen.value = true
  } catch {
    toast.error('检查机密引用失败')
  }
}

async function deleteEnv(force = false) {
  if (!deleteEnvId.value) return
  isDeleting.value = true
  try {
    const res = await api.env.delete(deleteEnvId.value, force)
    if (res.code === 409) {
      associatedTasks.value = res.data || []
      isDeleting.value = false
      return
    }
    if (res.code !== 200) {
      toast.error(res.msg || '删除失败')
      isDeleting.value = false
      return
    }
    toast.success(envType.value === ENV_TYPE.SECRET ? '机密已删除' : '变量已删除')
    isOpen.value = false
    emit('deleted')
  } catch {
    toast.error('网络错误，删除失败')
  } finally {
    isDeleting.value = false
  }
}

watch(isOpen, (val) => {
  if (!val) {
    associatedTasks.value = []
    deleteEnvId.value = null
  }
})

defineExpose({
  confirmDelete
})
</script>

<template>
  <BaihuDialog v-model:open="isOpen" :title="associatedTasks.length > 0 ? '风险删除确认' : '确认删除'">
    <div v-if="associatedTasks.length > 0" class="space-y-4">
      <div class="flex items-start gap-4 p-4 rounded-xl bg-destructive/5 border border-destructive/10">
        <AlertTriangle class="h-5 w-5 text-destructive shrink-0" />
        <div class="space-y-1">
          <p class="text-sm font-bold text-destructive">{{ envType === ENV_TYPE.SECRET ? '机密' : '环境变量' }}正在使用中</p>
          <p class="text-[13px] text-muted-foreground/80 leading-relaxed">
            该{{ envType === ENV_TYPE.SECRET ? '机密' : '变量' }}已被以下任务引用，直接删除可能导致任务运行失败。建议先移除引用或选择“强制删除”。
          </p>
        </div>
      </div>

      <div class="space-y-2">
        <p class="text-[11px] font-bold text-muted-foreground uppercase tracking-widest px-1">关联任务 ({{ associatedTasks.length }})</p>
        <div class="bg-muted/30 rounded-xl p-2 max-h-48 overflow-y-auto space-y-1.5 border border-border/40">
          <div v-for="task in associatedTasks" :key="task.id"
            class="text-xs flex items-center justify-between bg-card p-2.5 rounded-lg border border-border/50 hover:border-primary/30 transition-all">
            <div class="flex items-center gap-2.5 min-w-0">
              <Terminal class="h-3.5 w-3.5 text-primary/70" />
              <span class="font-medium truncate">{{ task.name }}</span>
            </div>
            <code class="text-[10px] text-muted-foreground/70 font-mono bg-muted/50 px-1.5 py-0.5 rounded">{{ task.id }}</code>
          </div>
        </div>
      </div>

      <div class="p-4 rounded-xl bg-muted/20 border border-border/10">
        <p class="text-xs text-muted-foreground leading-relaxed italic">
          提示：选择强制删除将自动解除以上任务对该{{ envType === 'secret' ? '机密' : '变量' }}的绑定并执行物理删除。
        </p>
      </div>
    </div>
    <p v-else class="text-[15px] leading-relaxed text-muted-foreground">确定要删除此{{ envType === ENV_TYPE.SECRET ? '机密' : '环境变量' }}吗？此操作无法撤销，请谨慎操作。</p>

    <template #footer>
      <Button variant="ghost" :disabled="isDeleting" @click="isOpen = false">取消</Button>
      <Button v-if="associatedTasks.length > 0" variant="destructive" class="shadow-lg shadow-destructive/20" @click="deleteEnv(true)" :disabled="isDeleting">
        {{ isDeleting ? '删除中...' : '确认强制删除' }}
      </Button>
      <Button v-else variant="destructive" class="shadow-lg shadow-destructive/20" @click="deleteEnv(false)" :disabled="isDeleting">
        {{ isDeleting ? '删除中...' : '确认删除' }}
      </Button>
    </template>
  </BaihuDialog>
</template>
