<script setup lang="ts">
import AppYamlDialog from '@/views/apps/AppYamlDialog.vue'

defineProps<{
  open: boolean
  app?: any
  appName?: string
  yamlContent?: string
}>()

defineEmits<{
  'update:open': [value: boolean]
}>()
</script>

<template>
  <AppYamlDialog
    :open="open"
    :app="app || (appName ? { name: appName, id: appName.toLowerCase().replace(/\s+/g, '-') } : null)"
    :yaml="yamlContent"
    :show-deploy="false"
    dialog-title="导出应用配置"
    @update:open="$emit('update:open', $event)"
  />
</template>
