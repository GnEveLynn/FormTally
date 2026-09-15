import type { OnboardingStatus, WeChatSessionResult } from '@formtally/api-contract/auth'

export const agreementVersion = '2026-09-10'

export function routeForSession(status: OnboardingStatus): '/pages/today/index' | '/pages/onboarding-profile/index' | '/pages/onboarding-goal/index' {
  if (status === 'profile_required') return '/pages/onboarding-profile/index'
  if (status === 'goal_required') return '/pages/onboarding-goal/index'
  return '/pages/today/index'
}

export function nextLoginAction(result: WeChatSessionResult): { route: ReturnType<typeof routeForSession> } {
  return { route: routeForSession(result.user.onboardingStatus) }
}
