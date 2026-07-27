<script setup lang="ts">
import { ref, computed } from 'vue'
import PushLog from '@/views/notify/components/PushLog.vue'

defineOptions({ name: 'PushLogTabWrapper' })

// 显式声明必填的非可选属性，完全对齐 PushLog 子组件强类型要求
defineProps<{
  filters: {
    keyword: string
    status: string
    [key: string]: any
  }
}>()

const pushLogRef = ref()

const showClearConfirm = computed({
  get() { return pushLogRef.value?.showClearConfirm },
  set(v: boolean) { if (pushLogRef.value) pushLogRef.value.showClearConfirm = v }
})

// 暴露标准代理接口给父级控制器 MessageLogs.vue 统一调用
defineExpose({
  fetchLogs: () => pushLogRef.value?.fetchLogs(),
  showClearConfirm
})
</script>

<template>
  <PushLog ref="pushLogRef" :filters="filters" v-bind="$attrs" />
</template>
