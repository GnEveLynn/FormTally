export function normalizeChinaPhone(input: string): string | null {
  const compact = input.replace(/[\s-]/g, '')
  const national = compact.startsWith('+86') ? compact.slice(3) : compact

  return /^1[3-9]\d{9}$/.test(national) ? `+86${national}` : null
}
