import { beforeEach, describe, expect, it, vi } from 'vitest'
import { saveProfile } from './profile'

describe('profile service', () => {
  let requestOptions: Record<string, any>

  beforeEach(() => {
    Object.assign(globalThis, {
      wx: {
        getAccountInfoSync: () => ({ miniProgram: { envVersion: 'trial' } }),
        getStorageSync: vi.fn(),
        request: vi.fn((options: Record<string, any>) => {
          requestOptions = options
          options.success({ statusCode: 200, data: { profile: null } })
        }),
      },
    })
  })

  it('保存时不发送资料响应中的只读字段', async () => {
    await saveProfile({
      biologicalSex: 'male', birthDate: '2002-09-15', heightCm: 183, weightKg: 77,
      activityLevel: 'moderate', timezone: 'Asia/Shanghai',
      healthContext: { pregnant: false, breastfeeding: false, clinicalDietRequired: false },
      expectedRevision: 1,
      automaticGoalEligible: true, revision: 1, updatedAt: '2026-09-15T07:00:00Z',
    } as Parameters<typeof saveProfile>[0])

    expect(requestOptions.data).toEqual({
      biologicalSex: 'male', birthDate: '2002-09-15', heightCm: 183, weightKg: 77,
      activityLevel: 'moderate', timezone: 'Asia/Shanghai',
      healthContext: { pregnant: false, breastfeeding: false, clinicalDietRequired: false },
      expectedRevision: 1,
    })
  })
})
