import type { SessionResponse } from '@formtally/api-contract/auth'
import { reactive } from 'vue'

import * as authApi from '../api/auth'
import { ApiError } from '../api/http'

type SessionStatus = 'idle' | 'loading' | 'anonymous' | 'authenticated' | 'failed'

export interface SessionState {
  status: SessionStatus
  data: SessionResponse | null
  error: string
}

export interface SessionClient {
  getSession(): Promise<SessionResponse>
}

export function createSessionStore(client: SessionClient = authApi) {
  const state = reactive<SessionState>({ status: 'idle', data: null, error: '' })
  let pending: Promise<void> | null = null

  const restore = (force = false): Promise<void> => {
    if (!force && pending) return pending
    if (!force && ['anonymous', 'authenticated', 'failed'].includes(state.status)) return Promise.resolve()

    state.status = 'loading'
    state.error = ''
    pending = client
      .getSession()
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
      .finally(() => {
        pending = null
      })
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
      state.data = null
      state.status = 'anonymous'
      state.error = ''
    },
  }
}

export const sessionStore = createSessionStore()
