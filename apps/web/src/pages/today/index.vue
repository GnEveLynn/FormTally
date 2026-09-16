<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import ImagePicker from '../../components/ImagePicker.vue'
import { mealDraftStore } from '../../stores/meal-draft'
import { todayStore } from '../../stores/today'

const router = useRouter()
const baseDate = new Date()
const localDate = (date: Date) => { const copy = new Date(date); copy.setMinutes(copy.getMinutes() - copy.getTimezoneOffset()); return copy.toISOString().slice(0, 10) }
const shiftedDate = (offset: number) => { const date = new Date(baseDate); date.setDate(date.getDate() + offset); return localDate(date) }
const displayDate = (value: string) => { const [, month, day] = value.split('-'); return `${Number(month)}月${Number(day)}日` }
const selectedDate = ref(localDate(baseDate))
const recordOpen = ref(false)
const error = ref('')
const tabs = [{ key: 'yesterday', label: '昨日', date: shiftedDate(-1) }, { key: 'today', label: '今日', date: shiftedDate(0) }, { key: 'tomorrow', label: '明日', date: shiftedDate(1) }]
const mealTypes = [{ value: 'breakfast', label: '早餐', icon: '☀' }, { value: 'lunch', label: '午餐', icon: '♨' }, { value: 'dinner', label: '晚餐', icon: '☾' }, { value: 'snack', label: '加餐', icon: '♡' }]
const day = computed(() => todayStore.state.day)
const energy = computed(() => day.value?.progress?.energy)
const energyDifference = computed(() => energy.value?.overBy ? { label: '已超出', value: energy.value.overBy } : { label: '还可摄入', value: energy.value?.remaining ?? 0 })
const ringStyle = computed(() => ({ '--progress': `${Math.min(energy.value?.percent ?? 0, 100) * 3.6}deg` }))
const mealGroup = (type: string) => day.value?.mealGroups.find(group => group.mealType === type)
const time = (value: string) => new Intl.DateTimeFormat('zh-CN', { hour: '2-digit', minute: '2-digit' }).format(new Date(value))
async function selectDate(date: string) { selectedDate.value = date; await todayStore.load(date) }
function openRecord(type = 'lunch') { mealDraftStore.state.mealType = type; recordOpen.value = true }
function closeRecord() { recordOpen.value = false; error.value = '' }
async function analyze() { if (!mealDraftStore.state.image) { error.value = '请先拍照或选择食物照片'; return }; mealDraftStore.state.mode = 'ai'; mealDraftStore.state.consentVersion = '2026-09-10'; await router.push('/meals/analyzing') }
async function manual() { mealDraftStore.state.mode = 'manual'; await router.push('/meals/manual') }
onMounted(() => todayStore.load(selectedDate.value))
</script>

<template>
  <main class="today">
    <header><div><h1>FormTally</h1><p>饮食记录</p></div><span class="calendar">▣</span></header>
    <nav class="date-tabs" aria-label="日期切换"><button v-for="tab in tabs" :key="tab.key" :data-test="`date-${tab.key}`" :class="{ active: selectedDate === tab.date }" @click="selectDate(tab.date)"><strong>{{ tab.label }}</strong><span>{{ displayDate(tab.date) }}</span></button></nav>
    <p v-if="todayStore.state.status === 'loading'" class="notice" role="status">正在加载记录…</p>
    <p v-else-if="todayStore.state.status === 'error'" class="notice error" role="alert">{{ todayStore.state.error }}</p>
    <template v-else-if="day">
      <section v-if="day.progress" class="summary-card"><div class="ring" :style="ringStyle"><div><strong>{{ day.progress.energy.consumed.toLocaleString() }}</strong><span>/ {{ day.progress.energy.target.toLocaleString() }}</span><small>千卡</small></div></div><div class="remaining"><span>{{ energyDifference.label }}</span><strong>{{ energyDifference.value.toLocaleString() }} <small>千卡</small></strong><p>保持均衡饮食，遇见更好的自己。</p></div></section>
      <section v-if="day.progress" class="nutrients"><article><span class="protein">◉</span><strong>蛋白质</strong><b>{{ day.progress.protein.consumed }} <small>/ {{ day.progress.protein.target }}g</small></b></article><article><span class="carbs">♨</span><strong>碳水</strong><b>{{ day.progress.carb.consumed }} <small>/ {{ day.progress.carb.target }}g</small></b></article><article><span class="fat">◯</span><strong>脂肪</strong><b>{{ day.progress.fat.consumed }} <small>/ {{ day.progress.fat.target }}g</small></b></article></section>
      <section class="meal-grid"><article v-for="type in mealTypes" :key="type.value" class="meal-card" @click="openRecord(type.value)"><header><span :class="type.value">{{ type.icon }}</span><strong>{{ type.label }}</strong><b>›</b></header><template v-if="mealGroup(type.value)?.meals.length"><a v-for="meal in mealGroup(type.value)?.meals" :key="meal.id" :href="`/meals/${meal.id}`" @click.stop><span>{{ time(meal.occurredAt) }}</span><strong>{{ meal.totals.energyKcal }} 千卡</strong></a></template><p v-else>还没有记录<br>快去记录吧～</p></article></section>
      <p v-if="todayStore.state.status === 'empty'" class="empty-title">今天还没有记录</p>
    </template>
    <nav class="bottom-nav"><a class="active" href="/today">⌂<small>今日</small></a><button data-test="open-record" @click="openRecord()">▣ 记录饮食</button><a href="/history">▤<small>历史</small></a><a href="/me">♙<small>我的</small></a></nav>
    <div v-if="recordOpen" class="modal-backdrop" @click.self="closeRecord"><section class="record-dialog" role="dialog" aria-modal="true" aria-label="记录饮食"><div class="handle"/><header><h2>记录饮食</h2><button aria-label="关闭" @click="closeRecord">×</button></header><label class="meal-type">餐别<select v-model="mealDraftStore.state.mealType"><option v-for="type in mealTypes" :key="type.value" :value="type.value">{{ type.label }}</option></select></label><ImagePicker :occurred-at="mealDraftStore.state.occurredAt" :meal-type="mealDraftStore.state.mealType" @selected="mealDraftStore.selectImage" @error="error = $event" /><label class="description">补充描述（选填）<textarea v-model="mealDraftStore.state.description" maxlength="500" placeholder="例如：鸡胸肉、米饭和西兰花，少油、酱汁另放" /></label><p class="hint">AI 会结合照片，自行判断食物、份量、做法和备注。</p><p v-if="error" class="error" role="alert">{{ error }}</p><footer><button class="secondary" @click="manual">手动记录</button><button class="primary" @click="analyze">确认并生成估算</button></footer></section></div>
  </main>
