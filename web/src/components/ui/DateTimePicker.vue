<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'
import { Button } from '@/components/ui/button'
import { ChevronLeft, ChevronRight, Calendar as CalendarIcon } from 'lucide-vue-next'
import { cn } from '@/lib/utils'

const modelValue = defineModel<string>({ default: '' })

const props = withDefaults(defineProps<{
  placeholder?: string
  disabled?: boolean
  class?: string
  type?: 'datetime' | 'date'
}>(), {
  type: 'datetime'
})

// Custom Calendar Picker State
const pickerDate = ref<Date | null>(null)
const viewYear = ref(new Date().getFullYear())
const viewMonth = ref(new Date().getMonth()) // 0-11
const hour = ref(0)
const minute = ref(0)
const isPopoverOpen = ref(false)

// Sync external value to calendar picker state
watch(modelValue, (val) => {
  if (val) {
    const d = new Date(val)
    if (!isNaN(d.getTime())) {
      pickerDate.value = d
      viewYear.value = d.getFullYear()
      viewMonth.value = d.getMonth()
      hour.value = d.getHours()
      minute.value = d.getMinutes()
      return
    }
  }
  pickerDate.value = null
}, { immediate: true })

function updateValue() {
  if (!pickerDate.value) {
    modelValue.value = ''
    return
  }
  const d = new Date(pickerDate.value)
  
  const yyyy = d.getFullYear()
  const mm = String(d.getMonth() + 1).padStart(2, '0')
  const dd = String(d.getDate()).padStart(2, '0')
  
  if (props.type === 'date') {
    modelValue.value = `${yyyy}-${mm}-${dd}`
  } else {
    d.setHours(hour.value)
    d.setMinutes(minute.value)
    const hh = String(d.getHours()).padStart(2, '0')
    const min = String(d.getMinutes()).padStart(2, '0')
    modelValue.value = `${yyyy}-${mm}-${dd}T${hh}:${min}`
  }
}

function selectDay(day: number) {
  const d = new Date(viewYear.value, viewMonth.value, day)
  pickerDate.value = d
  updateValue()
}

const calendarDays = computed(() => {
  const year = viewYear.value
  const month = viewMonth.value
  
  const firstDay = new Date(year, month, 1)
  let firstDayOfWeek = firstDay.getDay() - 1
  if (firstDayOfWeek < 0) firstDayOfWeek = 6 // Mon = 0, Sun = 6
  
  const totalDays = new Date(year, month + 1, 0).getDate()
  const prevMonthTotalDays = new Date(year, month, 0).getDate()
  
  const days: { day: number; isCurrent: boolean; isSelected: boolean; isToday: boolean }[] = []
  
  // Prev month filler
  for (let i = firstDayOfWeek - 1; i >= 0; i--) {
    days.push({
      day: prevMonthTotalDays - i,
      isCurrent: false,
      isSelected: false,
      isToday: false
    })
  }
  
  // Current month
  const today = new Date()
  for (let i = 1; i <= totalDays; i++) {
    const isSelected = !!(pickerDate.value && 
      pickerDate.value.getFullYear() === year && 
      pickerDate.value.getMonth() === month && 
      pickerDate.value.getDate() === i)
      
    const isToday = today.getFullYear() === year && 
      today.getMonth() === month && 
      today.getDate() === i
      
    days.push({
      day: i,
      isCurrent: true,
      isSelected,
      isToday
    })
  }
  
  // Next month filler
  const remaining = 42 - days.length
  for (let i = 1; i <= remaining; i++) {
    days.push({
      day: i,
      isCurrent: false,
      isSelected: false,
      isToday: false
    })
  }
  
  return days
})

function prevMonth() {
  if (viewMonth.value === 0) {
    viewMonth.value = 11
    viewYear.value--
  } else {
    viewMonth.value--
  }
}

function nextMonth() {
  if (viewMonth.value === 11) {
    viewMonth.value = 0
    viewYear.value++
  } else {
    viewMonth.value++
  }
}

