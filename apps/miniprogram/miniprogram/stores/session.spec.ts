import type { SessionResponse } from '@formtally/api-contract/auth'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { setSessionToken } from '../platform/storage'
import { createSessionStore } from './session'

const session: SessionResponse = {
  session: { expiresAt: '2026-09-15T00:00:00Z' },
  user: { id: 'u1', phoneMasked: '138****5678', onboardingStatus: 'completed' },
  consents: { termsVersion: '2026-09-10', privacyVersion: '2026-09-10', aiImageProcessingVersion: null, currentAiImageProcessingVersion: '2026-09-10' },
}

describe('session restoration', () => {
  const values = new Map<string, unknown>()

  beforeEach(() => {
    values.clear()
    Object.assign(globalThis, {
      wx: {
        getAccountInfoSync: () => ({ miniProgram: { envVersion: 'develop' } }),
        getStorageSync: vi.fn((key: string) => values.get(key)),
        setStorageSync: vi.fn((key: string, value: unknown) => values.set(key, value)),
        removeStorageSync: vi.fn((key: string) => values.delete(key)),
        reLaunch: vi.fn(),
        request: vi.fn((options: Record<string, any>) => options.success({ statusCode: 200, data: session })),
      },
    })
  })

  it('restores a valid persisted token through GET /v1/auth/session', async () => {
    setSessionToken('valid-token')
    const store = createSessionStore()
    await store.restore()
    expect(globalThis.wx.request).toHaveBeenCalledWith(expect.objectContaining({ url: 'https://develop-api.formtally.invalid/v1/auth/session', method: 'GET' }))
    expect(store.state).toMatchObject({ status: 'authenticated', data: session, error: '' })
  })

  it('clears a rejected token and becomes anonymous on 401', async () => {
    setSessionToken('expired-token')
    ;(globalThis.wx.request as any).mockImplementationOnce((options: Record<string, any>) => options.success({ statusCode: 401, data: { error: { code: 'UNAUTHENTICATED', message: '请先登录', requestId: 'req-4' } } }))
    const store = createSessionStore()
    await store.restore()
    expect(values.has('formtally.sessionToken')).toBe(false)
    expect(store.state.status).toBe('anonymous')
  })

  it('keeps the token and exposes a retryable failure after a network error', async () => {
    setSessionToken('offline-token')
    ;(globalThis.wx.request as any).mockImplementationOnce((options: Record<string, any>) => options.fail({ errMsg: 'request:fail timeout' }))
    const store = createSessionStore()
    await store.restore()
    expect(values.get('formtally.sessionToken')).toBe('offline-token')
    expect(store.state.status).toBe('failed')
    expect(store.state.error).toBe('网络连接失败，请重试')
  })
})
