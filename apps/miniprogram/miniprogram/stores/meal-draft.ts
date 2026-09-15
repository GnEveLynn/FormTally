import type { Analysis, AnalysisResponse, MealItem } from '@formtally/api-contract/analyses'
import type { SaveMealInput, SaveMealResponse } from '@formtally/api-contract/meals'
import { validateMeal } from '../domain/meal-editor'
import type { ProcessedImage } from '../platform/media'
import * as analysesApi from '../services/analyses'
import * as mealsApi from '../services/meals'
import { uploadAnalysis } from '../services/upload'

type DraftStatus = 'idle' | 'selected' | 'analyzing' | 'ready' | 'failed' | 'saving' | 'save_failed' | 'saved'

interface MealDraftClient {
  uploadAnalysis: typeof uploadAnalysis
  retryAnalysis(id: string, input: { aiConsentVersion: string; expectedRevision: number; description?: string }, key: string): Promise<AnalysisResponse>
  saveMeal(input: SaveMealInput, key: string): Promise<SaveMealResponse>
}

export interface MealDraftState {
  image: ProcessedImage | null
  occurredAt: string
  mealType: string
  description: string
  mode: 'ai' | 'manual'
  consentVersion: string
  status: DraftStatus
  progress: number
  analysis: Analysis | null
  items: MealItem[]
  warnings: string[]
  error: string
  errors: Record<string, string>
}

const defaultClient: MealDraftClient = { uploadAnalysis, retryAnalysis: analysesApi.retryAnalysis, saveMeal: mealsApi.saveMeal }

function localDateTime(date = new Date()): string {
  const offset = -date.getTimezoneOffset()
  const sign = offset >= 0 ? '+' : '-'
  const absolute = Math.abs(offset)
  const local = new Date(date.getTime() + offset * 60_000).toISOString().slice(0, 19)
  return `${local}${sign}${String(Math.floor(absolute / 60)).padStart(2, '0')}:${String(absolute % 60).padStart(2, '0')}`
}

function idempotencyKey(): Promise<string> {
  return new Promise((resolve, reject) => wx.getRandomValues({
    length: 16,
    success: ({ randomValues }) => resolve(Array.from(new Uint8Array(randomValues), (value) => value.toString(16).padStart(2, '0')).join('')),
    fail: () => reject(new Error('无法安全创建请求，请重试')),
  }))
}

export function emptyMealItem(): MealItem {
  return {
    draftItemId: null, name: '', grams: 100,
    nutrition: { energyKcal: 0, proteinGrams: 0, carbGrams: 0, fatGrams: 0 },
    basisPer100Grams: null, origin: 'manual', confidence: null, assumption: null,
  }
}

export function createMealDraftStore(client: MealDraftClient = defaultClient, createKey: () => Promise<string> = idempotencyKey) {
  const state: MealDraftState = {
    image: null, occurredAt: localDateTime(), mealType: 'lunch', description: '', mode: 'ai', consentVersion: '', status: 'idle', progress: 0,
    analysis: null, items: [], warnings: [], error: '', errors: {},
  }
  let analyzeKey = ''
  let retryKey = ''
  let saveKey = ''
  let analyzePending: Promise<void> | null = null
  let savePending: Promise<SaveMealResponse | undefined> | null = null
  let uploadTask: ReturnType<typeof uploadAnalysis> | null = null
  const listeners = new Set<() => void>()
  const notify = () => listeners.forEach((listener) => listener())

  function applyAnalysis(analysis: Analysis) {
    state.analysis = analysis
    state.items = analysis.items.map((item) => ({ ...item, nutrition: { ...item.nutrition }, basisPer100Grams: item.basisPer100Grams ? { ...item.basisPer100Grams } : null }))
    state.warnings = [...analysis.warnings]
    state.status = analysis.status === 'failed' ? 'failed' : 'ready'
    state.error = analysis.failure?.message ?? ''
    notify()
  }

  function reset() {
    uploadTask?.cancel()
    uploadTask = null
    analyzeKey = ''
    retryKey = ''
    saveKey = ''
    Object.assign(state, {
      image: null, occurredAt: localDateTime(), mealType: 'lunch', description: '', mode: 'ai', consentVersion: '', status: 'idle', progress: 0,
      analysis: null, items: [], warnings: [], error: '', errors: {},
    })
    notify()
  }

  return {
    state,
    subscribe(listener: () => void) {
      listeners.add(listener)
      return () => listeners.delete(listener)
    },
    selectImage(image: ProcessedImage) {
      state.image = image
      state.analysis = null
      state.items = []
      state.warnings = []
      state.status = 'selected'
      state.progress = 0
      state.error = ''
      state.errors = {}
      analyzeKey = ''
      retryKey = ''
      saveKey = ''
      notify()
    },
    startManual() {
      state.mode = 'manual'
      state.consentVersion = ''
      state.analysis = null
      state.warnings = []
      state.status = 'ready'
      state.error = ''
      if (!state.items.length) state.items = [emptyMealItem()]
      notify()
    },
    analyze(): Promise<void> {
      if (analyzePending) return analyzePending
      const image = state.image
      if (!image) return Promise.resolve()
      analyzePending = (async () => {
        state.status = 'analyzing'
        state.progress = 0
        state.error = ''
        notify()
        try {
          if (state.analysis?.status === 'failed') {
            retryKey ||= await createKey()
            const response = await client.retryAnalysis(state.analysis.id, {
              aiConsentVersion: state.consentVersion,
              expectedRevision: state.analysis.revision,
              ...(state.description.trim() ? { description: state.description.trim() } : {}),
            }, retryKey)
            applyAnalysis(response.analysis)
            retryKey = ''
          } else {
            analyzeKey ||= await createKey()
            uploadTask = client.uploadAnalysis({
              imagePath: image.path,
              processingMode: 'ai',
              occurredAt: state.occurredAt,
              mealType: state.mealType,
              description: state.description.trim() || undefined,
              aiConsentVersion: state.consentVersion || undefined,
              idempotencyKey: analyzeKey,
            }, (progress) => { state.progress = progress; notify() })
            const response = await uploadTask.promise
            uploadTask = null
            applyAnalysis(response.analysis)
            analyzeKey = ''
          }
        } catch (error) {
          state.status = 'failed'
          state.error = error instanceof Error ? error.message : '分析失败，请重试'
          notify()
        }
      })().finally(() => { analyzePending = null })
      return analyzePending
    },
    save(now = new Date()): Promise<SaveMealResponse | undefined> {
      if (savePending) return savePending
      state.errors = validateMeal({ occurredAt: state.occurredAt, mealType: state.mealType, items: state.items }, now)
      if (Object.keys(state.errors).length) {
        notify()
        return Promise.resolve(undefined)
      }
      savePending = (async () => {
        state.status = 'saving'
        state.error = ''
        notify()
        try {
          saveKey ||= await createKey()
          const result = await client.saveMeal({ analysisId: state.analysis?.id ?? null, occurredAt: state.occurredAt, mealType: state.mealType, items: state.items }, saveKey)
          state.status = 'saved'
          saveKey = ''
          notify()
          return result
        } catch (error) {
          state.status = 'save_failed'
          state.error = error instanceof Error ? error.message : '保存失败，请重试'
          notify()
          return undefined
        }
      })().finally(() => { savePending = null })
      return savePending
    },
    abandon: reset,
    clear: reset,
  }
}

export const mealDraftStore = createMealDraftStore()
