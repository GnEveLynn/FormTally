import { expect, test, type Page, type Route } from '@playwright/test'

const session = { session: { expiresAt: '2026-10-10T10:00:00+08:00' }, user: { id: 'user_123', phoneMasked: '+86 138****5678', onboardingStatus: 'completed' }, consents: { termsVersion: '2026-09-10', privacyVersion: '2026-09-10', aiImageProcessingVersion: null, currentAiImageProcessingVersion: '2026-09-10' } }
const profile = { biologicalSex: 'male', birthDate: '1995-06-18', heightCm: 178, weightKg: 72.5, activityLevel: 'moderate', timezone: 'Asia/Shanghai', healthContext: { pregnant: false, breastfeeding: false, clinicalDietRequired: false }, automaticGoalEligible: true, revision: 3, updatedAt: '2026-09-11T10:00:00+08:00' }
const calculation = { calculationVersion: 'daily_nutrition_target_v1', method: { id: 'mifflin_st_jeor_v1', displayName: 'Mifflin–St Jeor 静息能量估算', sourceUrl: 'https://pubmed.ncbi.nlm.nih.gov/2305711/', formulaExpression: '10 × 体重kg + 6.25 × 身高cm - 5 × 年龄 + 5' }, macroMethod: { id: 'macro_split_50_25_25_v1', carbPercent: 50, proteinPercent: 25, fatPercent: 25 }, inputs: { biologicalSex: 'male', ageYears: 31, heightCm: 178, weightKg: 72.5, activityLevel: 'moderate', activityMultiplier: 1.55, objective: 'maintain', pace: null, goalAdjustmentPercent: 0 }, steps: [{ key: 'finalEnergyTarget', label: '取整后的每日热量目标', value: 2400, unit: 'kcal/day' }], rounding: { energy: 'nearest_10_kcal', macros: 'nearest_1_gram', halfRule: 'half_away_from_zero' }, disclaimer: '该结果是估算起点，不构成医学建议。' }
const settings = { mode: 'automatic', automatic: { objective: 'fat_loss', pace: 'standard' }, manual: null, revision: 2, updatedAt: '2026-09-11T10:00:00+08:00' }
const target = { effectiveFrom: '2026-09-12', target: { energyKcal: 2400, proteinGrams: 150, carbGrams: 300, fatGrams: 67 }, calculation, warnings: [] }
const json = (route: Route, status: number, body: unknown) => route.fulfill({ status, contentType: 'application/json', body: JSON.stringify(body) })
async function prepare(page: Page) {
  await page.route('**/v1/auth/session', route => json(route, 200, session))
  await page.route('**/v1/profile', route => json(route, 200, { profile }))
  await page.route('**/v1/goals', route => json(route, 200, { settings, activeTarget: { ...target, localDate: '2026-09-11' }, pendingTarget: null }))
}

test('我的页面可以退出且重新进入受保护页面会回欢迎页', async ({ page }) => {
  let loggedIn = true
  await prepare(page)
  await page.unroute('**/v1/auth/session')
  await page.route('**/v1/auth/session', route => {
    if (route.request().method() === 'DELETE') { loggedIn = false; return route.fulfill({ status: 204 }) }
    return loggedIn ? json(route, 200, session) : json(route, 401, { error: { code: 'UNAUTHENTICATED', message: '请先登录' } })
  })
  await page.goto('/me')
  await expect(page.getByText('+86 138****5678')).toBeVisible()
  await expect(page.getByText('72.5 kg')).toBeVisible()
  await expect(page.getByText('2400 千卡')).toBeVisible()
  await page.getByRole('button', { name: '退出登录' }).click()
  await page.goto('/history')
  await expect(page).toHaveURL(/\/welcome$/)
})

test('修改目标明确显示次日生效并复用计算说明', async ({ page }) => {
  await prepare(page)
  await page.route('**/v1/goal-previews', route => json(route, 200, { preview: target }))
  await page.unroute('**/v1/goals')
  await page.route('**/v1/goals', route => route.request().method() === 'PUT' ? json(route, 200, { settings: { ...settings, revision: 3 }, effectiveTarget: target }) : json(route, 200, { settings, activeTarget: { ...target, localDate: '2026-09-11' }, pendingTarget: null }))
  await page.goto('/me/goals')
  await page.getByLabel('当前目标').selectOption('maintain')
  await page.getByRole('button', { name: '预览修改' }).click()
  await expect(page.getByText('2026-09-12 生效')).toBeVisible()
  await page.getByRole('button', { name: '如何计算' }).click()
  await expect(page.getByText('取整后的每日热量目标')).toBeVisible()
  await page.getByRole('button', { name: '确认保存' }).click()
  await expect(page.getByRole('status')).toContainText('2026-09-12')
})

test('短信验证和二次确认后删除账户并清空登录态', async ({ page }) => {
  await prepare(page)
  let deleted = false
  await page.unroute('**/v1/auth/session')
  await page.route('**/v1/auth/session', route => deleted ? json(route, 401, { error: { code: 'UNAUTHENTICATED', message: '请先登录' } }) : json(route, 200, session))
  await page.route('**/v1/auth/codes', route => json(route, 202, { verification: { requestId: 'verify_delete', expiresInSeconds: 300, retryAfterSeconds: 60 } }))
  await page.route('**/v1/account-deletions', route => { deleted = true; return json(route, 202, { accountDeletion: { status: 'accepted', accessRevokedAt: '2026-09-11T10:00:00+08:00', purgeBy: '2026-10-11T10:00:00+08:00' } }) })
  await page.goto('/me/delete-account')
  await page.getByLabel('完整手机号').fill('13812345678')
  await page.getByRole('button', { name: '发送删号验证码' }).click()
  await page.getByLabel('验证码').fill('123456')
  await page.getByLabel('确认文字').fill('DELETE')
  await page.getByRole('button', { name: '永久删除账户' }).click()
  await expect(page).toHaveURL(/\/welcome$/)
  for (const path of ['/today', '/history', '/me']) {
    await page.goto(path)
    await expect(page).toHaveURL(/\/welcome$/)
  }
})

test('资料修改携带 revision 并提示目标次日生效', async ({ page }) => {
  await prepare(page)
  await page.unroute('**/v1/profile')
  await page.route('**/v1/profile', route => {
    if (route.request().method() === 'PUT') {
      expect(route.request().postDataJSON()).toMatchObject({ weightKg: 75, expectedRevision: 3 })
      return json(route, 200, { profile: { ...profile, weightKg: 75, revision: 4 } })
    }
    return json(route, 200, { profile })
  })
  await page.goto('/me/profile')
  await page.getByLabel('体重（kg）').fill('75')
  await page.getByRole('button', { name: '保存资料' }).click()
  await expect(page.getByRole('status')).toContainText('下一个本地日期')
})
