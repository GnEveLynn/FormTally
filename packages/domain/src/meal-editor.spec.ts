import { describe, expect, it } from 'vitest'
import cases from '../testdata/meal-editor-cases.json'
import { scaleNutrition, sumNutrition, validateMeal } from './meal-editor'

describe('meal editor',()=>{
  it.each(cases)('$name',({basis,grams,want})=>{expect(scaleNutrition(basis,grams)).toEqual(want)})
  it('uses direct edits when summing',()=>{expect(sumNutrition([{energyKcal:100,proteinGrams:2.2,carbGrams:3.3,fatGrams:4.4},{energyKcal:50,proteinGrams:1.1,carbGrams:2.2,fatGrams:3.3}])).toEqual({energyKcal:150,proteinGrams:3.3,carbGrams:5.5,fatGrams:7.7})})
  it('locates invalid fields before save',()=>{expect(validateMeal({occurredAt:'2099-01-01T00:00:00Z',mealType:'lunch',items:[]})).toEqual({'occurredAt':'餐食时间不能位于未来','items':'至少添加一个食物'})})
})
