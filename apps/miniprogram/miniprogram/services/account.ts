import type { DeleteAccountInput, DeleteAccountResponse } from '@formtally/api-contract/account'
import type { VerificationResponse } from '@formtally/api-contract/auth'
import { request } from './http'

export const requestDeleteCode = (phone: string): Promise<VerificationResponse> =>
  request('/v1/auth/codes', { method: 'POST', body: { phone, purpose: 'delete_account' } })

export const deleteAccount = (input: DeleteAccountInput): Promise<DeleteAccountResponse> =>
  request('/v1/account-deletions', { method: 'POST', body: input })
