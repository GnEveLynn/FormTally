import type { ProfileInput } from '@formtally/api-contract/profile'
import { validateProfile } from '../../domain/profile-validation'
import * as profileApi from '../../services/profile'
import { accountStore } from '../../stores/account'

const sex = ['male', 'female'] as const, activity = ['sedentary', 'light', 'moderate', 'high', 'very_high'] as const
Page({
  data: { form: null as ProfileInput | null, error: '', message: '', busy: false, sexLabels: ['男性', '女性'], activityLabels: ['久坐', '轻度', '中度', '高度', '极高'] },
  async onLoad() { try { const result = await profileApi.getProfile(); if (result.profile) this.setData({ form: { ...result.profile, healthContext: { ...result.profile.healthContext }, expectedRevision: result.profile.revision } }) } catch (error) { this.setData({ error: error instanceof Error ? error.message : '加载失败' }) } },
  field(event: { currentTarget: { dataset: { field?: string } }; detail: { value: string } }) { const form = this.data.form; const field = event.currentTarget.dataset.field; if (!form || !field) return; (form as unknown as Record<string, unknown>)[field] = ['heightCm', 'weightKg'].includes(field) ? Number(event.detail.value) : event.detail.value; this.setData({ form }) },
  sex(event: { detail: { value: string } }) { const form = this.data.form; if (form) { form.biologicalSex = sex[Number(event.detail.value)] ?? 'male'; this.setData({ form }) } },
  activity(event: { detail: { value: string } }) { const form = this.data.form; if (form) { form.activityLevel = activity[Number(event.detail.value)] ?? 'moderate'; this.setData({ form }) } },
  health(event: { detail: { value: string[] } }) { const form = this.data.form; if (form) { const values = event.detail.value; form.healthContext = { pregnant: values.includes('pregnant'), breastfeeding: values.includes('breastfeeding'), clinicalDietRequired: values.includes('clinicalDietRequired') }; this.setData({ form }) } },
  async save() { const form = this.data.form; if (!form) return; const errors = validateProfile(form); if (Object.keys(errors).length) { this.setData({ error: Object.values(errors)[0], message: '' }); return } this.setData({ busy: true }); const ok = await accountStore.saveProfile(form); this.setData({ busy: false, error: accountStore.state.error, message: ok ? '资料已保存；自动目标变化从下一个本地日期生效。' : '' }) },
})
