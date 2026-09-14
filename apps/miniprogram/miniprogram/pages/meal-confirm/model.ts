import type { MealItem } from '@formtally/api-contract/analyses'
import { sumNutrition } from '@formtally/domain/meal-editor'

export function reviewModel(items: MealItem[], warnings: string[]) {
  return {
    estimateNotice: '图片识别和营养数据均为估算，请按实际情况修改。',
    warnings: [...warnings],
    lowConfidence: items.flatMap((item, index) => item.confidence === 'low'
      ? [{ index, name: item.name, assumption: item.assumption ?? '请核对食物和份量' }]
      : []),
    totals: sumNutrition(items.map(({ nutrition }) => nutrition)),
  }
}

export async function saveAndRefresh(
  draft: { save(): Promise<{ affectedLocalDates: string[] } | undefined>; clear(): void },
  today: { refreshAfterSave(dates: string[]): Promise<void> },
): Promise<boolean> {
  const result = await draft.save()
  if (!result) return false
  await today.refreshAfterSave(result.affectedLocalDates)
  draft.clear()
  return true
}
