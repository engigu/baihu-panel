<script setup lang="ts">
import type { SwitchRootProps } from "reka-ui"
import type { HTMLAttributes } from "vue"
import { SwitchRoot, SwitchThumb } from "reka-ui"
import { cn } from "@/lib/utils"

const props = defineProps<SwitchRootProps & { class?: HTMLAttributes["class"] }>()

const emits = defineEmits(['update:modelValue'])

function onUpdate(value: boolean) {
  emits('update:modelValue', value)
}
</script>

<template>
  <SwitchRoot
    :model-value="props.modelValue"
    :default-value="props.defaultValue"
    :disabled="props.disabled"
    :name="props.name"
    :required="props.required"
    :value="props.value"
    :id="props.id"
    :as-child="props.asChild"
    :as="props.as"
    @update:model-value="onUpdate"
    :class="cn(
      'peer data-[state=checked]:bg-primary data-[state=unchecked]:bg-input focus-visible:border-ring focus-visible:ring-ring/50 dark:data-[state=unchecked]:bg-input/80 inline-flex h-[1.15rem] w-8 shrink-0 items-center rounded-full border border-transparent shadow-xs transition-all outline-none focus-visible:ring-[3px] disabled:cursor-not-allowed disabled:opacity-50',
      props.class,
    )"
  >
    <SwitchThumb
      data-slot="switch-thumb"
      :class="cn('bg-background dark:data-[state=unchecked]:bg-foreground dark:data-[state=checked]:bg-primary-foreground pointer-events-none block size-4 rounded-full ring-0 transition-transform data-[state=checked]:translate-x-[calc(100%-2px)] data-[state=unchecked]:translate-x-0')"
    />
  </SwitchRoot>
</template>
