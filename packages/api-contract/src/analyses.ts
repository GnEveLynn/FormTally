export type Nutrition={energyKcal:number;proteinGrams:number;carbGrams:number;fatGrams:number}
export type MealItem={id?:string|null;draftItemId:string|null;name:string;grams:number;nutrition:Nutrition;basisPer100Grams:Nutrition|null;origin:'ai'|'ai_modified'|'manual';confidence:'high'|'medium'|'low'|null;assumption:string|null}
export type Analysis={id:string;status:'processing'|'review_required'|'failed'|'saved';processingMode:'ai'|'manual';occurredAt:string;localDate:string;mealType:string;image:{url:string;expiresAt:string;width:number;height:number;mimeType:string};items:MealItem[];warnings:string[];failure:{code:string;message:string;retryable:boolean}|null;mealId:string|null;expiresAt:string;revision:number;createdAt:string}
export type AnalysisResponse={analysis:Analysis}
