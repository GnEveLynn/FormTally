import { expect, test, type Page, type Route } from '@playwright/test'

const currentDate = new Date().toLocaleDateString('en-CA')
const currentMonth = currentDate.slice(0, 7)
const occurredAt = new Date(Date.now() - 60_000).toISOString()
const session = {
  session: { expiresAt: '2026-10-10T10:00:00+08:00' },
  user: { id: 'user_full_journey', phoneMasked: '+86 138****5678', onboardingStatus: 'completed' },
  consents: { termsVersion: '2026-09-10', privacyVersion: '2026-09-10', aiImageProcessingVersion: '2026-09-10', currentAiImageProcessingVersion: '2026-09-10' },
}
const target = { energyKcal: 2000, proteinGrams: 120, carbGrams: 250, fatGrams: 60 }
const nutrition = { energyKcal: 520, proteinGrams: 28, carbGrams: 65, fatGrams: 17 }
const item = { id: 'item_1', draftItemId: 'draft_1', name: '鸡肉饭', grams: 350, nutrition, basisPer100Grams: null, origin: 'ai_modified', confidence: 'medium', assumption: '按一份熟制鸡肉饭估算' }
const analysis = { id: 'analysis_1', status: 'review_required', processingMode: 'ai', occurredAt, localDate: currentDate, mealType: 'lunch', image: { url: '/meal.jpg', expiresAt: `${currentDate}T12:10:00+08:00`, width: 1, height: 1, mimeType: 'image/jpeg' }, items: [item], warnings: [], failure: null, mealId: null, expiresAt: `${currentDate}T23:59:59+08:00`, revision: 1, createdAt: occurredAt }
const meal = { id: 'meal_1', occurredAt, localDate: currentDate, mealType: 'lunch', image: analysis.image, items: [item], totals: nutrition, estimateNotice: '图片识别和营养数据均为估算', revision: 1, createdAt: occurredAt, updatedAt: occurredAt }

const json = (route: Route, status: number, body: unknown) => route.fulfill({ status, contentType: 'application/json', body: JSON.stringify(body) })
const metric = (consumed: number, goal: number) => ({ consumed, target: goal, remaining: Math.max(goal - consumed, 0), overBy: Math.max(consumed - goal, 0), percent: consumed / goal * 100, status: consumed > goal ? 'over' : 'under' })
const day = (saved: boolean, value = nutrition) => ({ day: { localDate: currentDate, target, totals: saved ? value : { energyKcal: 0, proteinGrams: 0, carbGrams: 0, fatGrams: 0 }, progress: { energy: metric(saved ? value.energyKcal : 0, target.energyKcal), protein: metric(saved ? value.proteinGrams : 0, target.proteinGrams), carb: metric(saved ? value.carbGrams : 0, target.carbGrams), fat: metric(saved ? value.fatGrams : 0, target.fatGrams) }, mealGroups: saved ? [{ mealType: 'lunch', meals: [{ id: 'meal_1', occurredAt, totals: value }] }] : [] } })

async function installJourneyAPI(page: Page) {
  let saved = false
  let deleted = false
  let currentMeal = structuredClone(meal)
  await page.route('**/v1/auth/session', route => json(route, 200, session))
  await page.route('**/v1/meal-analyses', route => json(route, 201, { analysis }))
  await page.route('**/v1/history?*', route => json(route, 200, { month: currentMonth, timezone: 'Asia/Shanghai', days: saved && !deleted ? [{ localDate: currentDate, mealCount: 1 }] : [] }))
  await page.route('**/v1/days/*', route => json(route, 200, day(saved && !deleted, currentMeal.totals)))
  await page.route('**/v1/meals', route => {
    saved = true
    return json(route, 201, { meal: currentMeal, affectedLocalDates: [currentDate] })
  })
  await page.route('**/v1/meals/meal_1*', route => {
    if (route.request().method() === 'PATCH') {
      const input = route.request().postDataJSON()
      currentMeal = { ...currentMeal, items: input.items, totals: input.items[0].nutrition, revision: 2 }
      return json(route, 200, { meal: currentMeal, affectedLocalDates: [currentDate] })
    }
    if (route.request().method() === 'DELETE') {
      deleted = true
      return json(route, 200, { deletedMealId: 'meal_1', affectedLocalDates: [currentDate] })
    }
    return json(route, 200, { meal: currentMeal })
  })
}

