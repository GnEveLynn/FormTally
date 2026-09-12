import { expect, test, type Page, type Route } from '@playwright/test'

const session = {
  session: { expiresAt: '2026-10-10T10:00:00+08:00' },
  user: { id: 'user_123', phoneMasked: '+86 138****5678', onboardingStatus: 'completed' },
  consents: { termsVersion: '2026-09-10', privacyVersion: '2026-09-10', aiImageProcessingVersion: null, currentAiImageProcessingVersion: '2026-09-10' },
}
const nutrition = { energyKcal: 520, proteinGrams: 28, carbGrams: 65, fatGrams: 17 }
const item = { draftItemId: 'draft_1', name: '鸡肉饭', grams: 350, nutrition, basisPer100Grams: { energyKcal: 149, proteinGrams: 8, carbGrams: 18.6, fatGrams: 4.9 }, origin: 'ai' as const, confidence: 'medium' as const, assumption: '按一份熟制鸡肉饭估算' }
const baseAnalysis = {
  id: 'analysis_1', status: 'review_required', processingMode: 'ai', occurredAt: '2026-09-11T12:00:00+08:00', localDate: '2026-09-11', mealType: 'lunch',
  image: { url: '/private/image', expiresAt: '2026-09-11T12:10:00+08:00', width: 1, height: 1, mimeType: 'image/jpeg' }, items: [item], warnings: [], failure: null, mealId: null, expiresAt: '2026-09-12T12:00:00+08:00', revision: 1, createdAt: '2026-09-11T12:00:00+08:00',
}

const json = (route: Route, status: number, body: unknown) => route.fulfill({ status, contentType: 'application/json', body: JSON.stringify(body) })
const emptyDay = () => ({ day: { localDate: new Date().toLocaleDateString('en-CA'), target: { energyKcal: 2000, proteinGrams: 100, carbGrams: 250, fatGrams: 60 }, totals: { energyKcal: 0, proteinGrams: 0, carbGrams: 0, fatGrams: 0 }, progress: { energy: { consumed: 0, target: 2000, remaining: 2000, overBy: 0, percent: 0, status: 'under' }, protein: { consumed: 0, target: 100, remaining: 100, overBy: 0, percent: 0, status: 'under' }, carb: { consumed: 0, target: 250, remaining: 250, overBy: 0, percent: 0, status: 'under' }, fat: { consumed: 0, target: 60, remaining: 60, overBy: 0, percent: 0, status: 'under' } }, mealGroups: [] } })

async function prepare(page: Page, analysis: object = baseAnalysis) {
  await page.route('**/v1/auth/session', route => json(route, 200, session))
  await page.route('**/v1/days/*', route => json(route, 200, emptyDay()))
  await page.route('**/v1/meal-analyses', route => json(route, 201, { analysis }))
  await page.goto('/meals/new')
  await page.locator('input[type=file]').setInputFiles({ name: 'meal.png', mimeType: 'image/png', buffer: Buffer.from('iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+A8AAQUBAScY42YAAAAASUVORK5CYII=', 'base64') })
  await expect(page.getByAltText('餐食图片预览')).toBeVisible()
  await page.getByRole('button', { name: '确认图片' }).click()
  await page.getByRole('button', { name: '同意并开始分析' }).click()
}

const mealResponse = { meal: { id: 'meal_1', occurredAt: '2026-09-11T12:00:00+08:00', localDate: '2026-09-11', mealType: 'lunch', image: null, items: [item], totals: nutrition, estimateNotice: '图片识别和营养数据均为估算', revision: 1, createdAt: '2026-09-11T12:00:00+08:00', updatedAt: '2026-09-11T12:00:00+08:00' }, affectedLocalDates: ['2026-09-11'] }

test('图片分析、确认并保存后返回今日页', async ({ page }) => {
  await page.route('**/v1/meals', route => json(route, 201, mealResponse))
  await prepare(page)
  await expect(page).toHaveURL(/\/meals\/confirm$/)
  await expect(page.getByLabel('食物名称')).toHaveValue('鸡肉饭')
  await page.getByRole('button', { name: '保存本餐' }).click()
  await expect(page).toHaveURL(/\/today$/)
})

test('AI 失败后保留图片和上下文并可转手工保存', async ({ page }) => {
  const failed = { ...baseAnalysis, status: 'failed', items: [], failure: { code: 'AI_TIMEOUT', message: '分析超时，可以手动录入', retryable: true } }
  await page.route('**/v1/meals', route => json(route, 201, mealResponse))
  await prepare(page, failed)
  await expect(page.getByRole('heading', { name: '暂时没分析出来' })).toBeVisible()
  await page.getByRole('button', { name: '手动录入' }).click()
  await page.getByLabel('食物名称').fill('米饭')
  await page.getByLabel('千卡').fill('116')
  await page.getByRole('button', { name: '保存本餐' }).click()
  await expect(page).toHaveURL(/\/today$/)
})

test('保存响应丢失后重试复用同一个幂等键', async ({ page }) => {
  const keys: string[] = []
  let attempts = 0
  await page.route('**/v1/meals', route => {
    attempts += 1
    keys.push(route.request().headers()['idempotency-key'] ?? '')
    return attempts === 1 ? route.abort('connectionreset') : json(route, 201, mealResponse)
  })
  await prepare(page)
  await page.getByRole('button', { name: '保存本餐' }).click()
  await expect(page.getByRole('alert')).not.toBeEmpty()
  await page.getByRole('button', { name: '保存本餐' }).click()
  await expect(page).toHaveURL(/\/today$/)
  expect(keys).toHaveLength(2)
  expect(keys[0]).toBe(keys[1])
})
