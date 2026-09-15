import type { Analysis, MealItem } from '@formtally/api-contract/analyses'
import { reactive } from 'vue'
import * as analyses from '../api/analyses'
import * as meals from '../api/meals'

const timezoneOffset = () => {
  const value = -new Date().getTimezoneOffset()
  const sign = value >= 0 ? '+' : '-'
  const absolute = Math.abs(value)
  return `${sign}${String(Math.floor(absolute / 60)).padStart(2, '0')}:${String(absolute % 60).padStart(2, '0')}`
}
const localDateTime = () => {
  const date = new Date()
  date.setMinutes(date.getMinutes() - date.getTimezoneOffset())
  return date.toISOString().slice(0, 19) + timezoneOffset()
}

type Client = {
  createAnalysis: typeof analyses.createAnalysis
  retryAnalysis?: typeof analyses.retryAnalysis
  saveMeal: typeof meals.saveMeal
}

const defaultClient = { createAnalysis: analyses.createAnalysis, retryAnalysis: analyses.retryAnalysis, saveMeal: meals.saveMeal }

export function createMealDraftStore(client: Client = defaultClient) {
  const state = reactive<{
    image: File | null
    occurredAt: string
    mealType: string
    description: string
    mode: 'ai' | 'manual'
    consentVersion: string
    status: string
    analysis: Analysis | null
    items: MealItem[]
    error: string
    analyzeKey: string
    retryKey: string
    saveKey: string
  }>({ image: null, occurredAt: localDateTime(), mealType: 'lunch', description: '', mode: 'ai', consentVersion: '', status: 'idle', analysis: null, items: [], error: '', analyzeKey: '', retryKey: '', saveKey: '' })

  const applyAnalysis = (analysis: Analysis) => {
    state.analysis = analysis
    state.items = [...analysis.items]
    state.status = analysis.status === 'failed' ? 'failed' : 'ready'
    state.error = analysis.failure?.message ?? ''
  }

  return {
    state,
    selectImage(file: File) {
      state.image = file
      state.analysis = null
      state.items = []
      state.analyzeKey = ''
      state.retryKey = ''
      state.status = 'selected'
      state.error = ''
    },
    async analyze() {
      if (!state.image) return
      state.status = 'analyzing'
      state.error = ''
      try {
        if (state.analysis?.status === 'failed' && client.retryAnalysis) {
          if (!state.retryKey) state.retryKey = crypto.randomUUID()
          const response = await client.retryAnalysis(state.analysis.id, { aiConsentVersion: state.consentVersion, expectedRevision: state.analysis.revision, ...(state.description.trim() ? { description: state.description.trim() } : {}) }, state.retryKey)
          applyAnalysis(response.analysis)
          return
        }
        if (!state.analyzeKey) state.analyzeKey = crypto.randomUUID()
        const response = await client.createAnalysis({ image: state.image, processingMode: state.mode, occurredAt: state.occurredAt, mealType: state.mealType, description: state.description.trim() || undefined, aiConsentVersion: state.consentVersion || undefined }, state.analyzeKey)
        applyAnalysis(response.analysis)
      } catch (error) {
        state.status = 'failed'
        state.error = error instanceof Error ? error.message : '分析失败'
      }
    },
    async save() {
      if (!state.saveKey) state.saveKey = crypto.randomUUID()
      state.status = 'saving'
      state.error = ''
      try {
        const result = await client.saveMeal({ analysisId: state.analysis?.id ?? null, occurredAt: state.occurredAt, mealType: state.mealType, items: state.items }, state.saveKey)
        state.status = 'saved'
        return result
      } catch (error) {
        state.status = 'save_failed'
        state.error = error instanceof Error ? error.message : '保存失败'
      }
    },
    clear() {
      state.image = null
      state.analysis = null
      state.items = []
      state.description = ''
      state.status = 'idle'
      state.error = ''
      state.analyzeKey = ''
      state.retryKey = ''
      state.saveKey = ''
    },
  }
}

export const mealDraftStore = createMealDraftStore()
