import type { SessionResponse } from '@formtally/api-contract/auth'
import { clearSessionToken, getSessionToken } from '../platform/storage'
import { ApiError, request } from '../services/http'

type SessionStatus = 'idle' | 'loading' | 'anonymous' | 'authenticated' | 'failed'

export interface SessionState {
  status: SessionStatus
  data: SessionResponse | null
  error: string
}

export function createSessionStore() {
  const state: SessionState = { status: 'idle', data: null, error: '' }
  let pending: Promise<void> | null = null

  function restore(force = false): Promise<void> {
    if (!getSessionToken()) {
      state.status = 'anonymous'
      state.data = null
      return Promise.resolve()
    }
    if (!force && pending) return pending
    if (!force && ['authenticated', 'failed'].includes(state.status)) return Promise.resolve()

    state.status = 'loading'
    state.error = ''
    pending = request<SessionResponse>('/v1/auth/session')
      .then((data) => {
        state.data = data
        state.status = 'authenticated'
      })
      .catch((error: unknown) => {
        state.data = null
        if (error instanceof ApiError && error.status === 401) {
          state.status = 'anonymous'
        } else {
          state.status = 'failed'
          state.error = error instanceof Error ? error.message : '服务暂时不可用，请稍后重试'
        }
      })
      .finally(() => { pending = null })
    return pending
  }

  return {
    state,
    restore,
    retry: () => restore(true),
    setSession(data: SessionResponse) {
      state.data = data
      state.status = 'authenticated'
      state.error = ''
    },
    clear() {
      clearSessionToken()
      state.data = null
      state.status = 'anonymous'
      state.error = ''
    },
    async logout() {
      try { await request<void>('/v1/auth/session', { method: 'DELETE' }) } finally {
        clearSessionToken(); state.data = null; state.status = 'anonymous'; state.error = ''
      }
    },
  }
}

export const sessionStore = createSessionStore()
