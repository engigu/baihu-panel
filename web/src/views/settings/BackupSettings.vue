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
  <div class="space-y-6">
    <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
      <!-- 备份 Card -->
      <div class="p-4 rounded-lg border border-border bg-muted/20 flex flex-col justify-between space-y-4">
        <div class="space-y-2">
          <div class="flex items-center gap-2">
            <Archive class="w-4 h-4 text-foreground/80" />
            <h4 class="text-xs font-semibold text-foreground">数据备份</h4>
          </div>
          <p class="text-[10px] text-muted-foreground leading-relaxed">
            备份包含任务、执行日志、环境变量、脚本、系统设置及整个 scripts 文件夹。
          </p>
          <p class="text-[10px] text-amber-600 dark:text-amber-500 font-medium">
            提示：备份文件在第一次被下载 5 分钟后将被系统自动物理删除。
          </p>
        </div>
        
        <div class="space-y-2">
          <div v-if="hasBackup && backupTime" class="text-[10px] text-muted-foreground flex items-center gap-1.5">
            <span class="inline-block w-1.5 h-1.5 rounded-full bg-green-500 animate-pulse shrink-0"></span>
            备份生成时间: {{ backupTime }}
          </div>
          <div class="flex flex-wrap gap-2.5">
            <Button @click="createBackup" :disabled="backupLoading" class="h-9 text-xs shadow-sm">
              {{ backupLoading ? '备份中...' : '创建备份' }}
            </Button>
            <Button v-if="hasBackup" @click="downloadBackup" variant="outline" class="h-9 text-xs shadow-sm">
              <Download class="w-3.5 h-3.5 mr-1.5" />
              下载备份
            </Button>
          </div>
        </div>
      </div>

      <!-- 恢复 Card -->
      <div class="p-4 rounded-lg border border-border bg-muted/20 flex flex-col justify-between space-y-4">
        <div class="space-y-2">
          <div class="flex items-center gap-2">
            <Upload class="w-4 h-4 text-foreground/80" />
            <h4 class="text-xs font-semibold text-foreground">数据恢复</h4>
          </div>
          <p class="text-[10px] text-muted-foreground leading-relaxed">
            上传此前下载的 .zip 格式备份文件，将当前系统数据与配置恢复到当时的快照状态。
          </p>
          <p class="text-[10px] text-destructive font-medium">
            警告：恢复操作是不可逆的，且会完全覆盖系统现存的所有数据和配置。
          </p>
        </div>

        <div>
          <Button @click="showRestoreConfirm" :disabled="restoreLoading" variant="outline" class="h-9 text-xs shadow-sm w-full sm:w-auto">
            {{ restoreLoading ? '恢复中...' : '恢复备份' }}
          </Button>
          <input ref="fileInput" type="file" accept=".zip" class="hidden" @change="handleFileSelect" />
        </div>
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
