<script setup lang="ts">
import { ref, onMounted, watch, computed, onUnmounted } from 'vue'
import { api } from '@/api'
import { Input } from '@/components/ui/input'
import { Loader2, Check } from 'lucide-vue-next'

const props = withDefaults(defineProps<{
  modelValue: string
  placeholder?: string
  icon?: any
  multiple?: boolean
  clearOnSelect?: boolean
}>(), {
  multiple: false,
  clearOnSelect: false
})

const emit = defineEmits(['update:modelValue', 'enter'])

const open = ref(false)
const allTags = ref<string[]>([])
const loading = ref(false)
const inputValue = ref(props.modelValue)
const containerRef = ref<HTMLElement | null>(null)

watch(() => props.modelValue, (newVal) => {
  inputValue.value = newVal
})

async function fetchTags() {
  loading.value = true
  try {
    const res = await api.tasks.tags()
    allTags.value = res || []
  } catch (e) {
    console.error('Failed to fetch tags', e)
  } finally {
    loading.value = false
  }
}

const currentTags = computed(() => {
  return (inputValue.value || '').split(',').map(t => t.trim()).filter(Boolean)
})

const filteredTags = computed(() => {
  const parts = (inputValue.value || '').split(',')
  const query = (parts[parts.length - 1] || '').trim().toLowerCase()
  
  if (!query) {
    return allTags.value.slice(0, 10)
  }
  return allTags.value.filter(t => t.toLowerCase().includes(query))
})

function selectTag(tag: string) {
  if (props.multiple) {
    const tags = [...currentTags.value]
    const index = tags.indexOf(tag)
    if (index > -1) {
      tags.splice(index, 1)
    } else {
      tags.push(tag)
    }
    // 多选模式下追加逗号，以便 filteredTags 计算 query 为空，从而显示所有候选
    inputValue.value = tags.length > 0 ? tags.join(',') + ',' : ''
    emit('update:modelValue', inputValue.value)
  } else {
    if (props.clearOnSelect) {
      inputValue.value = ''
      emit('update:modelValue', '')
    } else {
      inputValue.value = tag
      emit('update:modelValue', tag)
    }
    open.value = false
    // 即使清空了也发出 enter 事件，让父组件知道选中了一个值（如果需要通过 enter 处理）
    // 或者我们传递选中的 tag 给 enter
    emit('enter', tag)
  }
}

function onInput() {
  emit('update:modelValue', inputValue.value)
  if (!open.value && filteredTags.value.length > 0) {
    open.value = true
  }
}

function onEnter() {
  open.value = false
  emit('enter')
}

function handleClickOutside(e: MouseEvent) {
  if (containerRef.value && !containerRef.value.contains(e.target as Node)) {
    if (open.value) {
      open.value = false
    }
  }
}

onMounted(() => {
  fetchTags()
  window.addEventListener('mousedown', handleClickOutside)
})

onUnmounted(() => {
  window.removeEventListener('mousedown', handleClickOutside)
})
</script>

<template>
  <div ref="containerRef" class="relative w-full">
    <div class="relative group">
      <component 
        v-if="icon" 
        :is="icon" 
        class="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground group-focus-within:text-primary transition-colors pointer-events-none" 
      />
      <Input 
        v-model="inputValue" 
        :placeholder="placeholder" 
        :class="[icon ? 'pl-9' : 'pl-3', $attrs.class]"
        class="cursor-pointer"
        @input="onInput"
        @keydown.enter="onEnter"
        @click="open = true"
        @focus="open = true"
      />
    </div>

    <!-- 模拟下拉列表 -->
    <div 
      v-if="open" 
      class="absolute z-[100] top-full left-0 mt-1 w-full min-w-[200px] bg-popover text-popover-foreground rounded-md border shadow-xl p-1 animate-in fade-in zoom-in-95 duration-100"
    >
      <div v-if="loading" class="flex items-center justify-center py-4">
        <Loader2 class="h-4 w-4 animate-spin text-muted-foreground" />
      </div>
      <div v-else-if="filteredTags.length === 0" class="py-2 px-3 text-[11px] text-muted-foreground">
        无匹配标签
      </div>
      <div v-else class="max-h-[300px] overflow-y-auto space-y-0.5">
        <button
          v-for="tag in filteredTags"
          :key="tag"
          class="w-full text-left px-3 py-2 text-xs rounded-md hover:bg-primary/10 hover:text-primary transition-colors flex items-center justify-between group/item"
          @mousedown.prevent="selectTag(tag)"
        >
          <span class="truncate font-medium">{{ tag }}</span>
          <Check v-if="currentTags.includes(tag)" class="h-3.5 w-3.5 text-primary shrink-0" />
        </button>
      </div>
    </div>
  </div>
</template>
