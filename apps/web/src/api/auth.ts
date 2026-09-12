import type { LoginRequest, SessionResponse, VerificationResponse } from '@formtally/api-contract/auth'

import { http } from './http'

export function requestLoginCode(phone: string): Promise<VerificationResponse> {
  return http('/v1/auth/codes', {
    method: 'POST',
    body: JSON.stringify({ phone, purpose: 'login' }),
  })
}

export function requestDeleteCode(phone: string): Promise<VerificationResponse> {
  return http('/v1/auth/codes', { method: 'POST', body: JSON.stringify({ phone, purpose: 'delete_account' }) })
}

export function login(input: LoginRequest): Promise<SessionResponse> {
  return http('/v1/auth/sessions', { method: 'POST', body: JSON.stringify(input) })
}

export function getSession(): Promise<SessionResponse> {
  return http('/v1/auth/session')
}

export function logout(): Promise<void> {
  return http('/v1/auth/session', { method: 'DELETE' })
}
