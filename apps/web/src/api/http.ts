import type { ApiErrorBody } from '@formtally/api-contract/errors'

export class ApiError extends Error {
  readonly status: number
  readonly code: string
  readonly requestId: string
  readonly fieldErrors: Record<string, string>

  constructor(
    status: number,
    code: string,
    message: string,
    requestId = '',
    fieldErrors: Record<string, string> = {},
  ) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.code = code
    this.requestId = requestId
    this.fieldErrors = fieldErrors
  }
}

export async function http<T>(path: string, init: RequestInit = {}): Promise<T> {
  const headers = new Headers(init.headers)
  headers.set('Accept-Language', 'zh-CN')
  headers.set('X-Request-ID', crypto.randomUUID())
  if (init.body && !(init.body instanceof FormData) && !headers.has('Content-Type')) {
    headers.set('Content-Type', 'application/json; charset=utf-8')
  }

  const response = await fetch(path, { ...init, headers, credentials: 'include' })
  if (response.ok) {
    return response.status === 204 ? (undefined as T) : ((await response.json()) as T)
  }

  let body: ApiErrorBody | null = null
  try {
    body = (await response.json()) as ApiErrorBody
  } catch {
    // Empty and non-JSON failures still retain their HTTP status.
  }
  const error = body?.error
  const fieldErrors = Object.fromEntries(error?.details?.map(({ field, reason }) => [field, reason]) ?? [])
  throw new ApiError(
    response.status,
    error?.code ?? 'HTTP_ERROR',
    error?.message ?? (response.status >= 500 ? '服务暂时不可用，请稍后重试' : '请求失败，请重试'),
    error?.requestId,
    fieldErrors,
  )
}
