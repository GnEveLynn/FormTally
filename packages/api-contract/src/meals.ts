import type {MealItem,Nutrition} from './analyses'
export type Meal={id:string;occurredAt:string;localDate:string;mealType:string;image:{url:string;expiresAt:string;width:number;height:number;mimeType:string}|null;items:MealItem[];totals:Nutrition;estimateNotice:string;revision:number;createdAt:string;updatedAt:string}
export type SaveMealInput={analysisId:string|null;occurredAt:string;mealType:string;items:MealItem[]}
export type SaveMealResponse={meal:Meal;affectedLocalDates:string[]}
export type UpdateMealInput={expectedRevision:number;occurredAt?:string;mealType?:string;items?:MealItem[]}
export type DeleteMealResponse={deletedMealId:string;affectedLocalDates:string[]}
