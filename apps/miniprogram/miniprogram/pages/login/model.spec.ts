import { describe, expect, it } from 'vitest'
import { nextLoginAction, routeForSession } from './model'

describe('login flow decisions', () => {
  it.each([
    ['profile_required', '/pages/onboarding-profile/index'],
    ['goal_required', '/pages/onboarding-goal/index'],
    ['completed', '/pages/today/index'],
  ] as const)('routes %s sessions to %s', (status, route) => {
    expect(routeForSession(status)).toBe(route)
  })

  it('routes a direct WeChat session without phone binding', () => {
    expect(nextLoginAction({ token: 'token', session: { expiresAt: '2026-09-15T00:00:00Z' }, user: { id: 'u1', phoneMasked: null, onboardingStatus: 'completed' }, consents: { termsVersion: '2026-09-10', privacyVersion: '2026-09-10', aiImageProcessingVersion: null, currentAiImageProcessingVersion: '2026-09-10' } })).toEqual({ route: '/pages/today/index' })
  })
})
