<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { logout } from '../../api/auth'
import { getProfile } from '../../api/profile'
import { getGoals } from '../../api/goals'
import { sessionStore } from '../../stores/session'

const router=useRouter()
const phone=computed(()=>sessionStore.state.data?.user.phoneMasked??'')
const profileSummary=ref('正在加载…'),goalSummary=ref('正在加载…')
onMounted(async()=>{
  try {
    const [profileResult, goalsResult]=await Promise.all([getProfile(),getGoals()])
    profileSummary.value=profileResult.profile?`${profileResult.profile.weightKg} kg · ${profileResult.profile.heightCm} cm · ${profileResult.profile.timezone}`:'尚未设置'
    const target=goalsResult.pendingTarget??goalsResult.activeTarget
    goalSummary.value=target?`${target.target.energyKcal} 千卡 · 蛋白质 ${target.target.proteinGrams}g`:'尚未设置'
  } catch { profileSummary.value='暂时无法加载';goalSummary.value='暂时无法加载' }
})
async function signOut(){await logout();sessionStore.clear();await router.replace('/welcome')}
</script>
<template><main class="page"><header><div><p>FORMTALLY</p><h1>我的</h1></div><nav><RouterLink to="/today">今天</RouterLink><RouterLink to="/history">历史</RouterLink></nav></header><section class="card"><h2>账户</h2><p>{{ phone }}</p></section><nav class="menu"><RouterLink to="/me/profile"><span><strong>身体资料与时区</strong><small>{{ profileSummary }}</small></span><b>›</b></RouterLink><RouterLink to="/me/goals"><span><strong>每日目标</strong><small>{{ goalSummary }}</small></span><b>›</b></RouterLink></nav><section class="card"><h2>协议与说明</h2><p>用户协议 · 版本 2026-09-10</p><p>隐私政策 · 版本 2026-09-10</p><p>AI 图片处理说明 · 版本 2026-09-10</p></section><button class="logout" type="button" @click="signOut">退出登录</button><RouterLink class="danger" to="/me/delete-account">删除账户与个人数据</RouterLink></main></template>
<style scoped>.page{width:min(100%,36rem);min-height:100svh;margin:auto;padding:1.25rem;display:grid;align-content:start;gap:1rem}header{display:flex;justify-content:space-between;align-items:center}header p,h1{margin:0}header p{color:#6a7773;font-size:.78rem;letter-spacing:.14em}header nav{display:flex;gap:.8rem}a{color:#155e4e}.card,.menu{padding:1rem;border:1px solid #dbe5e1;border-radius:1rem;background:#fff}.card h2,.card p{margin:.25rem 0}.menu{display:grid;padding:0}.menu a{min-height:64px;padding:0 1rem;display:flex;align-items:center;justify-content:space-between;text-decoration:none}.menu span{display:grid;gap:.2rem}.menu small{color:#6a7773}.menu a+a{border-top:1px solid #e5ebe8}.logout,.danger{min-height:48px;border-radius:.8rem;font:inherit;font-weight:700}.logout{border:0;color:#fff;background:#155e4e}.danger{display:grid;place-items:center;border:1px solid #a22b2b;color:#a22b2b;text-decoration:none}</style>
