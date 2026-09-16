import { beforeEach, describe, expect, it, vi } from 'vitest'
import { setSessionToken } from '../platform/storage'
import { ApiError, NetworkError, request } from './http'

describe('mini-program HTTP client', () => {
  const values = new Map<string, unknown>()
  let requestOptions: Record<string, any>

  beforeEach(() => {
    values.clear()
    requestOptions = {}
    Object.assign(globalThis, {
      wx: {
        getAccountInfoSync: () => ({ miniProgram: { envVersion: 'trial' } }),
        getStorageSync: vi.fn((key: string) => values.get(key)),
        setStorageSync: vi.fn((key: string, value: unknown) => values.set(key, value)),
        removeStorageSync: vi.fn((key: string) => values.delete(key)),
        reLaunch: vi.fn(),
        request: vi.fn((options: Record<string, any>) => {
          requestOptions = options
          options.success({ statusCode: 200, data: { profile: null }, header: {} })
          return { abort: vi.fn() }
        }),
      },
    })
  })

  it('adds the API base URL, Bearer token, locale, request ID, and JSON body', async () => {
    setSessionToken('session-token')
    await expect(request('/v1/profile', { method: 'PUT', body: { heightCm: 178 } })).resolves.toEqual({ profile: null })

    expect(requestOptions.url).toBe('https://api.hzcoder.xyz/v1/profile')
    expect(requestOptions.method).toBe('PUT')
    expect(requestOptions.data).toEqual({ heightCm: 178 })
    expect(requestOptions.header.Authorization).toBe('Bearer session-token')
    expect(requestOptions.header['Accept-Language']).toBe('zh-CN')
    expect(requestOptions.header['Content-Type']).toBe('application/json; charset=utf-8')
    expect(requestOptions.header['X-Request-ID']).toMatch(/^mp-/)
  })

  it('maps the shared API error shape and field details', async () => {
    ;(globalThis.wx.request as any).mockImplementationOnce((options: Record<string, any>) => {
      options.success({ statusCode: 422, data: { error: { code: 'VALIDATION_FAILED', message: '资料无效', requestId: 'req-1', details: [{ field: 'heightCm', reason: '超出范围' }] } } })
    })

    const error = await request('/v1/profile').catch((cause) => cause)
    expect(error).toBeInstanceOf(ApiError)
    expect(error).toMatchObject({ status: 422, code: 'VALIDATION_FAILED', requestId: 'req-1', fieldErrors: { heightCm: '超出范围' } })
  })

  it('clears an expired session and returns to login on 401', async () => {
    setSessionToken('expired-token')
    ;(globalThis.wx.request as any).mockImplementationOnce((options: Record<string, any>) => {
      options.success({ statusCode: 401, data: { error: { code: 'UNAUTHENTICATED', message: '请先登录', requestId: 'req-2' } } })
    })

    await expect(request('/v1/auth/session')).rejects.toMatchObject({ status: 401 })
    expect(values.has('formtally.sessionToken')).toBe(false)
    expect(globalThis.wx.reLaunch).toHaveBeenCalledWith({ url: '/pages/login/index' })
  })

  it('keeps transport failures distinct from HTTP failures', async () => {
    ;(globalThis.wx.request as any).mockImplementationOnce((options: Record<string, any>) => options.fail({ errMsg: 'request:fail timeout' }))
    await expect(request('/v1/profile')).rejects.toBeInstanceOf(NetworkError)
  })
})
