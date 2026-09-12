export interface HistoryDay {
  localDate: string
  mealCount: number
  energyKcal?: number
  status?: string
}

export interface HistoryResponse {
  month: string
  timezone: string
  days: HistoryDay[]
}
