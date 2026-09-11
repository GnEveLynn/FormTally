<script setup lang="ts">
import { onMounted } from 'vue'
import { useRouter } from 'vue-router'

import { bootRoute } from '../../router/boot'
import { sessionStore } from '../../stores/session'

const router = useRouter()

async function start(retry = false) {
  if (retry) await sessionStore.retry()
  const target = await bootRoute(sessionStore)
  if (target) await router.replace(target)
}

onMounted(() => start())
</script>

<template>
  <main class="start-page" aria-live="polite">
    <template v-if="sessionStore.state.status === 'failed'">
      <h1>暂时无法连接</h1>
      <p>{{ sessionStore.state.error }}</p>
      <button type="button" @click="start(true)">重试</button>
    </template>
    <template v-else>
      <h1>FormTally</h1>
      <p>正在恢复你的登录状态…</p>
    </template>
  </main>
</template>

<style scoped>
.start-page { min-height: 100svh; display: grid; place-content: center; gap: var(--ft-space-sm); padding: var(--ft-space-lg); text-align: center; }
h1, p { margin: 0; }
p { color: var(--ft-color-text-muted); line-height: 1.6; }
button { min-width: 44px; min-height: 44px; border: 0; border-radius: 999px; color: white; background: var(--ft-color-text); font: inherit; font-weight: 700; }
</style>
