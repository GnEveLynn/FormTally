import { afterEach, describe, expect, it, vi } from 'vitest'

import { ApiError, http } from './http'

describe('HTTP 客户端', () => {
  afterEach(() => vi.restoreAllMocks())

  it('默认携带 Cookie、中文语言和请求标识', async () => {
    const fetchMock = vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      new Response(JSON.stringify({ ok: true }), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      }),
    )

    await http<{ ok: boolean }>('/v1/auth/session')

    const [, init] = fetchMock.mock.calls[0]!
    const headers = new Headers(init?.headers)
    expect(init?.credentials).toBe('include')
    expect(headers.get('Accept-Language')).toBe('zh-CN')
    expect(headers.get('X-Request-ID')).toMatch(/^[0-9a-f-]{36}$/)
  })

  it('把服务端错误保留为可按 code 和字段处理的错误', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      new Response(
        JSON.stringify({
          error: {
            code: 'VALIDATION_FAILED',
            message: '部分内容需要修改',
            requestId: 'req_123',
            details: [{ field: 'phone', reason: '手机号格式错误' }],
          },
        }),
        { status: 422, headers: { 'Content-Type': 'application/json' } },
      ),
    )

    await expect(http('/v1/auth/codes')).rejects.toEqual(
      new ApiError(422, 'VALIDATION_FAILED', '部分内容需要修改', 'req_123', {
        phone: '手机号格式错误',
      }),
    )
  })

  it('不会把服务故障伪装成未登录', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(new Response('', { status: 503 }))

    await expect(http('/v1/auth/session')).rejects.toMatchObject({
      status: 503,
      code: 'HTTP_ERROR',
    })
  })
})
