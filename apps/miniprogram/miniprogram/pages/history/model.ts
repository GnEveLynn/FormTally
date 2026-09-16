export async function loadHistoryPage(store: { state: { selectedDate: string }; loadMonth(): Promise<void>; selectDate(date: string): Promise<void> }, sync: () => void) {
  const pending = Promise.all([store.loadMonth(), store.selectDate(store.state.selectedDate)])
  sync()
  await pending
  sync()
}

export function shiftMonth(month: string, offset: number): string {
  const [year, value] = month.split('-').map(Number)
  const date = new Date(Date.UTC(year!, value! - 1 + offset, 1))
  return `${date.getUTCFullYear()}-${String(date.getUTCMonth() + 1).padStart(2, '0')}`
}

export function dateForMonth(month: string, recordedDates: Set<string>): string {
  return [...recordedDates].sort()[0] ?? `${month}-01`
}

export function buildCalendar(month: string, recordedDates: Set<string>, selectedDate: string) {
  const [year, value] = month.split('-').map(Number)
  const first = new Date(Date.UTC(year!, value! - 1, 1))
  const mondayOffset = (first.getUTCDay() + 6) % 7
  const start = new Date(first)
  start.setUTCDate(1 - mondayOffset)

  return Array.from({ length: 42 }, (_, index) => {
    const date = new Date(start)
    date.setUTCDate(start.getUTCDate() + index)
    const localDate = date.toISOString().slice(0, 10)
    return { date: localDate, day: date.getUTCDate(), inMonth: date.getUTCMonth() === value! - 1, recorded: recordedDates.has(localDate), selected: localDate === selectedDate }
  })
}
