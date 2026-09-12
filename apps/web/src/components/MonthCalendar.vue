<script setup lang="ts">
const props = defineProps<{ month: string; selected: string; recordedDates: Set<string> }>()
const emit = defineEmits<{ select: [date: string]; 'change-month': [month: string] }>()

const days = () => {
  const [year, month] = props.month.split('-').map(Number)
  const count = new Date(Date.UTC(year, month, 0)).getUTCDate()
  return Array.from({ length: count }, (_, index) => `${year}-${String(month).padStart(2, '0')}-${String(index + 1).padStart(2, '0')}`)
}
const firstColumn = () => {
  const [year, month] = props.month.split('-').map(Number)
  return new Date(Date.UTC(year, month - 1, 1)).getUTCDay() || 7
}

function adjacent(delta: number) {
  const [year, month] = props.month.split('-').map(Number)
  const value = new Date(Date.UTC(year, month - 1 + delta, 1))
  return `${value.getUTCFullYear()}-${String(value.getUTCMonth() + 1).padStart(2, '0')}`
}
</script>

<template>
  <section class="calendar" aria-label="历史月份">
    <header>
      <button type="button" data-action="previous-month" aria-label="上个月" @click="emit('change-month', adjacent(-1))">‹</button>
      <h2>{{ month }}</h2>
      <button type="button" data-action="next-month" aria-label="下个月" @click="emit('change-month', adjacent(1))">›</button>
    </header>
    <div class="weekdays" aria-hidden="true"><span v-for="label in ['一','二','三','四','五','六','日']" :key="label">{{ label }}</span></div>
    <div class="days">
      <button
        v-for="(date, index) in days()"
        :key="date"
        type="button"
        :data-date="date"
        :class="{ selected: selected === date, recorded: recordedDates.has(date) }"
        :aria-label="`${date}${recordedDates.has(date) ? '，有记录' : ''}`"
        :style="index === 0 ? { gridColumnStart: firstColumn() } : undefined"
        @click="emit('select', date)"
      >{{ Number(date.slice(-2)) }}</button>
    </div>
  </section>
</template>

<style scoped>
.calendar{padding:1rem;border:1px solid #dbe5e1;border-radius:1rem;background:#fff}.calendar header{display:grid;grid-template-columns:44px 1fr 44px;align-items:center}.calendar h2{margin:0;text-align:center;font-size:1.1rem}.calendar button{min-width:44px;min-height:44px;border:0;border-radius:.75rem;background:transparent;font:inherit}.weekdays,.days{display:grid;grid-template-columns:repeat(7,1fr);text-align:center}.weekdays span{padding:.5rem 0;color:#6a7773;font-size:.8rem}.days button.recorded::after{content:'';display:block;width:5px;height:5px;margin:2px auto 0;border-radius:50%;background:#16806b}.days button.selected{color:#fff;background:#155e4e}.days button.selected::after{background:#fff}
</style>
