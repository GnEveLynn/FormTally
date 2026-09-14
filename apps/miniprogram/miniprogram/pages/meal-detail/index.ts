import type { Meal } from '@formtally/api-contract/meals'
import { ApiError } from '../../services/http'
import * as meals from '../../services/meals'
import { historyStore } from '../../stores/history'

Page({
  data: { id: '', meal: null as Meal | null, status: 'loading', error: '', busy: false },
  onLoad(query: Record<string, string | undefined>) { this.setData({ id: query.id ?? '' }); void this.load() },
  async load() { this.setData({ status: 'loading', error: '' }); try { this.setData({ meal: await meals.getMeal(this.data.id), status: 'ready' }) } catch (error) { this.setData({ status: 'error', error: error instanceof Error ? error.message : '加载失败' }) } },
  edit() { wx.navigateTo({ url: `/pages/meal-edit/index?id=${encodeURIComponent(this.data.id)}` }) },
  removeImage() { this.confirm('仅移除图片？', async () => { const meal = this.data.meal; if (meal) this.setData({ meal: await meals.removeMealImage(meal.id, meal.revision) }) }) },
  removeMeal() { this.confirm('删除整餐？', async () => { const meal = this.data.meal; if (!meal) return; const result = await meals.deleteMeal(meal.id, meal.revision); await historyStore.refreshAfterMutation(result.affectedLocalDates); wx.switchTab({ url: '/pages/history/index' }) }) },
  confirm(content: string, operation: () => Promise<void>) { wx.showModal({ title: '请确认', content, confirmColor: '#a22b2b', success: async ({ confirm }) => { if (!confirm) return; this.setData({ busy: true, error: '' }); try { await operation() } catch (error) { if (error instanceof ApiError && error.code === 'REVISION_CONFLICT') { this.setData({ error: '数据已变化，已重新载入最新内容' }); await this.load() } else this.setData({ error: error instanceof Error ? error.message : '操作失败' }) } finally { this.setData({ busy: false }) } } }) },
})
