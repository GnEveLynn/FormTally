import { describe, expect, it } from 'vitest'
import { nextLoginAction, phoneAuthorization, routeForSession } from './model'

describe('login flow decisions', () => {
  it.each([
    ['profile_required', '/pages/onboarding-profile/index'],
    ['goal_required', '/pages/onboarding-goal/index'],
    ['completed', '/pages/today/index'],
  ] as const)('routes %s sessions to %s', (status, route) => {
    expect(routeForSession(status)).toBe(route)
  })

  it('requires explicit agreement before accepting a phone code', () => {
    expect(phoneAuthorization({ acceptedAgreements: false, code: 'phone-code', errMsg: 'getPhoneNumber:ok' })).toEqual({ error: '请先阅读并同意服务协议与隐私政策' })
  })

  it('asks for a phone only when the server says binding is required', () => {
    expect(nextLoginAction({ bindingRequired: true, bindingTicket: 'ticket-1', expiresInSeconds: 300 })).toEqual({ bindingTicket: 'ticket-1' })
    expect(nextLoginAction({ bindingRequired: false, token: 'token', session: { expiresAt: '2026-09-15T00:00:00Z' }, user: { id: 'u1', phoneMasked: '138****5678', onboardingStatus: 'completed' }, consents: { termsVersion: '2026-09-10', privacyVersion: '2026-09-10', aiImageProcessingVersion: null, currentAiImageProcessingVersion: '2026-09-10' } })).toEqual({ route: '/pages/today/index' })
  })

  it('keeps the user on login after refusal and accepts a later retry', () => {
    expect(phoneAuthorization({ acceptedAgreements: true, errMsg: 'getPhoneNumber:fail user deny' })).toEqual({ error: '未获得手机号授权，你可以再次尝试' })
    expect(phoneAuthorization({ acceptedAgreements: true, code: 'fresh-phone-code', errMsg: 'getPhoneNumber:ok' })).toEqual({ code: 'fresh-phone-code' })
  })
})
