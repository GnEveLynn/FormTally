<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import ImagePicker from '../../components/ImagePicker.vue'
import { mealDraftStore } from '../../stores/meal-draft'
const router = useRouter()
const consent = ref(false)
const error = ref('')
function chooseAI() { mealDraftStore.state.mode = 'ai'; mealDraftStore.state.consentVersion = '2026-09-10'; router.push('/meals/analyzing') }
function manual() { mealDraftStore.state.mode = 'manual'; router.push('/meals/manual') }
</script>
<template>
  <main class="flow">
    <button class="back" type="button" @click="router.back()">‹ 返回</button>
    <h1>拍下这一餐</h1><p>尽量拍清整份食物，支持 JPEG、PNG、WebP。</p>
    <label>餐别<select v-model="mealDraftStore.state.mealType"><option value="breakfast">早餐</option><option value="lunch">午餐</option><option value="dinner">晚餐</option><option value="snack">加餐</option></select></label>
    <ImagePicker :occurred-at="mealDraftStore.state.occurredAt" :meal-type="mealDraftStore.state.mealType" @selected="mealDraftStore.selectImage" @error="error = $event" />
    <p v-if="error" role="alert" class="error">{{ error }}</p>
    <button class="primary" :disabled="!mealDraftStore.state.image" @click="consent = true">确认图片</button>
    <section v-if="consent" class="consent" role="dialog" aria-label="AI 图片处理确认"><h2>AI 图片处理确认</h2><p>这张餐食图片将发送给第三方 AI 服务进行识别。图片识别结果仅为估算。</p><button class="primary" @click="chooseAI">同意并开始分析</button><button @click="manual">不同意，手动录入</button></section>
  </main>
</template>
<style scoped>.flow{width:min(100%,32rem);margin:auto;min-height:100svh;padding:1.25rem;display:grid;align-content:start;gap:1rem}h1,p{margin:0}label{display:grid;gap:.4rem}select,button{min-height:48px;border:1px solid #dbe4e0;border-radius:.9rem;padding:0 .9rem;font:inherit}.primary{border:0;color:#fff;background:#155e4e;font-weight:700}.back{justify-self:start;border:0;background:none}.consent{display:grid;gap:.8rem;padding:1rem;border-radius:1rem;background:#eef7f3}.error{color:#a22b2b}</style>
