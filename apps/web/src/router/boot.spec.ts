import { describe, expect, it, vi } from 'vitest'

import { ApiError } from '../api/http'
import { createSessionStore } from '../stores/session'
import { bootRoute, guardRoute } from './boot'

const session = (onboardingStatus: 'profile_required' | 'goal_required' | 'completed') => ({
  session: { expiresAt: '2026-10-10T10:00:00+08:00' },
  user: { id: 'user_123', phoneMasked: '+86 138****5678', onboardingStatus },
  consents: {
    termsVersion: '2026-09-10',
    privacyVersion: '2026-09-10',
    aiImageProcessingVersion: null,
    currentAiImageProcessingVersion: '2026-09-10',
  },
})

describe('启动路由', () => {
  it.each([
    ['profile_required', '/profile'],
    ['goal_required', '/goals'],
    ['completed', '/today'],
  ] as const)('根据 %s 进入 %s', async (status, expected) => {
    const store = createSessionStore({ getSession: vi.fn().mockResolvedValue(session(status)) })
    expect(await bootRoute(store)).toBe(expected)
  })

  it('并发启动只探测一次当前会话', async () => {
    const getSession = vi.fn().mockResolvedValue(session('completed'))
    const store = createSessionStore({ getSession })

    await Promise.all([bootRoute(store), bootRoute(store)])

    expect(getSession).toHaveBeenCalledTimes(1)
  })

  it('401 进入欢迎页，服务故障保留失败态', async () => {
    const anonymous = createSessionStore({
      getSession: vi.fn().mockRejectedValue(new ApiError(401, 'UNAUTHENTICATED', '请先登录', 'req_1')),
    })
    const failed = createSessionStore({
      getSession: vi.fn().mockRejectedValue(new ApiError(503, 'DEPENDENCY_UNAVAILABLE', '服务暂时不可用', 'req_2')),
    })

    expect(await bootRoute(anonymous)).toBe('/welcome')
    expect(await bootRoute(failed)).toBeNull()
    expect(failed.state.status).toBe('failed')
    expect(failed.state.error).toBe('服务暂时不可用')
  })

  it('失败后允许重试探测', async () => {
    const getSession = vi
      .fn()
      .mockRejectedValueOnce(new ApiError(503, 'DEPENDENCY_UNAVAILABLE', '服务暂时不可用', 'req_2'))
      .mockResolvedValueOnce(session('completed'))
    const store = createSessionStore({ getSession })

    expect(await bootRoute(store)).toBeNull()
    await store.retry()

    expect(store.state.status).toBe('authenticated')
    expect(getSession).toHaveBeenCalledTimes(2)
  })

  it('已登录用户不能绕过自己的 onboarding 路径', async () => {
    const profileRequired = createSessionStore({
      getSession: vi.fn().mockResolvedValue(session('profile_required')),
    })
    const completed = createSessionStore({ getSession: vi.fn().mockResolvedValue(session('completed')) })

    expect(await guardRoute('/today', true, profileRequired)).toBe('/profile')
    expect(await guardRoute('/today', true, completed)).toBe(true)
    expect(await guardRoute('/meals/new', true, completed)).toBe(true)
  })

	 it('目标设置用户可以进入目标结果子页面', async () => {
		 const store = createSessionStore({ getSession: vi.fn().mockResolvedValue(session('goal_required')) })
		 expect(await guardRoute('/goals/result', true, store)).toBe(true)
	 })
})
