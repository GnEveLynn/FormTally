export async function loadHistoryPage(store: { state: { selectedDate: string }; loadMonth(): Promise<void>; selectDate(date: string): Promise<void> }, sync: () => void) {
  const pending = Promise.all([store.loadMonth(), store.selectDate(store.state.selectedDate)])
  sync()
  await pending
  sync()
}
