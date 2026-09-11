import type {Nutrition} from './analyses'
export type MetricProgress={consumed:number;target:number;remaining:number;overBy:number;percent:number;status:'under'|'near'|'over'|'unavailable'}
export type DaySummary={localDate:string;target:Nutrition;totals:Nutrition;progress:{energy:MetricProgress;protein:MetricProgress;carb:MetricProgress;fat:MetricProgress};mealGroups:Array<{mealType:string;meals:Array<{id:string;occurredAt:string;totals:Nutrition}>}>}
export type DayResponse={day:DaySummary}
