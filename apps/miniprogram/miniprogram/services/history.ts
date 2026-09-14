import type { HistoryResponse } from '@formtally/api-contract/history'
import { request } from './http'

export const getHistory = (month: string): Promise<HistoryResponse> =>
  request(`/v1/history?month=${encodeURIComponent(month)}`)