function setToday() {
  const now = new Date()
  pickerDate.value = now
  viewYear.value = now.getFullYear()
  viewMonth.value = now.getMonth()
  hour.value = now.getHours()
  minute.value = now.getMinutes()
  updateValue()
}

function clearDate() {
  pickerDate.value = null
  modelValue.value = ''
  isPopoverOpen.value = false
}

function formatDateTime(dateStr: string) {
  if (!dateStr) return ''
  const d = new Date(dateStr)
  if (isNaN(d.getTime())) return ''
  const yyyy = d.getFullYear()
  const mm = String(d.getMonth() + 1).padStart(2, '0')
  const dd = String(d.getDate()).padStart(2, '0')
  if (props.type === 'date') {
    return `${yyyy}-${mm}-${dd}`
  }
  const hh = String(d.getHours()).padStart(2, '0')
  const min = String(d.getMinutes()).padStart(2, '0')
  return `${yyyy}-${mm}-${dd} ${hh}:${min}`
}

function incrementHour() {
  hour.value = (hour.value + 1) % 24
  updateValue()
}

function decrementHour() {
  hour.value = (hour.value - 1 + 24) % 24
  updateValue()
}

function incrementMinute() {
  minute.value = (minute.value + 1) % 60
  updateValue()
}

function decrementMinute() {
  minute.value = (minute.value - 1 + 60) % 60
  updateValue()
}

// Support manual keyboard input for time
const hourStr = ref('00')
const minuteStr = ref('00')

watch(hour, (newVal) => {
  if (hourStr.value === '') return
  if (parseInt(hourStr.value, 10) === newVal) return
  hourStr.value = String(newVal).padStart(2, '0')
}, { immediate: true })

watch(minute, (newVal) => {
  if (minuteStr.value === '') return
  if (parseInt(minuteStr.value, 10) === newVal) return
  minuteStr.value = String(newVal).padStart(2, '0')
}, { immediate: true })

function handleHourInput() {
  let clean = hourStr.value.replace(/\D/g, '')
  hourStr.value = clean
  if (clean === '') {
    hour.value = 0
    updateValue()
    return
  }
  let val = parseInt(clean, 10)
  if (val > 23) val = 23
  hour.value = val
  updateValue()
}

function handleHourBlur() {
  hourStr.value = String(hour.value).padStart(2, '0')
}

function handleMinuteInput() {
  let clean = minuteStr.value.replace(/\D/g, '')
  minuteStr.value = clean
  if (clean === '') {
    minute.value = 0
    updateValue()
    return
  }
  let val = parseInt(clean, 10)
  if (val > 59) val = 59
  minute.value = val
  updateValue()
}

function handleMinuteBlur() {
  minuteStr.value = String(minute.value).padStart(2, '0')
}
</script>

