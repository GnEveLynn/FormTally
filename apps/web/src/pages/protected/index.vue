<script setup lang="ts">
import { useRouter } from 'vue-router'

import { logout } from '../../api/auth'
import { sessionStore } from '../../stores/session'

defineProps<{ title: string; description: string; allowLogout?: boolean }>()
const router = useRouter()

async function signOut() {
  await logout()
  sessionStore.clear()
  await router.replace('/welcome')
}
</script>

<template>
  <main class="protected-page">
    <p class="eyebrow">FORMTALLY</p>
    <h1>{{ title }}</h1>
    <p>{{ description }}</p>
    <button v-if="allowLogout" type="button" @click="signOut">退出登录</button>
  </main>
</template>

<style scoped>
.protected-page { min-height: 100svh; display: grid; place-content: center; gap: .75rem; padding: 1.5rem; text-align: center; }
.eyebrow { margin: 0; color: var(--ft-color-text-muted); font-size: .8rem; letter-spacing: .15em; }
h1, p { margin: 0; }
button { min-width: 44px; min-height: 44px; margin-top: 1rem; border: 0; border-radius: 999px; color: white; background: var(--ft-color-text); font: inherit; font-weight: 700; }
</style>
