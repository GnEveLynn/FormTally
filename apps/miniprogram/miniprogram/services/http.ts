import type { ApiErrorBody } from '@formtally/api-contract/errors'
import { currentApiBaseUrl } from '../config/env'
import { clearSessionToken, getSessionToken } from '../platform/storage'

export class ApiError extends Error {
  constructor(
    readonly status: number,
    readonly code: string,
    message: string,
    readonly requestId = '',
    readonly fieldErrors: Record<string, string> = {},
  ) {
    super(message)
    this.name = 'ApiError'
  }
}

export class NetworkError extends Error {
  constructor(message = '网络连接失败，请重试') {
    super(message)
    this.name = 'NetworkError'
  }
}

export interface RequestOptions {
  method?: WechatMiniprogram.RequestOption['method']
  body?: WechatMiniprogram.IAnyObject | string | ArrayBuffer
  headers?: Record<string, string>
}

export function requestId(): string {
  return `mp-${Date.now().toString(36)}-${Math.random().toString(36).slice(2)}`
}

function errorBody(data: unknown): ApiErrorBody['error'] | undefined {
  let value = data
  if (typeof value === 'string') {
    try { value = JSON.parse(value) as unknown } catch { return undefined }
  }
  if (!value || typeof value !== 'object' || !('error' in value)) return undefined
  const error = (value as { error?: unknown }).error
  return error && typeof error === 'object' ? error as ApiErrorBody['error'] : undefined
}

export function apiError(status: number, data: unknown): ApiError {
  const body = errorBody(data)
  const fieldErrors = Object.fromEntries(body?.details?.map(({ field, reason }) => [field, reason]) ?? [])
  return new ApiError(
    status,
    body?.code ?? 'HTTP_ERROR',
    body?.message ?? (status >= 500 ? '服务暂时不可用，请稍后重试' : '请求失败，请重试'),
    body?.requestId,
    fieldErrors,
  )
}

export function authHeaders(extra: Record<string, string> = {}): Record<string, string> {
  const token = getSessionToken()
  return {
    'Accept-Language': 'zh-CN',
    'X-Request-ID': requestId(),
    ...(token ? { Authorization: `Bearer ${token}` } : {}),
    ...extra,
  }
}

export function request<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const headers = authHeaders(options.headers)
  if (options.body !== undefined && !headers['Content-Type']) headers['Content-Type'] = 'application/json; charset=utf-8'

  return new Promise<T>((resolve, reject) => {
    wx.request({
      url: `${currentApiBaseUrl()}${path}`,
      method: options.method ?? 'GET',
      data: options.body,
      header: headers,
      success(response) {
        if (response.statusCode >= 200 && response.statusCode < 300) {
          resolve(response.statusCode === 204 ? undefined as T : response.data as T)
          return
        }
        const error = apiError(response.statusCode, response.data)
        if (response.statusCode === 401) {
          clearSessionToken()
          wx.reLaunch({ url: '/pages/login/index' })
        }
        reject(error)
      },
      fail() {
        reject(new NetworkError())
      },
    })
  })
}
