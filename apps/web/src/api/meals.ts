import type {SaveMealInput,SaveMealResponse} from '@formtally/api-contract/meals'
import {http} from './http'
export function saveMeal(input:SaveMealInput,key:string):Promise<SaveMealResponse>{return http('/v1/meals',{method:'POST',headers:{'Idempotency-Key':key},body:JSON.stringify(input)})}
