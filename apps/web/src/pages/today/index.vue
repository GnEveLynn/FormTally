<script setup lang="ts">
import { onMounted } from 'vue'
import { useRouter } from 'vue-router'
import NutritionProgress from '../../components/NutritionProgress.vue'
import { logout } from '../../api/auth'
import { sessionStore } from '../../stores/session'
import { todayStore } from '../../stores/today'

const router = useRouter()
const localDate = new Date().toLocaleDateString('en-CA')
const mealLabels: Record<string, string> = { breakfast: '早餐', lunch: '午餐', dinner: '晚餐', snack: '加餐' }
const time = (value: string) => new Intl.DateTimeFormat('zh-CN', { hour: '2-digit', minute: '2-digit' }).format(new Date(value))
const load = () => todayStore.load(localDate)
const signOut = async () => { await logout(); sessionStore.clear(); await router.replace('/welcome') }

onMounted(load)
</script>

<template>
  <main class="today">
    <header><div><p>FORMTALLY</p><h1>今天</h1></div><div class="actions"><RouterLink to="/history">历史</RouterLink><RouterLink to="/me">我的</RouterLink><a class="add" href="/meals/new">＋ 记录一餐</a><button class="logout" type="button" @click="signOut">退出登录</button></div></header>
    <p class="date">{{ localDate }}</p>
    <p v-if="todayStore.state.status === 'loading'" role="status">正在加载今天的记录…</p>
    <section v-else-if="todayStore.state.status === 'error'" class="state"><h2>暂时无法加载</h2><p role="alert">{{ todayStore.state.error }}</p><button @click="load">重试</button></section>
    <template v-else-if="todayStore.state.day">
      <section v-if="todayStore.state.day.progress" class="progress-grid" aria-label="今日营养进度">
        <NutritionProgress label="热量" unit="千卡" :progress="todayStore.state.day.progress.energy" />
        <NutritionProgress label="蛋白质" unit="克" :progress="todayStore.state.day.progress.protein" />
        <NutritionProgress label="碳水" unit="克" :progress="todayStore.state.day.progress.carb" />
        <NutritionProgress label="脂肪" unit="克" :progress="todayStore.state.day.progress.fat" />
      </section>
      <section v-if="todayStore.state.status === 'empty'" class="state"><h2>今天还没有记录</h2><p>拍张照片，或直接手动录入这一餐。</p><a class="primary" href="/meals/new">记录一餐</a></section>
      <section v-else class="groups">
        <article v-for="group in todayStore.state.day.mealGroups" :key="group.mealType">
          <h2>{{ mealLabels[group.mealType] ?? group.mealType }}</h2>
          <a v-for="meal in group.meals" :key="meal.id" class="meal" :href="`/meals/${meal.id}`"><span>{{ time(meal.occurredAt) }}</span><strong>{{ meal.totals.energyKcal }} 千卡</strong><small>蛋白质 {{ meal.totals.proteinGrams }}g · 碳水 {{ meal.totals.carbGrams }}g · 脂肪 {{ meal.totals.fatGrams }}g</small></a>
        </article>
      </section>
    </template>
  </main>
</template>

<style scoped>
.today{width:min(100%,42rem);margin:auto;min-height:100svh;padding:1.25rem;display:grid;align-content:start;gap:1rem}header{display:flex;align-items:center;justify-content:space-between;gap:1rem}header p,h1,.date{margin:0}header p{letter-spacing:.14em;color:#6a7773;font-size:.78rem}h1{font-size:2rem}.date{color:#6a7773}.actions{display:flex;align-items:center;gap:.5rem}.logout{min-height:44px;border:0;background:none;color:#6a7773}.add,.primary{display:inline-grid;place-items:center;min-height:46px;padding:0 1rem;border-radius:.9rem;color:white;background:#155e4e;text-decoration:none;font-weight:700}.progress-grid{display:grid;grid-template-columns:1fr 1fr;gap:.7rem}.state{display:grid;justify-items:start;gap:.7rem;padding:1.4rem;border-radius:1.1rem;background:#eef7f3}.state h2,.state p{margin:0}.state button{min-height:44px;padding:0 1rem;border:0;border-radius:.8rem;background:#155e4e;color:#fff}.groups,.groups article{display:grid;gap:.7rem}.groups h2{margin:.5rem 0 0}.meal{display:grid;grid-template-columns:auto 1fr;gap:.3rem .8rem;padding:1rem;border:1px solid #e0e8e4;border-radius:1rem;background:#fff;color:inherit;text-decoration:none}.meal strong{text-align:right}.meal small{grid-column:1/-1;color:#6a7773}@media(max-width:420px){.progress-grid{grid-template-columns:1fr}.actions{align-items:flex-end;flex-direction:column}}
</style>
