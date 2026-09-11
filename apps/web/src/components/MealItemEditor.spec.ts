import {mount} from '@vue/test-utils'
import {describe,expect,it} from 'vitest'
import MealItemEditor from './MealItemEditor.vue'
const item={name:'鸡胸肉',grams:100,nutrition:{energyKcal:165,proteinGrams:31,carbGrams:0,fatGrams:3.6},basisPer100Grams:{energyKcal:165,proteinGrams:31,carbGrams:0,fatGrams:3.6},origin:'ai' as const,confidence:'low' as const,assumption:'按熟制估算',draftItemId:'draft_1'}
describe('MealItemEditor',()=>{it('rescales nutrition immediately when grams change',async()=>{const wrapper=mount(MealItemEditor,{props:{items:[item]}});await wrapper.get('input[name="grams-0"]').setValue('120');const emitted=wrapper.emitted('update:items')?.at(-1)?.[0] as any[];expect(emitted[0].nutrition).toEqual({energyKcal:198,proteinGrams:37.2,carbGrams:0,fatGrams:4.3});expect(wrapper.text()).toContain('低置信度')})})
