<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { Button } from '@/components/ui/button'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle
} from '@/components/ui/alert-dialog'
import { api } from '@/api'
import { toast } from 'vue-sonner'
import { Download, Upload, Archive } from 'lucide-vue-next'

const hasBackup = ref(false)
const backupTime = ref('')
const backupLoading = ref(false)
const restoreLoading = ref(false)
const fileInput = ref<HTMLInputElement>()
const showConfirm = ref(false)

async function checkBackupStatus() {
  try {
    const res = await api.settings.getBackupStatus()
    hasBackup.value = res.has_backup
    backupTime.value = res.backup_time || ''
  } catch {}
}

async function createBackup() {
  backupLoading.value = true
  try {
    await api.settings.createBackup()
    toast.success('备份创建成功')
    await checkBackupStatus()
  } catch (e: any) {
    toast.error(e.message || '备份失败')
  } finally {
    backupLoading.value = false
  }
}

function downloadBackup() {
  window.open(api.settings.downloadBackup(), '_blank')
  setTimeout(checkBackupStatus, 6000)
}

function showRestoreConfirm() {
  showConfirm.value = true
}

function confirmRestore() {
  showConfirm.value = false
  fileInput.value?.click()
}

function cancelRestore() {
  showConfirm.value = false
}

async function handleFileSelect(e: Event) {
  const target = e.target as HTMLInputElement
  const file = target.files?.[0]
  if (!file) return

  if (!file.name.endsWith('.zip')) {
    toast.error('请选择 .zip 备份文件')
    target.value = ''
    return
  }

  restoreLoading.value = true
  try {
    await api.settings.restoreBackup(file)
    toast.success('恢复成功，页面即将刷新')
    setTimeout(() => window.location.reload(), 1500)
  } catch (e: any) {
    toast.error(e.message || '恢复失败')
  } finally {
    restoreLoading.value = false
    target.value = ''
  }
}

onMounted(checkBackupStatus)
</script>

<template>
  <div class="space-y-4">
    <div class="flex items-center gap-4">
      <Button @click="createBackup" :disabled="backupLoading" variant="outline">
        <Archive class="w-4 h-4 mr-2" />
        {{ backupLoading ? '备份中...' : '创建备份' }}
      </Button>
      <Button v-if="hasBackup" @click="downloadBackup" variant="outline">
        <Download class="w-4 h-4 mr-2" />
        下载备份
      </Button>
      <span v-if="hasBackup && backupTime" class="text-xs text-muted-foreground">{{ backupTime }}</span>
    </div>
    <div class="text-xs text-muted-foreground">
      备份包含：任务、执行日志、环境变量、脚本、设置及 scripts 文件夹。第一次下载后，5分钟后文件将被删除。
    </div>
    <div class="border-t pt-4 mt-4">
      <div class="flex items-center gap-4">
        <Button @click="showRestoreConfirm" :disabled="restoreLoading" variant="outline">
          <Upload class="w-4 h-4 mr-2" />
          {{ restoreLoading ? '恢复中...' : '恢复备份' }}
        </Button>
        <input ref="fileInput" type="file" accept=".zip" class="hidden" @change="handleFileSelect" />
      </div>
      <div class="text-xs text-muted-foreground mt-2">
        上传 .zip 备份文件进行恢复，恢复会覆盖现有数据
      </div>
    </div>

    <AlertDialog :open="showConfirm" @update:open="showConfirm = $event">
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>确认恢复</AlertDialogTitle>
          <AlertDialogDescription>
            恢复备份将覆盖现有所有数据，此操作不可撤销。确定要继续吗？
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel @click="cancelRestore">取消</AlertDialogCancel>
          <AlertDialogAction @click="confirmRestore">确认恢复</AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  </div>
</template>
