<script setup lang="ts">
import { onMounted } from 'vue'
import { useRoute } from 'vue-router'

import MonthCalendar from '../../components/MonthCalendar.vue'
import NutritionProgress from '../../components/NutritionProgress.vue'
import { historyStore } from '../../stores/history'

const route = useRoute()
const mealLabels: Record<string, string> = { breakfast: '早餐', lunch: '午餐', dinner: '晚餐', snack: '加餐' }
const time = (value: string) => new Intl.DateTimeFormat('zh-CN', { hour: '2-digit', minute: '2-digit' }).format(new Date(value))

async function changeMonth(month: string) {
  await historyStore.loadMonth(month)
}

onMounted(async () => {
  const date = typeof route.query.date === 'string' ? route.query.date : historyStore.state.selectedDate
  await historyStore.loadMonth(date.slice(0, 7))
  await historyStore.selectDate(date)
})
</script>

<template>
  <main class="page">
    <header><div><p>FORMTALLY</p><h1>历史</h1></div><nav><RouterLink to="/today">今天</RouterLink><RouterLink to="/me">我的</RouterLink></nav></header>
    <MonthCalendar :month="historyStore.state.month" :selected="historyStore.state.selectedDate" :recorded-dates="historyStore.state.recordedDates" @select="historyStore.selectDate" @change-month="changeMonth" />
    <p v-if="historyStore.state.monthStatus === 'loading' || historyStore.state.dayStatus === 'loading'" role="status">正在加载记录…</p>
    <section v-else-if="historyStore.state.dayStatus === 'error'" class="state"><h2>暂时无法加载</h2><p role="alert">{{ historyStore.state.error }}</p><button type="button" @click="historyStore.selectDate(historyStore.state.selectedDate)">重试</button></section>
    <template v-else-if="historyStore.state.day">
      <h2 class="selected-date">{{ historyStore.state.selectedDate }}</h2>
      <section v-if="historyStore.state.day.progress" class="progress-grid" aria-label="当日营养进度">
        <NutritionProgress label="热量" unit="千卡" :progress="historyStore.state.day.progress.energy" />
        <NutritionProgress label="蛋白质" unit="克" :progress="historyStore.state.day.progress.protein" />
        <NutritionProgress label="碳水" unit="克" :progress="historyStore.state.day.progress.carb" />
        <NutritionProgress label="脂肪" unit="克" :progress="historyStore.state.day.progress.fat" />
      </section>
      <section v-if="historyStore.state.dayStatus === 'empty'" class="state"><h2>这天没有记录</h2><p>查看其他日期，或回到今天记录一餐。</p></section>
      <section v-else class="groups">
        <article v-for="group in historyStore.state.day.mealGroups" :key="group.mealType">
          <h2>{{ mealLabels[group.mealType] ?? group.mealType }}</h2>
          <RouterLink v-for="meal in group.meals" :key="meal.id" class="meal" :to="`/meals/${meal.id}`"><span>{{ time(meal.occurredAt) }}</span><strong>{{ meal.totals.energyKcal }} 千卡</strong><small>蛋白质 {{ meal.totals.proteinGrams }}g · 碳水 {{ meal.totals.carbGrams }}g · 脂肪 {{ meal.totals.fatGrams }}g</small></RouterLink>
        </article>
      </section>
    </template>
  </main>
</template>

<style scoped>
.page{width:min(100%,42rem);min-height:100svh;margin:auto;padding:1.25rem;display:grid;align-content:start;gap:1rem}header{display:flex;align-items:center;justify-content:space-between}header p,h1,.selected-date{margin:0}header p{color:#6a7773;font-size:.78rem;letter-spacing:.14em}nav{display:flex;gap:.8rem}nav a{min-height:44px;display:grid;place-items:center;color:#155e4e}.progress-grid{display:grid;grid-template-columns:1fr 1fr;gap:.7rem}.state{padding:1.25rem;border-radius:1rem;background:#eef7f3}.groups,.groups article{display:grid;gap:.7rem}.meal{display:grid;grid-template-columns:auto 1fr;gap:.3rem .8rem;padding:1rem;border:1px solid #e0e8e4;border-radius:1rem;background:#fff;color:inherit;text-decoration:none}.meal strong{text-align:right}.meal small{grid-column:1/-1;color:#6a7773}@media(max-width:420px){.progress-grid{grid-template-columns:1fr}}
</style>
