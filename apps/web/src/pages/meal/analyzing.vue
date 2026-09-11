<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { mealDraftStore } from '../../stores/meal-draft'
const router=useRouter();const slow=ref(false);const preview=ref('');let timer=0
async function run(){timer=window.setTimeout(()=>slow.value=true,12000);await mealDraftStore.analyze();clearTimeout(timer);if(mealDraftStore.state.status==='ready')await router.replace('/meals/confirm')}
onMounted(()=>{if(!mealDraftStore.state.image)router.replace('/meals/new');else{preview.value=window.URL.createObjectURL(mealDraftStore.state.image);run()}});onBeforeUnmount(()=>{clearTimeout(timer);if(preview.value)window.URL.revokeObjectURL(preview.value)})
</script>
<template><main class="flow"><p class="brand">FORMTALLY</p><h1 v-if="mealDraftStore.state.status!=='failed'">正在分析</h1><h1 v-else>暂时没分析出来</h1><img v-if="preview" :src="preview" alt="正在分析的餐食"><progress v-if="mealDraftStore.state.status!=='failed'"/><p v-if="slow">仍在认真识别，你也可以稍后重试。</p><template v-if="mealDraftStore.state.status==='failed'"><p role="alert">{{ mealDraftStore.state.error }}</p><button class="primary" @click="run">使用当前图片重试</button><button @click="router.replace('/meals/manual')">手动录入</button></template></main></template>
<style scoped>.flow{width:min(100%,32rem);margin:auto;min-height:100svh;padding:1.5rem;display:grid;align-content:center;gap:1rem;text-align:center}.brand{letter-spacing:.15em;color:#6a7773}h1,p{margin:0}img{width:100%;max-height:55svh;object-fit:cover;border-radius:1.2rem}progress{width:100%;accent-color:#55b98d}button{min-height:48px;border:1px solid #dbe4e0;border-radius:.9rem;font:inherit}.primary{border:0;color:#fff;background:#155e4e;font-weight:700}</style>