test('完整用户旅程覆盖分析、保存、今日、编辑、历史和删除', async ({ page }) => {
  await installJourneyAPI(page)
  await page.goto('/meals/new')
  await page.locator('input[type=file]').setInputFiles({ name: 'meal.png', mimeType: 'image/png', buffer: Buffer.from('iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+A8AAQUBAScY42YAAAAASUVORK5CYII=', 'base64') })
  await page.getByRole('button', { name: '确认图片' }).click()
  await expect(page.getByRole('dialog')).toContainText('第三方 AI 服务')
  await page.getByRole('button', { name: '同意并开始分析' }).click()
  await expect(page).toHaveURL(/\/meals\/confirm$/)
  await expect(page.getByText('营养数据均为估算', { exact: false })).toBeVisible()
  await page.getByRole('button', { name: '保存本餐' }).click()

  await expect(page).toHaveURL(/\/today$/)
  await page.getByRole('link', { name: /520 千卡/ }).click()
  await expect(page.getByText('估算值：')).toBeVisible()
  await page.getByRole('link', { name: '编辑' }).click()
  await page.getByLabel('千卡').fill('530')
  await page.getByRole('button', { name: '保存修改' }).click()

  await expect(page).toHaveURL(new RegExp(`/history\\?date=${currentDate}`))
  await page.getByRole('link', { name: /530 千卡/ }).click()
  await page.getByRole('button', { name: '删除整餐' }).click()
  await expect(page.getByRole('dialog')).toContainText('餐食、明细和图片都会进入删除流程')
  await page.getByRole('dialog').getByRole('button', { name: '删除整餐' }).click()
  await expect(page.getByRole('heading', { name: '这天没有记录' })).toBeVisible()
})

test('今日页在一秒后保持 loading、三秒内展示数据并提供文字超标状态', async ({ page }) => {
  await page.route('**/v1/auth/session', route => json(route, 200, session))
  await page.route('**/v1/days/*', async route => {
    await new Promise(resolve => setTimeout(resolve, 1100))
    const over = { energyKcal: 2100, proteinGrams: 121, carbGrams: 251, fatGrams: 61 }
    await json(route, 200, day(true, over))
  })
  const started = Date.now()
  await page.goto('/today')
  await expect(page.getByRole('status')).toContainText('正在加载')
  await expect(page.getByText('已超出', { exact: false }).first()).toBeVisible({ timeout: 3000 })
  expect(Date.now() - started).toBeLessThan(3000)
  await page.addStyleTag({ content: ':root { font-size: 200% !important; }' })
  const record = page.getByRole('button', { name: '▣ 记录饮食' })
  await expect(record).toBeVisible()
  expect((await record.boundingBox())?.height).toBeGreaterThanOrEqual(44)
  await page.goto('/me')
  expect((await page.getByRole('button', { name: '退出登录' }).boundingBox())?.height).toBeGreaterThanOrEqual(44)
})

test('AI 超过十二秒仍显示处理中反馈', async ({ page }) => {
  await page.route('**/v1/auth/session', route => json(route, 200, session))
  await page.route('**/v1/meal-analyses', async route => {
    await new Promise(resolve => setTimeout(resolve, 14_000))
    await json(route, 201, { analysis })
  })
  await page.goto('/meals/new')
  await page.locator('input[type=file]').setInputFiles({ name: 'meal.png', mimeType: 'image/png', buffer: Buffer.from('iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+A8AAQUBAScY42YAAAAASUVORK5CYII=', 'base64') })
  await page.getByRole('button', { name: '确认图片' }).click()
  await page.getByRole('button', { name: '同意并开始分析' }).click()
  await expect(page.getByText('仍在认真识别')).toBeVisible({ timeout: 13_000 })
  await expect(page).toHaveURL(/\/meals\/confirm$/, { timeout: 16_000 })
})
