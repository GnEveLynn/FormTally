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
          options.success({ statusCode: 200, data: { token: `token-for-${loginCode}`, session: { expiresAt: '2026-09-15T00:00:00Z' }, user: { id: 'u1', phoneMasked: null, onboardingStatus: 'profile_required' }, consents: { termsVersion: '2026-09-10', privacyVersion: '2026-09-10', aiImageProcessingVersion: null, currentAiImageProcessingVersion: '2026-09-10' } } })
        }),
      },
    })
  })

  it('gets a fresh one-time wx.login code for every session attempt', async () => {
    await expect(weChatLogin({ termsVersion: '2026-09-10', privacyVersion: '2026-09-10' })).resolves.toMatchObject({ token: 'token-for-login-1' })
    await expect(weChatLogin({ termsVersion: '2026-09-10', privacyVersion: '2026-09-10' })).resolves.toMatchObject({ token: 'token-for-login-2' })
    expect(globalThis.wx.login).toHaveBeenCalledTimes(2)
    expect(getSessionToken()).toBe('token-for-login-2')
  })

  it('identifies wx.login failures before sending a backend request', async () => {
    ;(globalThis.wx.login as any).mockImplementationOnce(({ fail }: Record<string, any>) => fail({ errMsg: 'login:fail system error' }))

    await expect(weChatLogin({ termsVersion: '2026-09-10', privacyVersion: '2026-09-10' }))
      .rejects.toThrow('wx.login 失败：login:fail system error')
    expect(globalThis.wx.request).not.toHaveBeenCalled()
  })

  it('identifies wx.request failures after wx.login succeeds', async () => {
    ;(globalThis.wx.request as any).mockImplementationOnce(({ fail }: Record<string, any>) => fail({ errMsg: 'request:fail timeout' }))

    await expect(weChatLogin({ termsVersion: '2026-09-10', privacyVersion: '2026-09-10' }))
      .rejects.toThrow('wx.request 失败：request:fail timeout')
    expect(globalThis.wx.login).toHaveBeenCalledOnce()
    expect(globalThis.wx.request).toHaveBeenCalledOnce()
  })

  it('stores a returned session token without logging it', async () => {
    const log = vi.spyOn(console, 'log').mockImplementation(() => undefined)
    ;(globalThis.wx.request as any).mockImplementationOnce((options: Record<string, any>) => {
      expect(options.data).toEqual({ loginCode: 'login-1', agreements: { termsVersion: '2026-09-10', privacyVersion: '2026-09-10' } })
      options.success({ statusCode: 200, data: { token: 'secret-session-token', session: { expiresAt: '2026-09-15T00:00:00Z' }, user: { id: 'u1', phoneMasked: null, onboardingStatus: 'completed' }, consents: { termsVersion: '2026-09-10', privacyVersion: '2026-09-10', aiImageProcessingVersion: null, currentAiImageProcessingVersion: '2026-09-10' } } })
    })

    await expect(weChatLogin({ termsVersion: '2026-09-10', privacyVersion: '2026-09-10' })).resolves.toMatchObject({ token: 'secret-session-token' })
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
