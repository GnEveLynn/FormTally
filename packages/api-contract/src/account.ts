export type DeleteAccountInput =
  | { code: string; verificationRequestId: string; confirmation: 'DELETE' }
  | { loginCode: string; confirmation: 'DELETE' }
export interface AccountDeletion { status: 'accepted'; accessRevokedAt: string; purgeBy: string }
export interface DeleteAccountResponse { accountDeletion: AccountDeletion }
