import { beforeEach, describe, expect, it, vi } from 'vitest'
import { getSessionToken } from './storage'
import { bindWeChatPhone, weChatLogin } from './auth'

describe('WeChat auth platform boundary', () => {
  const values = new Map<string, unknown>()
  let loginSequence = 0

  beforeEach(() => {
    values.clear()
    loginSequence = 0
    Object.assign(globalThis, {
      wx: {
        getAccountInfoSync: () => ({ miniProgram: { envVersion: 'develop' } }),
        getStorageSync: vi.fn((key: string) => values.get(key)),
        setStorageSync: vi.fn((key: string, value: unknown) => values.set(key, value)),
        removeStorageSync: vi.fn((key: string) => values.delete(key)),
        login: vi.fn(({ success }: Record<string, any>) => success({ code: `login-${++loginSequence}` })),
        request: vi.fn((options: Record<string, any>) => {
          const loginCode = options.data.loginCode
          options.success({ statusCode: 200, data: { bindingRequired: true, bindingTicket: `ticket-for-${loginCode}`, expiresInSeconds: 300 } })
        }),
      },
    })
  })

  it('gets a fresh one-time wx.login code for every session attempt', async () => {
    await expect(weChatLogin()).resolves.toMatchObject({ bindingTicket: 'ticket-for-login-1' })
    await expect(weChatLogin()).resolves.toMatchObject({ bindingTicket: 'ticket-for-login-2' })
    expect(globalThis.wx.login).toHaveBeenCalledTimes(2)
    expect(getSessionToken()).toBeNull()
  })

  it('stores a returned session token without logging it', async () => {
    const log = vi.spyOn(console, 'log').mockImplementation(() => undefined)
    ;(globalThis.wx.request as any).mockImplementationOnce((options: Record<string, any>) => {
      options.success({ statusCode: 200, data: { bindingRequired: false, token: 'secret-session-token', session: { expiresAt: '2026-09-15T00:00:00Z' }, user: { id: 'u1', phoneMasked: '138****5678', onboardingStatus: 'completed' }, consents: { termsVersion: '2026-09-10', privacyVersion: '2026-09-10', aiImageProcessingVersion: null, currentAiImageProcessingVersion: '2026-09-10' } } })
    })

    await expect(weChatLogin()).resolves.toMatchObject({ bindingRequired: false })
    expect(getSessionToken()).toBe('secret-session-token')
    expect(log).not.toHaveBeenCalled()
  })

  it('exchanges a fresh phone code with the binding ticket and agreements', async () => {
    ;(globalThis.wx.request as any).mockImplementationOnce((options: Record<string, any>) => {
      expect(options.data).toEqual({ bindingTicket: 'ticket-1', phoneCode: 'phone-code-1', agreements: { termsVersion: '2026-09-10', privacyVersion: '2026-09-10' } })
      options.success({ statusCode: 200, data: { token: 'bound-token', session: { expiresAt: '2026-09-15T00:00:00Z' }, user: { id: 'u1', phoneMasked: '138****5678', onboardingStatus: 'profile_required' }, consents: { termsVersion: '2026-09-10', privacyVersion: '2026-09-10', aiImageProcessingVersion: null, currentAiImageProcessingVersion: '2026-09-10' } } })
    })

    await bindWeChatPhone('ticket-1', 'phone-code-1', { termsVersion: '2026-09-10', privacyVersion: '2026-09-10' })
    expect(getSessionToken()).toBe('bound-token')
  })
})
