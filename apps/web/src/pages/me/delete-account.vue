<script setup lang="ts">
import { normalizeChinaPhone } from '@formtally/domain/phone'
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { deleteAccount } from '../../api/account'
import { requestDeleteCode } from '../../api/auth'
import { sessionStore } from '../../stores/session'

const router=useRouter(),phone=ref(''),code=ref(''),confirmation=ref(''),requestId=ref(''),busy=ref(false),error=ref('')
async function send(){const normalized=normalizeChinaPhone(phone.value);if(!normalized){error.value='请输入正确的完整手机号';return}busy.value=true;error.value='';try{requestId.value=(await requestDeleteCode(normalized)).verification.requestId;phone.value=normalized}catch(cause){error.value=cause instanceof Error?cause.message:'发送失败'}finally{busy.value=false}}
async function remove(){if(!requestId.value||!/^[0-9]{6}$/.test(code.value)){error.value='请先获取并输入 6 位验证码';return}if(confirmation.value!=='DELETE'){error.value='请输入 DELETE 确认删除';return}busy.value=true;error.value='';try{await deleteAccount({code:code.value,verificationRequestId:requestId.value,confirmation:'DELETE'});sessionStore.clear();await router.replace('/welcome')}catch(cause){error.value=cause instanceof Error?cause.message:'删除失败'}finally{busy.value=false}}
</script>
<template><main class="page"><RouterLink to="/me">← 返回我的</RouterLink><h1>删除账户与个人数据</h1><section class="warning"><h2>此操作不可恢复</h2><p>账户、资料、目标、饮食记录和图片将进入删除流程，所有登录会话立即失效。</p></section><label>完整手机号<input v-model="phone" inputmode="tel" autocomplete="tel"></label><button type="button" :disabled="busy" @click="send">发送删号验证码</button><label>验证码<input v-model="code" inputmode="numeric" maxlength="6" autocomplete="one-time-code"></label><label>确认文字<input v-model="confirmation" autocomplete="off" placeholder="输入 DELETE"></label><p v-if="error" role="alert">{{ error }}</p><button class="danger" type="button" :disabled="busy" @click="remove">{{ busy?'处理中…':'永久删除账户' }}</button></main></template>
<style scoped>.page{width:min(100%,32rem);min-height:100svh;margin:auto;padding:1.25rem;display:grid;align-content:start;gap:1rem}.page>a{min-height:44px;display:flex;align-items:center;color:#155e4e}.page h1{margin:0}.warning{padding:1rem;border-radius:1rem;color:#7d2020;background:#fff0f0}.warning h2,.warning p{margin:.25rem 0}label{display:grid;gap:.4rem;font-weight:700}input,button{min-height:48px;padding:0 .75rem;border:1px solid #cbd5d1;border-radius:.75rem;font:inherit}button{font-weight:700}.danger{border:0;color:#fff;background:#a22b2b}</style>
