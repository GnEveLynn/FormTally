<script setup lang="ts">
import type { Meal, UpdateMealInput } from '@formtally/api-contract/meals'
import { onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'

import { getMeal, updateMeal } from '../../api/meals'
import { ApiError } from '../../api/http'
import MealItemEditor from '../../components/MealItemEditor.vue'
import { historyStore } from '../../stores/history'
import { validateMeal } from '@formtally/domain/meal-editor'

const route=useRoute(),router=useRouter(),id=String(route.params.id)
const meal=ref<Meal|null>(null),occurredAt=ref(''),mealType=ref('lunch'),items=ref<Meal['items']>([]),busy=ref(false),error=ref('')
function apply(value:Meal){meal.value=value;occurredAt.value=value.occurredAt.slice(0,16);mealType.value=value.mealType;items.value=value.items.map(item=>({...item,nutrition:{...item.nutrition}}))}
async function load(){try{apply(await getMeal(id))}catch(cause){error.value=cause instanceof Error?cause.message:'加载失败'}}
function withOffset(value:string){const date=new Date(value);const offset=-date.getTimezoneOffset(),sign=offset>=0?'+':'-',absolute=Math.abs(offset);return `${value}:00${sign}${String(Math.floor(absolute/60)).padStart(2,'0')}:${String(absolute%60).padStart(2,'0')}`}
async function save(){if(!meal.value)return;error.value='';const occurred=withOffset(occurredAt.value);const errors=validateMeal({occurredAt:occurred,mealType:mealType.value,items:items.value});if(Object.keys(errors).length){error.value=Object.values(errors)[0]!;return}busy.value=true;const input:UpdateMealInput={expectedRevision:meal.value.revision,occurredAt:occurred,mealType:mealType.value,items:items.value};try{const result=await updateMeal(id,input);await historyStore.refreshDates(result.affectedLocalDates);const date=result.affectedLocalDates.at(-1)??result.meal.localDate;await router.replace({path:'/history',query:{date}})}catch(cause){if(cause instanceof ApiError&&cause.code==='REVISION_CONFLICT'){error.value='数据已变化，已重新载入最新内容';await load()}else error.value=cause instanceof Error?cause.message:'保存失败'}finally{busy.value=false}}
onMounted(load)
</script>

<template><main class="page"><RouterLink class="back" :to="`/meals/${id}`">← 返回详情</RouterLink><h1>编辑餐食</h1><p class="notice">所有营养数据均为估算值，请按实际情况核对。</p><template v-if="meal"><label>日期时间<input v-model="occurredAt" type="datetime-local"></label><label>餐别<select v-model="mealType"><option value="breakfast">早餐</option><option value="lunch">午餐</option><option value="dinner">晚餐</option><option value="snack">加餐</option></select></label><MealItemEditor v-model:items="items"/><p v-if="error" role="alert">{{ error }}</p><button class="save" type="button" :disabled="busy" @click="save">{{ busy?'保存中…':'保存修改' }}</button></template><p v-else-if="!error" role="status">正在加载餐食…</p><p v-else role="alert">{{ error }}</p></main></template>
<style scoped>.page{width:min(100%,38rem);min-height:100svh;margin:auto;padding:1.25rem;display:grid;align-content:start;gap:1rem}.back{min-height:44px;display:flex;align-items:center;color:#155e4e}.page h1{margin:0}.notice{padding:.75rem;border-radius:.75rem;background:#fff2d8}label{display:grid;gap:.35rem;font-weight:700}input,select,.save{min-height:48px;padding:0 .75rem;border:1px solid #dbe4e0;border-radius:.75rem;font:inherit}.save{border:0;color:#fff;background:#155e4e;font-weight:700}</style>
