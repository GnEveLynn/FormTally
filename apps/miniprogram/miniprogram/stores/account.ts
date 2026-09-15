import type { DeleteAccountInput } from '@formtally/api-contract/account'
import type { GoalSettingsInput } from '@formtally/api-contract/goals'
import type { ProfileInput } from '@formtally/api-contract/profile'
import { normalizeChinaPhone } from '../domain/phone'
import { freshWeChatLoginCode } from '../platform/auth'
import * as accountApi from '../services/account'
import * as goalsApi from '../services/goals'
import * as profileApi from '../services/profile'
import { mealDraftStore } from './meal-draft'
import { sessionStore } from './session'

interface AccountClient {
  saveProfile: typeof profileApi.saveProfile
  saveGoal: typeof goalsApi.saveGoal
  logout(): Promise<void>
  requestDeleteCode: typeof accountApi.requestDeleteCode
  deleteAccount(input: DeleteAccountInput): ReturnType<typeof accountApi.deleteAccount>
}

export function createAccountStore(client: AccountClient = { ...profileApi, ...goalsApi, ...accountApi, logout: sessionStore.logout }, clearLocal = () => { sessionStore.clear(); mealDraftStore.clear() }) {
  const state = { busy: false, error: '', verificationRequestId: '' }
  async function run(operation: () => Promise<unknown>) {
    state.busy = true; state.error = ''
    try { await operation(); return true } catch (error) {
      state.error = error instanceof Error ? error.message : '请求失败，请重试'; return false
    } finally { state.busy = false }
  }
  return {
    state,
    saveProfile: (input: ProfileInput) => run(() => client.saveProfile(input)),
    saveGoal: (input: GoalSettingsInput) => run(() => client.saveGoal(input)),
    async logout() { try { await client.logout() } finally { clearLocal() } },
    async requestDeleteCode(phone: string) {
      const normalized = normalizeChinaPhone(phone)
      if (!normalized) { state.error = '请输入当前账户绑定的完整手机号'; return false }
      return run(async () => { state.verificationRequestId = (await client.requestDeleteCode(normalized)).verification.requestId })
    },
    async deleteAccount(code: string, confirmation: string) {
      if (!state.verificationRequestId || !/^\d{6}$/.test(code)) { state.error = '请先获取并输入 6 位验证码'; return false }
      if (confirmation !== 'DELETE') { state.error = '请输入 DELETE 确认删除'; return false }
      const success = await run(() => client.deleteAccount({ code, verificationRequestId: state.verificationRequestId, confirmation: 'DELETE' }))
      if (success) clearLocal()
      return success
    },
    async deleteWeChatAccount(confirmation: string) {
      if (confirmation !== 'DELETE') { state.error = '请输入 DELETE 确认删除'; return false }
      const success = await run(async () => client.deleteAccount({ loginCode: await freshWeChatLoginCode(), confirmation: 'DELETE' }))
      if (success) clearLocal()
      return success
    },
  }
}

export const accountStore = createAccountStore()