<template>
  <Popover v-model:open="isPopoverOpen">
    <PopoverTrigger as-child>
      <Button
        variant="outline"
        :disabled="disabled"
        :class="cn('h-9 w-full justify-start text-left font-normal text-xs px-3 bg-muted/20 border-muted-foreground/10 focus:bg-background cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed', $props.class)"
      >
        <CalendarIcon class="mr-2 h-3.5 w-3.5 opacity-70" />
        <span>{{ modelValue ? formatDateTime(modelValue) : (placeholder || '永久有效') }}</span>
      </Button>
    </PopoverTrigger>
    <PopoverContent class="w-60 p-2.5 bg-popover border border-border/40 rounded-lg shadow-md z-[100]" align="start">
      <!-- Month/Year Header -->
      <div class="flex items-center justify-between mb-2">
        <span class="text-xs font-semibold select-none">{{ viewYear }}年{{ viewMonth + 1 }}月</span>
        <div class="flex gap-0.5">
          <Button variant="ghost" size="icon" class="h-6 w-6" type="button" @click="prevMonth">
            <ChevronLeft class="h-3.5 w-3.5" />
          </Button>
          <Button variant="ghost" size="icon" class="h-6 w-6" type="button" @click="nextMonth">
            <ChevronRight class="h-3.5 w-3.5" />
          </Button>
        </div>
      </div>
      <!-- Weekdays Header -->
      <div class="grid grid-cols-7 gap-0.5 text-center text-[10px] font-medium text-muted-foreground/80 mb-1 select-none">
        <span>一</span><span>二</span><span>三</span><span>四</span><span>五</span><span>六</span><span>日</span>
      </div>
      <!-- Calendar Grid -->
      <div class="grid grid-cols-7 gap-0.5">
        <button
          v-for="(cell, idx) in calendarDays"
          :key="idx"
          @click="cell.isCurrent && selectDay(cell.day)"
          :disabled="!cell.isCurrent"
          type="button"
          class="h-6 w-full text-[10px] rounded-md transition-all flex items-center justify-center cursor-pointer select-none"
          :class="[
            cell.isCurrent ? 'hover:bg-accent hover:text-accent-foreground text-foreground' : 'opacity-20 cursor-not-allowed text-muted-foreground',
            cell.isSelected ? 'bg-primary text-primary-foreground font-semibold hover:bg-primary hover:text-primary-foreground' : '',
            cell.isToday && !cell.isSelected ? 'border border-primary/40 text-primary font-semibold' : ''
          ]"
        >
          {{ cell.day }}
        </button>
      </div>
      <!-- Time Selection -->
      <div v-if="type === 'datetime'" class="flex items-center justify-between mt-3 pt-2.5 border-t border-border/20 px-1">
        <span class="text-[10px] text-muted-foreground font-medium select-none">时间</span>
        <div class="flex items-center gap-1.5">
          <!-- Hour Stepper -->
          <div class="flex items-center border border-border/10 rounded-md bg-muted/15 overflow-hidden h-7">
            <button type="button" @click="decrementHour" class="w-5 h-full hover:bg-accent text-muted-foreground hover:text-foreground text-[10px] font-bold cursor-pointer select-none transition-colors">-</button>
            <input
              type="text"
              v-model="hourStr"
              @input="handleHourInput"
              @blur="handleHourBlur"
              @keydown.enter="handleHourBlur"
              class="w-7 text-center text-[10px] font-mono font-semibold text-foreground bg-transparent border-none p-0 focus:ring-0 focus:outline-hidden"
              maxlength="2"
            />
            <button type="button" @click="incrementHour" class="w-5 h-full hover:bg-accent text-muted-foreground hover:text-foreground text-[10px] font-bold cursor-pointer select-none transition-colors">+</button>
          </div>
          <span class="text-[10px] text-muted-foreground font-bold select-none">:</span>
          <!-- Minute Stepper -->
          <div class="flex items-center border border-border/10 rounded-md bg-muted/15 overflow-hidden h-7">
            <button type="button" @click="decrementMinute" class="w-5 h-full hover:bg-accent text-muted-foreground hover:text-foreground text-[10px] font-bold cursor-pointer select-none transition-colors">-</button>
            <input
              type="text"
              v-model="minuteStr"
              @input="handleMinuteInput"
              @blur="handleMinuteBlur"
              @keydown.enter="handleMinuteBlur"
              class="w-7 text-center text-[10px] font-mono font-semibold text-foreground bg-transparent border-none p-0 focus:ring-0 focus:outline-hidden"
              maxlength="2"
            />
            <button type="button" @click="incrementMinute" class="w-5 h-full hover:bg-accent text-muted-foreground hover:text-foreground text-[10px] font-bold cursor-pointer select-none transition-colors">+</button>
          </div>
        </div>
      </div>
      <!-- Footer Action Buttons -->
      <div class="flex items-center justify-between mt-2.5 pt-2 border-t border-border/20">
        <Button variant="ghost" size="sm" class="h-6 text-[10px] px-2 text-destructive hover:bg-destructive/10 cursor-pointer" type="button" @click="clearDate">清除</Button>
        <div class="flex gap-1">
          <Button variant="ghost" size="sm" class="h-6 text-[10px] px-2 cursor-pointer" type="button" @click="setToday">今天</Button>
          <Button size="sm" class="h-6 text-[10px] px-2 cursor-pointer" type="button" @click="isPopoverOpen = false">确定</Button>
        </div>
      </div>
    </PopoverContent>
  </Popover>
</template>
