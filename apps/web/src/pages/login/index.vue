<script setup lang="ts">
import type { SessionResponse } from '@formtally/api-contract/auth'
import { normalizeChinaPhone } from '@formtally/domain/phone'
import { onBeforeUnmount, ref } from 'vue'
import { useRouter } from 'vue-router'

import { login, requestLoginCode } from '../../api/auth'
import { ApiError } from '../../api/http'
import { onboardingRoute } from '../../router/boot'
import { sessionStore } from '../../stores/session'

const AGREEMENT_VERSION = '2026-09-10'
const router = useRouter()
const phone = ref('')
const code = ref('')
const agreements = ref(false)
const verificationRequestId = ref('')
const remainingSeconds = ref(0)
const busy = ref(false)
const generalError = ref('')
const fieldErrors = ref<Record<string, string>>({})
let timer: ReturnType<typeof setInterval> | undefined

onBeforeUnmount(() => clearInterval(timer))

function showError(error: unknown) {
  if (error instanceof ApiError) {
    const fallbackField = {
      PHONE_UNSUPPORTED: 'phone',
      VERIFICATION_CODE_INVALID: 'code',
      VERIFICATION_CODE_EXPIRED: 'code',
      AGREEMENT_VERSION_OUTDATED: 'agreements',
    }[error.code]
    fieldErrors.value = Object.keys(error.fieldErrors).length
      ? error.fieldErrors
      : fallbackField
        ? { [fallbackField]: error.message }
        : {}
    generalError.value = Object.keys(fieldErrors.value).length ? '' : error.message
  } else {
    generalError.value = '网络连接失败，请重试'
  }
}

async function sendCode() {
  fieldErrors.value = {}
  generalError.value = ''
  const normalized = normalizeChinaPhone(phone.value)
  if (!normalized) {
    fieldErrors.value.phone = '请输入正确的中国大陆手机号'
    return
  }

  busy.value = true
  try {
    const result = await requestLoginCode(normalized)
    phone.value = normalized
    verificationRequestId.value = result.verification.requestId
    remainingSeconds.value = result.verification.retryAfterSeconds
    clearInterval(timer)
    timer = setInterval(() => {
      remainingSeconds.value = Math.max(0, remainingSeconds.value - 1)
      if (remainingSeconds.value === 0) clearInterval(timer)
    }, 1000)
  } catch (error) {
    showError(error)
  } finally {
    busy.value = false
  }
}

async function submit() {
  fieldErrors.value = {}
  generalError.value = ''
  if (!agreements.value) {
    fieldErrors.value.agreements = '请先阅读并同意服务协议与隐私政策'
    return
  }
  const normalized = normalizeChinaPhone(phone.value)
  if (!normalized) {
    fieldErrors.value.phone = '请输入正确的中国大陆手机号'
    return
  }
  if (!verificationRequestId.value || !/^\d{6}$/.test(code.value)) {
    fieldErrors.value.code = verificationRequestId.value ? '请输入 6 位验证码' : '请先获取验证码'
    return
  }

  busy.value = true
  try {
    const result: SessionResponse = await login({
      phone: normalized,
      code: code.value,
      verificationRequestId: verificationRequestId.value,
      agreements: { termsVersion: AGREEMENT_VERSION, privacyVersion: AGREEMENT_VERSION },
    })
    sessionStore.setSession(result)
    await router.replace(onboardingRoute(result.user.onboardingStatus))
  } catch (error) {
    showError(error)
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <main class="login-page">
    <RouterLink class="back" to="/welcome" aria-label="返回欢迎页">←</RouterLink>
    <p class="eyebrow">安全登录</p>
    <h1>欢迎回来</h1>
    <p class="intro">首次登录会自动为你创建账户。</p>

    <form novalidate @submit.prevent="submit">
      <label for="phone">手机号</label>
      <div class="phone-row">
        <span>+86</span>
        <input
          id="phone"
          v-model="phone"
          name="phone"
          inputmode="tel"
          autocomplete="tel-national"
          placeholder="138 1234 5678"
          :aria-invalid="Boolean(fieldErrors.phone)"
          :aria-describedby="fieldErrors.phone ? 'phone-error' : undefined"
        >
      </div>
      <p v-if="fieldErrors.phone" id="phone-error" class="field-error">{{ fieldErrors.phone }}</p>

      <label for="code">验证码</label>
      <div class="code-row">
        <input
          id="code"
          v-model="code"
          name="code"
          inputmode="numeric"
          autocomplete="one-time-code"
          maxlength="6"
          placeholder="6 位验证码"
          :aria-invalid="Boolean(fieldErrors.code)"
          :aria-describedby="fieldErrors.code ? 'code-error' : undefined"
        >
        <button
          type="button"
          data-action="send-code"
          :disabled="busy || remainingSeconds > 0"
          @click="sendCode"
        >
          {{ remainingSeconds > 0 ? `${remainingSeconds} 秒后重发` : '获取验证码' }}
        </button>
      </div>
      <p v-if="fieldErrors.code" id="code-error" class="field-error">{{ fieldErrors.code }}</p>

      <label class="agreement">
        <input
          v-model="agreements"
          name="agreements"
          type="checkbox"
          :aria-invalid="Boolean(fieldErrors.agreements)"
          :aria-describedby="fieldErrors.agreements ? 'agreements-error' : undefined"
        >
        <span>我已阅读并同意服务协议与隐私政策</span>
      </label>
      <p v-if="fieldErrors.agreements" id="agreements-error" class="field-error">{{ fieldErrors.agreements }}</p>
      <p v-if="generalError" class="general-error" role="alert">{{ generalError }}</p>

      <button class="submit" type="submit" :disabled="busy">{{ busy ? '处理中…' : '登录并继续' }}</button>
    </form>
  </main>
</template>

<style scoped>
.login-page { width: min(100%, 30rem); min-height: 100svh; margin: auto; padding: 1.5rem; }
.back { display: grid; place-items: center; width: 44px; height: 44px; color: var(--ft-color-text); text-decoration: none; font-size: 1.5rem; }
.eyebrow { margin: 3rem 0 .5rem; color: var(--ft-color-text-muted); font-size: .8rem; font-weight: 700; letter-spacing: .14em; }
h1 { margin: 0; font-size: var(--ft-font-size-title); }
.intro { margin: .5rem 0 2rem; color: var(--ft-color-text-muted); }
form { display: grid; gap: .6rem; }
label { margin-top: .75rem; font-weight: 700; }
.phone-row, .code-row { display: flex; align-items: center; gap: .75rem; min-height: 52px; border: 1px solid #cbd5d1; border-radius: .9rem; background: white; padding: 0 .9rem; }
.phone-row span { padding-right: .75rem; border-right: 1px solid #dce3e0; }
input { min-width: 0; min-height: 44px; flex: 1; border: 0; outline: none; font: inherit; background: transparent; }
button { min-width: 44px; min-height: 44px; border: 0; font: inherit; font-weight: 700; cursor: pointer; }
.code-row button { color: var(--ft-color-text); background: transparent; white-space: nowrap; }
button:disabled { cursor: wait; opacity: .55; }
.agreement { display: flex; align-items: center; gap: .65rem; min-height: 44px; font-size: .9rem; font-weight: 400; }
.agreement input { flex: 0 0 44px; width: 44px; height: 44px; margin: 0; }
.field-error, .general-error { margin: 0; color: #a22b2b; font-size: .85rem; }
.submit { width: 100%; min-height: 52px; margin-top: 1rem; border-radius: 999px; color: white; background: var(--ft-color-text); }
</style>
