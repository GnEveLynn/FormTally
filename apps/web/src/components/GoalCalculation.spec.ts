import { mount } from '@vue/test-utils'
import { expect, it } from 'vitest'
import GoalCalculation from './GoalCalculation.vue'

it('默认折叠并按后端说明展开计算步骤', async () => {
  const wrapper = mount(GoalCalculation, { props: { calculation: {
    calculationVersion: 'daily_nutrition_target_v1',
    method: { id: 'mifflin_st_jeor_v1', displayName: 'Mifflin–St Jeor 静息能量估算', sourceUrl: 'https://pubmed.ncbi.nlm.nih.gov/2305711/', formulaExpression: '服务端公式' },
    macroMethod: { id: 'macro_split_50_25_25_v1', carbPercent: 50, proteinPercent: 25, fatPercent: 25 },
    inputs: { biologicalSex: 'male', ageYears: 31, heightCm: 178, weightKg: 72.5, activityLevel: 'moderate', activityMultiplier: 1.55, objective: 'fat_loss', pace: 'standard', goalAdjustmentPercent: -15 },
    steps: [{ key: 'finalEnergyTarget', label: '取整后的每日热量目标', value: 2220, unit: 'kcal/day' }],
    rounding: { energy: 'nearest_10_kcal', macros: 'nearest_1_gram', halfRule: 'half_away_from_zero' }, disclaimer: '估算起点，不构成医学建议。',
  } } })
  expect(wrapper.text()).not.toContain('取整后的每日热量目标')
  await wrapper.get('button').trigger('click')
  expect(wrapper.text()).toContain('取整后的每日热量目标')
  expect(wrapper.text()).toContain('2220')
})