</template>

<style scoped>
.today{width:min(100%,42rem);margin:auto;min-height:100svh;padding:1.5rem 1rem 7rem;background:radial-gradient(circle at 35% 0,#eafbf5 0,transparent 25rem),#f7faf8;color:#173c35}.today>header{display:flex;justify-content:space-between;align-items:center}.today h1,.today header p{margin:0}.calendar{font-size:1.6rem}.date-tabs,.nutrients,.meal-grid{display:grid;gap:.6rem}.date-tabs{grid-template-columns:repeat(3,1fr);margin:1rem 0;padding:.35rem;border:1px solid #e4ece8;border-radius:1.1rem;background:#fff}.date-tabs button{display:grid;gap:.15rem;min-height:3.5rem;border:0;border-radius:.85rem;background:transparent;color:#71807b}.date-tabs .active{background:#e2f8ec;color:#199b5c}.summary-card{display:grid;grid-template-columns:1fr 1fr;align-items:center;gap:1rem;padding:1.1rem;border:1px solid #e2ebe6;border-radius:1.2rem;background:#fff}.ring{aspect-ratio:1;border-radius:50%;display:grid;place-items:center;background:conic-gradient(#35bf76 var(--progress),#dff5e8 0);position:relative}.ring:before{content:"";position:absolute;inset:12%;border-radius:50%;background:#fff}.ring div{z-index:1;text-align:center}.ring small{display:block}.remaining{display:grid;gap:.3rem}.remaining>strong{font-size:1.8rem}.remaining p,.empty-title{color:#71807a;font-size:.82rem}.nutrients{grid-template-columns:repeat(3,1fr);margin-top:.7rem}.nutrients article{display:grid;grid-template-columns:auto 1fr;gap:.4rem;padding:.8rem;border:1px solid #e4ebe8;border-radius:1rem;background:#fff}.nutrients b{grid-column:1/-1}.nutrients small{color:#7c8884}.protein{color:#ff6b57}.carbs{color:#f1a21d}.fat{color:#2c8ee8}.meal-grid{grid-template-columns:repeat(2,1fr);margin-top:.7rem}.meal-card{min-height:8rem;padding:.9rem;border:1px solid #e3ebe7;border-radius:1rem;background:#fff}.meal-card header,.meal-card a{display:flex;justify-content:space-between;gap:.5rem}.meal-card p{color:#84908b}.meal-card a{margin-top:.6rem;color:inherit;text-decoration:none}.bottom-nav{position:fixed;left:50%;bottom:1rem;transform:translateX(-50%);width:min(calc(100% - 2rem),40rem);display:grid;grid-template-columns:1fr 1.7fr 1fr 1fr;align-items:center;padding:.55rem;border-radius:1.4rem;background:#fffffff2;box-shadow:0 12px 35px #183c3520;z-index:5}.bottom-nav a{display:grid;justify-items:center;color:#71807a;text-decoration:none}.bottom-nav button{min-height:3.2rem;border:0;border-radius:999px;background:#38c676;color:#fff;font-weight:700}.modal-backdrop{position:fixed;inset:0;display:grid;align-items:end;background:#10282099;z-index:10}.record-dialog{width:min(100%,42rem);max-height:92svh;overflow:auto;margin:auto;padding:.5rem 1.15rem 1.2rem;border-radius:1.5rem 1.5rem 0 0;background:#fff}.handle{width:2.6rem;height:.28rem;margin:auto;border-radius:99px;background:#d3dad7}.record-dialog>header{display:flex;justify-content:space-between;align-items:center}.record-dialog>header button{width:2.3rem;height:2.3rem;border:0;border-radius:50%}.meal-type,.description{display:grid;gap:.5rem;margin:.6rem 0}.meal-type{grid-template-columns:auto 1fr;padding:.6rem;background:#effaf4;border-radius:.8rem}.meal-type select{border:0;background:transparent}.description textarea{min-height:4.5rem;padding:.8rem;border:1px solid #d5dfda;border-radius:.8rem;font:inherit}.hint{color:#71807a;font-size:.78rem}.record-dialog footer{display:grid;grid-template-columns:1fr 1.3fr;gap:.65rem}.record-dialog footer button{min-height:3rem;border-radius:999px;font-weight:700}.secondary{border:1px solid #dde5e1;background:#fff}.primary{border:0;background:#38c676;color:#fff}.error{color:#a22b2b}
</style>
