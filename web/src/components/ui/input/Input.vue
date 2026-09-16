<script setup lang="ts">
import { ref, type HTMLAttributes } from "vue"
import { useVModel } from "@vueuse/core"
import { cn } from "@/lib/utils"

const props = defineProps<{
  defaultValue?: string | number
  modelValue?: string | number
  class?: HTMLAttributes["class"]
  readonly?: boolean
  autocomplete?: string
  name?: string
  type?: string
}>()

const emits = defineEmits<{
  (e: "update:modelValue", payload: string | number): void
  (e: "focus", payload: FocusEvent): void
  (e: "blur", payload: FocusEvent): void
  (e: "click", payload: MouseEvent): void
}>()

const modelValue = useVModel(props, "modelValue", emits, {
  passive: true,
  defaultValue: props.defaultValue,
})

// 生成不重复随机 name 避免 Chrome/Edge 密码管理器匹配保存的凭据
const inputName = props.name || `field_${Math.random().toString(36).substring(2, 9)}`

// 防自动填充：初始为 readonly，在用户主动交互（focus/click/mousedown）时解锁，失焦(blur)时恢复锁定
const isReadonly = ref(props.readonly ?? true)

function handleFocus(e: FocusEvent) {
  if (!props.readonly) {
    isReadonly.value = false
  }
  emits("focus", e)
}

function handleBlur(e: FocusEvent) {
  if (!props.readonly) {
    isReadonly.value = true
  }
  emits("blur", e)
}

function handleClick(e: MouseEvent) {
  if (!props.readonly) {
    isReadonly.value = false
  }
  emits("click", e)
}
</script>

<template>
  <input
    v-model="modelValue"
    data-slot="input"
    :type="props.type || 'text'"
    :name="inputName"
    :readonly="isReadonly"
    :autocomplete="props.autocomplete || 'new-password'"
    autocorrect="off"
    autocapitalize="off"
    spellcheck="false"
    data-lpignore="true"
    data-1p-ignore="true"
    data-bwignore="true"
    data-form-type="other"
    :class="cn(
      'file:text-foreground placeholder:text-muted-foreground selection:bg-primary selection:text-primary-foreground dark:bg-input/30 border-input h-9 w-full min-w-0 rounded-md border bg-transparent px-3 py-1 text-base shadow-xs transition-[color,box-shadow] outline-none file:inline-flex file:h-7 file:border-0 file:bg-transparent file:text-sm file:font-medium disabled:pointer-events-none disabled:cursor-not-allowed disabled:opacity-50 md:text-sm',
      'focus-visible:border-ring focus-visible:ring-ring/50 focus-visible:ring-[3px]',
      'aria-invalid:ring-destructive/20 dark:aria-invalid:ring-destructive/40 aria-invalid:border-destructive',
      props.class,
    )"
    @focus="handleFocus"
    @blur="handleBlur"
    @click="handleClick"
    @mousedown="handleFocus"
  >
</template>
