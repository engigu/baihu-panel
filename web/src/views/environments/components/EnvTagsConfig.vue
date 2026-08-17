<script setup lang="ts">
import { ref, computed } from 'vue'
import { Label } from '@/components/ui/label'
import TagInput from '@/components/TagInput.vue'
import { X } from 'lucide-vue-next'
import { api } from '@/api'

const props = defineProps<{
  modelValue?: string
}>()

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()

const tagInput = ref('')

const tagsList = computed(() => {
  return props.modelValue ? props.modelValue.split(',').filter(Boolean) : []
})

function addTag(passedTag?: string) {
  const val = (passedTag || tagInput.value).trim()
  if (!val) return
  const currentTags = [...tagsList.value]
  if (!currentTags.includes(val)) {
    currentTags.push(val)
    emit('update:modelValue', currentTags.join(','))
  }
  tagInput.value = ''
}

function removeTag(tagToRemove: string) {
  const currentTags = tagsList.value.filter(t => t !== tagToRemove)
  emit('update:modelValue', currentTags.join(','))
}
</script>

<template>
  <div class="space-y-2 min-w-0">
    <Label class="text-sm">变量标签</Label>
    <div class="flex flex-wrap gap-1.5 p-2 min-h-[42px] bg-muted/20 border border-muted-foreground/15 rounded-md focus-within:border-primary/30 transition-colors">
      <span v-for="tag in tagsList" :key="tag" 
        class="flex items-center gap-1.5 bg-primary/5 text-primary px-2.5 py-1 rounded-full text-[11px] font-medium border border-primary/10 group transition-all hover:bg-primary/10">
        {{ tag }}
        <button type="button" class="text-primary/40 hover:text-destructive transition-colors shrink-0" @click.prevent="removeTag(tag)"><X class="h-3 w-3" /></button>
      </span>
      <div class="flex-1 min-w-[100px]">
        <TagInput v-model="tagInput" placeholder="输入并回车添加标签..." 
          clearOnSelect
          :fetchTags="api.env.tags"
          class="h-6 border-none bg-transparent shadow-none focus-visible:ring-0 px-0 text-xs"
          @enter="addTag" />
      </div>
    </div>
  </div>
</template>
