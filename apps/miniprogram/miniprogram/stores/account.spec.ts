import type { GoalSettingsInput } from '@formtally/api-contract/goals'
import type { ProfileInput } from '@formtally/api-contract/profile'
import { describe, expect, it, vi } from 'vitest'
import { createAccountStore } from './account'

const profile: ProfileInput = {
  biologicalSex: 'male', birthDate: '1990-01-01', heightCm: 175, weightKg: 70,
  activityLevel: 'moderate', timezone: 'Asia/Shanghai',
  healthContext: { pregnant: false, breastfeeding: false, clinicalDietRequired: false },
}
const goal: GoalSettingsInput = { mode: 'manual', automatic: null, manual: { target: { energyKcal: 2000, proteinGrams: 100, carbGrams: 250, fatGrams: 60 } } }

describe('mini-program account store', () => {
  it('shows profile and goal save errors', async () => {
    const store = createAccountStore({
      saveProfile: vi.fn().mockRejectedValue(new Error('资料冲突')),
      saveGoal: vi.fn().mockRejectedValue(new Error('目标冲突')),
      logout: vi.fn(), requestDeleteCode: vi.fn(), deleteAccount: vi.fn(),
    })
    expect(await store.saveProfile(profile)).toBe(false)
    expect(store.state.error).toBe('资料冲突')
    expect(await store.saveGoal(goal)).toBe(false)
    expect(store.state.error).toBe('目标冲突')
  })

  it('revokes the bearer session before clearing local state on logout', async () => {
    const order: string[] = []
    const store = createAccountStore({
      saveProfile: vi.fn(), saveGoal: vi.fn(), requestDeleteCode: vi.fn(), deleteAccount: vi.fn(),
      logout: vi.fn(async () => { order.push('revoke') }),
    }, () => order.push('clear'))
    await store.logout()
    expect(order).toEqual(['revoke', 'clear'])
  })

  it('requires the bound full phone, verification code, and confirmation before deletion', async () => {
    const requestDeleteCode = vi.fn().mockResolvedValue({ verification: { requestId: 'vr_1', expiresInSeconds: 300, retryAfterSeconds: 60 } })
    const deleteAccount = vi.fn().mockResolvedValue({ accountDeletion: { status: 'accepted', accessRevokedAt: '2026-09-14T00:00:00Z', purgeBy: '2026-10-14T00:00:00Z' } })
    const clear = vi.fn()
    const store = createAccountStore({ saveProfile: vi.fn(), saveGoal: vi.fn(), logout: vi.fn(), requestDeleteCode, deleteAccount }, clear)

    expect(await store.requestDeleteCode('138 0013 8000')).toBe(true)
    expect(requestDeleteCode).toHaveBeenCalledWith('+8613800138000')
    expect(await store.deleteAccount('123456', 'delete')).toBe(false)
    expect(deleteAccount).not.toHaveBeenCalled()
    expect(await store.deleteAccount('123456', 'DELETE')).toBe(true)
    expect(deleteAccount).toHaveBeenCalledWith({ code: '123456', verificationRequestId: 'vr_1', confirmation: 'DELETE' })
    expect(clear).toHaveBeenCalledOnce()
  })

  it('uses a fresh WeChat login code when deleting an account without a phone', async () => {
    Object.assign(globalThis, { wx: { login: ({ success }: Record<string, any>) => success({ code: 'fresh-login-code' }) } })
    const deleteAccount = vi.fn().mockResolvedValue({ accountDeletion: { status: 'accepted' } })
    const clear = vi.fn()
    const store = createAccountStore({ saveProfile: vi.fn(), saveGoal: vi.fn(), logout: vi.fn(), requestDeleteCode: vi.fn(), deleteAccount }, clear)
    expect(await store.deleteWeChatAccount('DELETE')).toBe(true)
    expect(deleteAccount).toHaveBeenCalledWith({ loginCode: 'fresh-login-code', confirmation: 'DELETE' })
    expect(clear).toHaveBeenCalledOnce()
  })
})
