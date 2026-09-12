import type { HistoryResponse } from '@formtally/api-contract/history'

import { http } from './http'

export const getHistory = (month: string) => http<HistoryResponse>(`/v1/history?month=${encodeURIComponent(month)}`)
