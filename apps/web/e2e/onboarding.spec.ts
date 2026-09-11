import { expect, test, type Route } from '@playwright/test'

const session = { session: { expiresAt: '2026-10-10T10:00:00+08:00' }, user: { id: 'user_new', phoneMasked: '+86 138****5678', onboardingStatus: 'profile_required' }, consents: { termsVersion: '2026-09-10', privacyVersion: '2026-09-10', aiImageProcessingVersion: null, currentAiImageProcessingVersion: '2026-09-10' } }
const calculation = { calculationVersion: 'daily_nutrition_target_v1', method: { id: 'mifflin_st_jeor_v1', displayName: 'Mifflin–St Jeor 静息能量估算', sourceUrl: 'https://pubmed.ncbi.nlm.nih.gov/2305711/', formulaExpression: '10 × 体重kg + 6.25 × 身高cm - 5 × 年龄 + 5' }, macroMethod: { id: 'macro_split_50_25_25_v1', carbPercent: 50, proteinPercent: 25, fatPercent: 25 }, inputs: { biologicalSex: 'male', ageYears: 31, heightCm: 178, weightKg: 72.5, activityLevel: 'moderate', activityMultiplier: 1.55, objective: 'fat_loss', pace: 'standard', goalAdjustmentPercent: -15 }, steps: [{ key: 'finalEnergyTarget', label: '取整后的每日热量目标', value: 2220, unit: 'kcal/day' }], rounding: { energy: 'nearest_10_kcal', macros: 'nearest_1_gram', halfRule: 'half_away_from_zero' }, disclaimer: '该结果是估算起点，不构成医学建议。' }
const effectiveTarget = { effectiveFrom: '2026-09-11', target: { energyKcal: 2220, proteinGrams: 139, carbGrams: 278, fatGrams: 62 }, calculation, warnings: [] }
const json = (route: Route, body: unknown) => route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(body) })

test('新用户填写资料、查看计算并保存目标后进入今日页', async ({ page }) => {
  await page.route('**/v1/auth/session', route => json(route, session))
  await page.route('**/v1/profile', route => json(route, { profile: { biologicalSex: 'male', birthDate: '1995-06-18', heightCm: 178, weightKg: 72.5, activityLevel: 'moderate', timezone: 'Asia/Shanghai', healthContext: { pregnant: false, breastfeeding: false, clinicalDietRequired: false }, automaticGoalEligible: true, revision: 1, updatedAt: '2026-09-11T10:00:00+08:00' } }))
  await page.route('**/v1/goal-previews', route => json(route, { preview: effectiveTarget }))
  await page.route('**/v1/goals', route => json(route, { settings: { mode: 'automatic', automatic: { objective: 'fat_loss', pace: 'standard' }, manual: null, revision: 1, updatedAt: '2026-09-11T10:00:00+08:00' }, effectiveTarget }))
  await page.route('**/v1/days/*', route => route.fulfill({ status: 503, contentType: 'application/json', body: JSON.stringify({ error: { code: 'DEPENDENCY_UNAVAILABLE', message: '测试中未加载日期数据', requestId: 'req_day' } }) }))

  await page.goto('/')
  await expect(page).toHaveURL(/\/profile$/)
  await page.getByLabel('出生日期').fill('1995-06-18')
  await page.getByLabel('身高（cm）').fill('178')
  await page.getByLabel('体重（kg）').fill('72.5')
  await page.getByRole('button', { name: '下一步' }).click()
  await expect(page).toHaveURL(/\/goals$/)
  await page.getByRole('button', { name: '预览目标' }).click()
  await expect(page).toHaveURL(/\/goals\/result$/)
  await expect(page.getByText('2220', { exact: false }).first()).toBeVisible()
  await page.getByRole('button', { name: '如何计算' }).click()
  await expect(page.getByText('取整后的每日热量目标')).toBeVisible()
  await page.getByRole('button', { name: '确认并进入今天' }).click()
  await expect(page).toHaveURL(/\/today$/)
})
