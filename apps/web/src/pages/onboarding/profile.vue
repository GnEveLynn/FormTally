<script setup lang="ts">
import { validateProfile } from '@formtally/domain/profile-validation'
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { saveProfile } from '../../api/profile'
import { onboardingStore } from '../../stores/onboarding'
import { sessionStore } from '../../stores/session'
const router = useRouter(); const busy = ref(false); const errors = ref<Record<string,string>>({}); const generalError = ref('')
async function submit() {
  errors.value = validateProfile(onboardingStore.profile); if (Object.keys(errors.value).length) return
  busy.value = true; generalError.value = ''
  try { await saveProfile(onboardingStore.profile); if (sessionStore.state.data) sessionStore.state.data.user.onboardingStatus = 'goal_required'; await router.push('/goals') }
  catch (error) { generalError.value = error instanceof Error ? error.message : '保存失败，请重试' } finally { busy.value = false }
}
</script>
<template><main class="page"><p class="step">第 1 步，共 3 步</p><h1>完善身体资料</h1><p>用于估算每日营养目标，你可以之后修改。</p>
<form @submit.prevent="submit">
<label>生理性别<select v-model="onboardingStore.profile.biologicalSex"><option value="male">男性</option><option value="female">女性</option></select></label>
<label>出生日期<input v-model="onboardingStore.profile.birthDate" type="date"></label><p v-if="errors.birthDate" class="error">{{ errors.birthDate }}</p>
<label>身高（cm）<input v-model.number="onboardingStore.profile.heightCm" type="number" min="100" max="250" step="0.1"></label><p v-if="errors.heightCm" class="error">{{ errors.heightCm }}</p>
<label>体重（kg）<input v-model.number="onboardingStore.profile.weightKg" type="number" min="25" max="350" step="0.1"></label><p v-if="errors.weightKg" class="error">{{ errors.weightKg }}</p>
<label>活动水平<select v-model="onboardingStore.profile.activityLevel"><option value="sedentary">久坐</option><option value="light">轻度</option><option value="moderate">中度</option><option value="high">高度</option><option value="very_high">极高</option></select></label>
<fieldset><legend>以下情况适用时请选择</legend><label><input v-model="onboardingStore.profile.healthContext.pregnant" type="checkbox">孕期</label><label><input v-model="onboardingStore.profile.healthContext.breastfeeding" type="checkbox">哺乳期</label><label><input v-model="onboardingStore.profile.healthContext.clinicalDietRequired" type="checkbox">需要疾病膳食管理</label></fieldset>
<p v-if="generalError" class="error" role="alert">{{ generalError }}</p><button type="submit" :disabled="busy">下一步</button></form></main></template>
<style scoped>.page{width:min(100%,32rem);margin:auto;padding:1.5rem}.step,p{color:var(--ft-color-text-muted)}form,label{display:grid;gap:.45rem}form{gap:1rem}input,select,button{min-height:48px;border:1px solid #cbd5d1;border-radius:.8rem;padding:0 .8rem;font:inherit}fieldset label{display:flex;align-items:center;min-height:44px}fieldset input{min-height:auto}.error{color:#a22b2b;margin:0}button{border:0;background:var(--ft-color-text);color:#fff;font-weight:700}</style>
