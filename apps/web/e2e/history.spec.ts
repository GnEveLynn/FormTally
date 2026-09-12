import { expect, test, type Page, type Route } from '@playwright/test'

const session = { session: { expiresAt: '2026-10-10T10:00:00+08:00' }, user: { id: 'user_123', phoneMasked: '+86 138****5678', onboardingStatus: 'completed' }, consents: { termsVersion: '2026-09-10', privacyVersion: '2026-09-10', aiImageProcessingVersion: null, currentAiImageProcessingVersion: '2026-09-10' } }
const nutrition = { energyKcal: 520, proteinGrams: 28, carbGrams: 65, fatGrams: 17 }
const item = { id: 'item_1', draftItemId: null, name: '鸡肉饭', grams: 350, nutrition, basisPer100Grams: null, origin: 'manual', confidence: null, assumption: null }
const meal = { id: 'meal_1', occurredAt: '2026-09-09T12:00:00+08:00', localDate: '2026-09-09', mealType: 'lunch', image: { url: '/meal.jpg', expiresAt: '2026-09-11T12:10:00+08:00', width: 1, height: 1, mimeType: 'image/jpeg' }, items: [item], totals: nutrition, estimateNotice: '图片识别和营养数据均为估算', revision: 2, createdAt: '2026-09-09T12:00:00+08:00', updatedAt: '2026-09-09T12:00:00+08:00' }
const progress = (consumed: number, target: number) => ({ consumed, target, remaining: Math.max(target - consumed, 0), overBy: Math.max(consumed - target, 0), percent: consumed / target * 100, status: 'under' })
const day = { localDate: '2026-09-09', target: { energyKcal: 2000, proteinGrams: 120, carbGrams: 250, fatGrams: 60 }, totals: nutrition, progress: { energy: progress(520, 2000), protein: progress(28, 120), carb: progress(65, 250), fat: progress(17, 60) }, mealGroups: [{ mealType: 'lunch', meals: [{ id: 'meal_1', occurredAt: meal.occurredAt, totals: nutrition }] }] }
const json = (route: Route, status: number, body: unknown) => route.fulfill({ status, contentType: 'application/json', body: JSON.stringify(body) })

async function prepare(page: Page) {
  await page.route('**/v1/auth/session', route => json(route, 200, session))
  await page.route('**/v1/history?*', route => json(route, 200, { month: '2026-09', timezone: 'Asia/Shanghai', days: [{ localDate: '2026-09-09', mealCount: 1 }] }))
  await page.route('**/v1/days/*', route => json(route, 200, { day: { ...day, localDate: route.request().url().split('/').pop() } }))
}

test('历史月历只标记正式记录并能打开详情', async ({ page }) => {
  await prepare(page)
  await page.route('**/v1/meals/meal_1', route => json(route, 200, { meal }))
  await page.goto('/history')
  await expect(page.locator('[data-date="2026-09-09"]')).toHaveClass(/recorded/)
  await page.locator('[data-date="2026-09-09"]').click()
  await page.getByRole('link', { name: /520 千卡/ }).click()
  await expect(page).toHaveURL(/\/meals\/meal_1$/)
  await expect(page.getByText('估算')).toBeVisible()
})

test('编辑携带 revision 并按 affectedLocalDates 返回历史', async ({ page }) => {
  await prepare(page)
  const refreshedDates: string[] = []
  await page.unroute('**/v1/days/*')
  await page.route('**/v1/days/*', route => {
    const localDate = route.request().url().split('/').pop()!
    refreshedDates.push(localDate)
    return json(route, 200, { day: { ...day, localDate } })
  })
  await page.route('**/v1/meals/meal_1', async route => {
    if (route.request().method() === 'PATCH') {
      expect(route.request().postDataJSON()).toMatchObject({ expectedRevision: 2 })
      return json(route, 200, { meal: { ...meal, revision: 3 }, affectedLocalDates: ['2026-09-09', '2026-09-10'] })
    }
    return json(route, 200, { meal })
  })
  await page.goto('/meals/meal_1/edit')
  await page.getByLabel('千卡').fill('530')
  await page.getByRole('button', { name: '保存修改' }).click()
  await expect(page).toHaveURL(/\/history\?date=2026-09-10/)
  expect(refreshedDates).toEqual(expect.arrayContaining(['2026-09-09', '2026-09-10']))
})

test('取消不删除，确认后才删除整餐', async ({ page }) => {
  await prepare(page)
  let deletes = 0
  await page.route('**/v1/meals/meal_1*', async route => {
    if (route.request().method() === 'DELETE') {
      deletes += 1
      return json(route, 200, { deletedMealId: 'meal_1', affectedLocalDates: ['2026-09-09'] })
    }
    return json(route, 200, { meal })
  })
  await page.goto('/meals/meal_1')
  await page.getByRole('button', { name: '删除整餐' }).click()
  await page.getByRole('button', { name: '取消' }).click()
  expect(deletes).toBe(0)
  await page.getByRole('button', { name: '删除整餐' }).click()
  await page.getByRole('dialog').getByRole('button', { name: '删除整餐' }).click()
  await expect(page).toHaveURL(/\/history\?date=2026-09-09/)
  expect(deletes).toBe(1)
})

test('revision 冲突时提示并重新加载服务端最新值', async ({ page }) => {
  await prepare(page)
  let reads = 0
  await page.route('**/v1/meals/meal_1', route => {
    if (route.request().method() === 'PATCH') return json(route, 409, { error: { code: 'REVISION_CONFLICT', message: '记录已更新' } })
    reads += 1
    return json(route, 200, { meal: reads === 1 ? meal : { ...meal, revision: 3, items: [{ ...item, nutrition: { ...nutrition, energyKcal: 540 } }] } })
  })
  await page.goto('/meals/meal_1/edit')
  await page.getByLabel('千卡').fill('530')
  await page.getByRole('button', { name: '保存修改' }).click()
  await expect(page.getByRole('alert')).toContainText('数据已变化')
  await expect(page.getByLabel('千卡')).toHaveValue('540')
})

test('只移除图片时保留食物和营养数据', async ({ page }) => {
  await prepare(page)
  await page.route('**/v1/meals/meal_1', route => json(route, 200, { meal }))
  await page.route('**/v1/meals/meal_1/image?*', route => json(route, 200, { meal: { ...meal, image: null, revision: 3 } }))
  await page.goto('/meals/meal_1')
  await page.getByRole('button', { name: '仅移除图片' }).click()
  await page.getByRole('dialog').getByRole('button', { name: '仅移除图片' }).click()
  await expect(page.getByAltText('餐食图片')).toBeHidden()
  await expect(page.getByRole('heading', { name: '鸡肉饭' })).toBeVisible()
  await expect(page.getByText('520 千卡', { exact: false }).first()).toBeVisible()
})
