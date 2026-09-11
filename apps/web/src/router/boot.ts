import type { OnboardingStatus } from '@formtally/api-contract/auth'

import type { createSessionStore } from '../stores/session'

type SessionStore = ReturnType<typeof createSessionStore>

export function onboardingRoute(status: OnboardingStatus): string {
  return {
    profile_required: '/profile',
    goal_required: '/goals',
    completed: '/today',
  }[status]
}

export async function bootRoute(store: SessionStore): Promise<string | null> {
  await store.restore()
  if (store.state.status === 'anonymous') return '/welcome'
  if (store.state.status === 'authenticated' && store.state.data) {
    return onboardingRoute(store.state.data.user.onboardingStatus)
  }
  return null
}

export async function guardRoute(path: string, requiresAuth: boolean, store: SessionStore): Promise<true | string> {
  if (!requiresAuth) return true
  const target = await bootRoute(store)
  if (store.state.status !== 'authenticated') return target ?? '/'
  if (target === '/today') return true
  return target === path || (target === '/goals' && path.startsWith('/goals/')) ? true : (target ?? '/')
}
