import type { DayResponse, DaySummary } from '@formtally/api-contract/days'
import { request } from './http'

export async function getDay(localDate: string): Promise<DaySummary> {
  return (await request<DayResponse>(`/v1/days/${localDate}`)).day
}
