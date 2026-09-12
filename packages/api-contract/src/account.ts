export interface DeleteAccountInput { code: string; verificationRequestId: string; confirmation: 'DELETE' }
export interface AccountDeletion { status: 'accepted'; accessRevokedAt: string; purgeBy: string }
export interface DeleteAccountResponse { accountDeletion: AccountDeletion }
