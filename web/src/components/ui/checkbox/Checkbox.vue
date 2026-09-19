<script setup lang="ts">
import type { CheckboxRootProps } from "reka-ui"
import type { HTMLAttributes } from "vue"
import { Check } from "lucide-vue-next"
import { CheckboxIndicator, CheckboxRoot } from "reka-ui"
import { cn } from "@/lib/utils"

const props = defineProps<CheckboxRootProps & {
  class?: HTMLAttributes["class"]
  checked?: boolean | 'indeterminate'
}>()

const emits = defineEmits<{
  (e: 'update:modelValue', val: boolean | 'indeterminate'): void
  (e: 'update:checked', val: boolean | 'indeterminate'): void
}>()

function onUpdate(value: boolean | 'indeterminate') {
  emits('update:modelValue', value)
  emits('update:checked', value)
}
</script>

<template>
  <CheckboxRoot
    :model-value="props.checked !== undefined ? props.checked : props.modelValue"
    :default-value="props.defaultValue"
    :disabled="props.disabled"
    :name="props.name"
    :required="props.required"
    :value="props.value"
    :id="props.id"
    :as-child="props.asChild"
    :as="props.as"
    @update:model-value="onUpdate"
    data-slot="checkbox"
    :class="
      cn('peer border-muted-foreground/30 data-[state=unchecked]:bg-background/40 data-[state=checked]:bg-primary data-[state=checked]:text-primary-foreground data-[state=checked]:border-primary focus-visible:border-ring focus-visible:ring-ring/50 aria-invalid:ring-destructive/20 dark:aria-invalid:ring-destructive/40 aria-invalid:border-destructive size-4 shrink-0 rounded-[4px] border transition-all outline-none focus-visible:ring-[3px] disabled:cursor-not-allowed disabled:opacity-50',
         props.class)"
  >
    <CheckboxIndicator
      data-slot="checkbox-indicator"
      class="grid place-content-center text-current transition-none"
    >
      <slot>
        <Check class="size-3.5" />
      </slot>
    </CheckboxIndicator>
  </CheckboxRoot>
</template>
