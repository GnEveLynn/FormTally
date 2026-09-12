import type { DeleteAccountInput, DeleteAccountResponse } from '@formtally/api-contract/account'

import { http } from './http'

export const deleteAccount = (input: DeleteAccountInput) => http<DeleteAccountResponse>('/v1/account-deletions', { method: 'POST', body: JSON.stringify(input) })
