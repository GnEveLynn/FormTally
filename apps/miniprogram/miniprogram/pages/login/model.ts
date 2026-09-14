import type { OnboardingStatus, WeChatSessionResult } from '@formtally/api-contract/auth'

export const agreementVersion = '2026-09-10'

export function routeForSession(status: OnboardingStatus): '/pages/today/index' | '/pages/onboarding-profile/index' | '/pages/onboarding-goal/index' {
  if (status === 'profile_required') return '/pages/onboarding-profile/index'
  if (status === 'goal_required') return '/pages/onboarding-goal/index'
  return '/pages/today/index'
}

export function nextLoginAction(result: WeChatSessionResult): { bindingTicket: string } | { route: ReturnType<typeof routeForSession> } {
  return result.bindingRequired ? { bindingTicket: result.bindingTicket } : { route: routeForSession(result.user.onboardingStatus) }
}

export function phoneAuthorization(input: { acceptedAgreements: boolean; code?: string; errMsg?: string }): { code: string } | { error: string } {
  if (!input.acceptedAgreements) return { error: '请先阅读并同意服务协议与隐私政策' }
  if (input.errMsg !== 'getPhoneNumber:ok' || !input.code) return { error: '未获得手机号授权，你可以再次尝试' }
  return { code: input.code }
}
