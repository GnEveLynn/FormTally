<script setup lang="ts">
import type { Meal } from '@formtally/api-contract/meals'
import { onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'

import { deleteMeal, getMeal, removeMealImage } from '../../api/meals'
import ConfirmDialog from '../../components/ConfirmDialog.vue'
import { ApiError } from '../../api/http'
import { historyStore } from '../../stores/history'

const route = useRoute()
const router = useRouter()
const meal = ref<Meal | null>(null)
const status = ref<'loading'|'ready'|'error'>('loading')
const error = ref('')
const confirmation = ref<'image'|'meal'|null>(null)
const busy = ref(false)
const id = String(route.params.id)
const mealLabels:Record<string,string>={breakfast:'早餐',lunch:'午餐',dinner:'晚餐',snack:'加餐'}

async function load() {
  status.value = 'loading'
  try { meal.value = await getMeal(id); status.value = 'ready' } catch (cause) { status.value = 'error'; error.value = cause instanceof Error ? cause.message : '加载失败' }
}
async function removeImage() {
  if (!meal.value) return
  busy.value = true
  try { meal.value = await removeMealImage(id, meal.value.revision); confirmation.value = null } catch (cause) { await handleConflict(cause) } finally { busy.value = false }
}
async function removeMeal() {
  if (!meal.value) return
  busy.value = true
  try { const result = await deleteMeal(id, meal.value.revision); await historyStore.refreshDates(result.affectedLocalDates); await router.replace({ path: '/history', query: { date: result.affectedLocalDates[0] } }) } catch (cause) { await handleConflict(cause) } finally { busy.value = false }
}
async function handleConflict(cause: unknown) {
  confirmation.value = null
  if (cause instanceof ApiError && cause.code === 'REVISION_CONFLICT') { error.value = '数据已变化，已重新载入最新内容'; await load() } else error.value = cause instanceof Error ? cause.message : '操作失败'
}
onMounted(load)
</script>

<template>
  <main class="page">
    <RouterLink class="back" :to="meal ? `/history?date=${meal.localDate}` : '/history'">← 返回历史</RouterLink>
    <p v-if="status==='loading'" role="status">正在加载餐食…</p>
    <p v-else-if="status==='error'" role="alert">{{ error }}</p>
    <template v-else-if="meal">
      <header><div><p>{{ meal.localDate }} · {{ mealLabels[meal.mealType] }}</p><h1>{{ meal.totals.energyKcal }} 千卡</h1></div><RouterLink class="edit" :to="`/meals/${meal.id}/edit`">编辑</RouterLink></header>
      <img v-if="meal.image" :src="meal.image.url" alt="餐食图片">
      <p class="notice">估算值：{{ meal.estimateNotice }}</p>
      <section><article v-for="item in meal.items" :key="item.id ?? item.name"><h2>{{ item.name }}</h2><p>{{ item.grams }} 克 · {{ item.nutrition.energyKcal }} 千卡</p><small>蛋白质 {{ item.nutrition.proteinGrams }}g · 碳水 {{ item.nutrition.carbGrams }}g · 脂肪 {{ item.nutrition.fatGrams }}g</small></article></section>
      <p v-if="error" role="alert">{{ error }}</p>
      <div class="danger-actions"><button v-if="meal.image" type="button" @click="confirmation='image'">仅移除图片</button><button type="button" @click="confirmation='meal'">删除整餐</button></div>
    </template>
    <ConfirmDialog :open="confirmation==='image'" title="仅移除这张图片？" confirm-label="仅移除图片" :busy="busy" @cancel="confirmation=null" @confirm="removeImage"><p>食物和营养数据会保留。</p></ConfirmDialog>
    <ConfirmDialog :open="confirmation==='meal'" title="删除这餐？" confirm-label="删除整餐" :busy="busy" @cancel="confirmation=null" @confirm="removeMeal"><p>餐食、明细和图片都会进入删除流程。</p></ConfirmDialog>
  </main>
</template>

<style scoped>
.page{width:min(100%,38rem);min-height:100svh;margin:auto;padding:1.25rem;display:grid;align-content:start;gap:1rem}.back{min-height:44px;display:flex;align-items:center;color:#155e4e}header{display:flex;justify-content:space-between;align-items:center}header p,h1{margin:.25rem 0}.edit{min-height:44px;padding:0 1rem;display:grid;place-items:center;border-radius:.8rem;color:#fff;background:#155e4e;text-decoration:none}img{width:100%;max-height:22rem;object-fit:cover;border-radius:1rem}.notice{padding:.75rem;border-radius:.75rem;background:#fff2d8}section{display:grid;gap:.75rem}article{padding:1rem;border:1px solid #dbe5e1;border-radius:1rem;background:#fff}article h2,article p{margin:0 0 .35rem}.danger-actions{display:flex;justify-content:flex-end;gap:.75rem}.danger-actions button{min-height:44px;padding:0 1rem;border:1px solid #a22b2b;border-radius:.75rem;color:#a22b2b;background:#fff}
</style>
