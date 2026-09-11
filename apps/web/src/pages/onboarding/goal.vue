<script setup lang="ts">
import { automaticGoalEligible } from '@formtally/domain/profile-validation'
import { ref } from 'vue'; import { useRouter } from 'vue-router'
import { previewGoal } from '../../api/goals'; import { onboardingStore } from '../../stores/onboarding'
const router=useRouter(); const busy=ref(false); const error=ref('')
if (!automaticGoalEligible(onboardingStore.profile.healthContext)) onboardingStore.goal.mode='manual'
async function submit(){busy.value=true;error.value='';try{const result=await previewGoal(onboardingStore.goalInput());onboardingStore.preview.value=result.preview;await router.push('/goals/result')}catch(e){error.value=e instanceof Error?e.message:'预览失败，请重试'}finally{busy.value=false}}
</script>
<template><main class="page"><p>第 2 步，共 3 步</p><h1>设置营养目标</h1><form @submit.prevent="submit"><label>目标模式<select v-model="onboardingStore.goal.mode"><option value="automatic" :disabled="!automaticGoalEligible(onboardingStore.profile.healthContext)">自动估算</option><option value="manual">手动设置</option></select></label>
<template v-if="onboardingStore.goal.mode==='automatic'"><label>当前目标<select v-model="onboardingStore.goal.automatic.objective"><option value="fat_loss">减脂</option><option value="maintain">保持</option><option value="muscle_gain">增肌</option></select></label><label v-if="onboardingStore.goal.automatic.objective!=='maintain'">速度<select v-model="onboardingStore.goal.automatic.pace"><option value="slow">慢速</option><option value="standard">标准</option><option value="fast">快速</option></select></label></template>
<template v-else><p>特殊健康状态建议咨询专业人士，当前仅提供手动目标。</p><label>热量（kcal）<input v-model.number="onboardingStore.goal.manual.target.energyKcal" type="number"></label><label>蛋白质（g）<input v-model.number="onboardingStore.goal.manual.target.proteinGrams" type="number"></label><label>碳水（g）<input v-model.number="onboardingStore.goal.manual.target.carbGrams" type="number"></label><label>脂肪（g）<input v-model.number="onboardingStore.goal.manual.target.fatGrams" type="number"></label></template>
<p v-if="error" class="error" role="alert">{{ error }}</p><button type="submit" :disabled="busy">预览目标</button></form></main></template>
<style scoped>.page{width:min(100%,32rem);margin:auto;padding:1.5rem}p{color:var(--ft-color-text-muted)}form,label{display:grid;gap:.5rem}form{gap:1rem}input,select,button{min-height:48px;border:1px solid #cbd5d1;border-radius:.8rem;padding:0 .8rem;font:inherit}.error{color:#a22b2b}button{border:0;background:var(--ft-color-text);color:#fff;font-weight:700}</style>
