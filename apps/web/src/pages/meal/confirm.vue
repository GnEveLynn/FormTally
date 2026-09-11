<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRouter } from 'vue-router'
import { sumNutrition, validateMeal } from '@formtally/domain/meal-editor'
import AIEstimateNotice from '../../components/AIEstimateNotice.vue'
import MealItemEditor from '../../components/MealItemEditor.vue'
import { mealDraftStore } from '../../stores/meal-draft'
const router=useRouter();const errors=ref<Record<string,string>>({});const totals=computed(()=>sumNutrition(mealDraftStore.state.items.map(i=>i.nutrition)))
async function save(){errors.value=validateMeal({occurredAt:mealDraftStore.state.occurredAt,mealType:mealDraftStore.state.mealType,items:mealDraftStore.state.items});if(Object.keys(errors.value).length)return;const result=await mealDraftStore.save();if(result){mealDraftStore.clear();await router.replace('/today')}}
</script>
<template><main class="flow"><h1>确认识别结果</h1><AIEstimateNotice/><MealItemEditor :items="mealDraftStore.state.items" @update:items="mealDraftStore.state.items=$event"/><p v-if="errors.items" role="alert" class="error">{{ errors.items }}</p><section class="totals"><strong>约 {{ totals.energyKcal }} 千卡</strong><span>蛋白质 {{ totals.proteinGrams }} g · 碳水 {{ totals.carbGrams }} g · 脂肪 {{ totals.fatGrams }} g</span></section><p v-if="mealDraftStore.state.error" role="alert" class="error">{{ mealDraftStore.state.error }}</p><button class="primary" :disabled="mealDraftStore.state.status==='saving'" @click="save">{{ mealDraftStore.state.status==='saving'?'保存中…':'保存本餐' }}</button></main></template>
<style scoped>.flow{width:min(100%,32rem);margin:auto;min-height:100svh;padding:1.25rem;display:grid;align-content:start;gap:1rem}.totals{position:sticky;bottom:4.5rem;display:grid;gap:.35rem;padding:1rem;border-radius:1rem;background:#e7f8f0}.totals strong{font-size:1.5rem}.totals span{color:#48635d}.primary{min-height:50px;border:0;border-radius:.9rem;color:#fff;background:#155e4e;font:inherit;font-weight:700}.error{color:#a22b2b}</style>
