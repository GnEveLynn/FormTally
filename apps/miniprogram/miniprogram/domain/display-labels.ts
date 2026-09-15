export function displayLabel(value: string): string {
  return labels[value] ?? '未知'
}

const labels: Record<string, string> = {
  male: '男性',
  female: '女性',
  sedentary: '久坐',
  light: '轻度',
  moderate: '中度',
  high: '高度',
  very_high: '极高',
  automatic: '自动估算',
  manual: '手动设置',
  fat_loss: '减脂',
  maintain: '保持',
  muscle_gain: '增肌',
  slow: '慢速',
  standard: '标准',
  fast: '快速',
  breakfast: '早餐',
  lunch: '午餐',
  dinner: '晚餐',
  snack: '加餐',
  'Asia/Shanghai': '中国标准时间（上海）',
}
