<script setup lang="ts">
import { ref, watch, nextTick, onBeforeUnmount, onMounted } from 'vue'

const props = withDefaults(defineProps<{
  modelValue?: string
  placeholder?: string
  rows?: number
  minHeightClass?: string
  maxHeightClass?: string
}>(), {
  modelValue: '',
  placeholder: '',
  rows: 5,
  minHeightClass: 'min-h-16',
  maxHeightClass: 'max-h-40'
})

const emit = defineEmits<{
  (e: 'update:modelValue', value: string): void
}>()

const valueTextareaRef = ref<HTMLTextAreaElement | null>(null)
const lineNumbersRef = ref<HTMLDivElement | null>(null)
const lineMeasureRef = ref<HTMLDivElement | null>(null)
const visualLineNumbers = ref<string[]>(['1'])
let textareaResizeObserver: ResizeObserver | null = null

function syncValueLineNumbers() {
  if (!valueTextareaRef.value || !lineNumbersRef.value) return
  lineNumbersRef.value.scrollTop = valueTextareaRef.value.scrollTop
}

async function updateVisualLineNumbers() {
  await nextTick()

  const textarea = valueTextareaRef.value
  const measure = lineMeasureRef.value
  if (!textarea || !measure) return

  const style = window.getComputedStyle(textarea)
  const lineHeight = Number.parseFloat(style.lineHeight) || Number.parseFloat(style.fontSize) * 1.5 || 24
  const lines = String(props.modelValue ?? '').split('\n')

  measure.style.width = `${textarea.clientWidth}px`
  measure.innerHTML = ''

  const nextLineNumbers: string[] = []
  lines.forEach((line, index) => {
    const lineEl = document.createElement('div')
    lineEl.className = 'break-all whitespace-pre-wrap'
    lineEl.textContent = line || ' '
    measure.appendChild(lineEl)

    const visualRows = Math.max(1, Math.round(lineEl.getBoundingClientRect().height / lineHeight))
    nextLineNumbers.push(String(index + 1))
    for (let i = 1; i < visualRows; i += 1) {
      nextLineNumbers.push('\u00A0')
    }
  })

  visualLineNumbers.value = nextLineNumbers.length > 0 ? nextLineNumbers : ['1']
  syncValueLineNumbers()
}

watch(() => props.modelValue, () => {
  void updateVisualLineNumbers()
})

onMounted(() => {
  updateVisualLineNumbers()
  if (valueTextareaRef.value) {
    textareaResizeObserver = new ResizeObserver(() => {
      void updateVisualLineNumbers()
    })
    textareaResizeObserver.observe(valueTextareaRef.value)
  }
})

onBeforeUnmount(() => {
  textareaResizeObserver?.disconnect()
})

function onInput(e: Event) {
  const target = e.target as HTMLTextAreaElement
  emit('update:modelValue', target.value)
}

// 暴露更新方法，供父组件弹窗打开等时机手动触发重绘
defineExpose({
  updateVisualLineNumbers
})
</script>

<template>
  <div>
    <div class="relative flex min-w-0 overflow-hidden rounded-md border border-input bg-transparent shadow-xs focus-within:border-ring focus-within:ring-ring/50 focus-within:ring-[3px]">
      <div ref="lineNumbersRef" :class="['flex w-6 shrink-0 flex-col overflow-hidden border-r border-border bg-muted/30 py-2 text-right font-mono text-[10px] leading-6 text-muted-foreground', maxHeightClass]">
        <span v-for="(line, index) in visualLineNumbers" :key="`${index}-${line}`" class="block h-6 px-1">{{ line }}</span>
      </div>
      <textarea
        ref="valueTextareaRef"
        :value="modelValue"
        @input="onInput"
        :rows="rows"
        :placeholder="placeholder"
        :class="['w-full min-w-0 resize-none overflow-x-hidden bg-transparent pl-2 pr-3 py-2 text-sm leading-6 break-all whitespace-pre-wrap outline-none', minHeightClass, maxHeightClass]"
        @scroll="syncValueLineNumbers"
      />
      <div aria-hidden="true" class="pointer-events-none absolute bottom-1.5 right-1.5 h-3.5 w-3.5 opacity-45">
        <span class="absolute bottom-0 right-0 h-px w-3 rotate-[-45deg] bg-border" />
        <span class="absolute bottom-1 right-0.5 h-px w-2 rotate-[-45deg] bg-border/80" />
        <span class="absolute bottom-2 right-1 h-px w-1 rotate-[-45deg] bg-border/60" />
      </div>
    </div>
    <div
      ref="lineMeasureRef"
      aria-hidden="true"
      :class="['pointer-events-none invisible fixed left-0 top-0 -z-10 break-all whitespace-pre-wrap px-2 py-2 text-sm leading-6', minHeightClass]"
    />
  </div>
</template>
