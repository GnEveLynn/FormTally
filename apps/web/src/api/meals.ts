import type {DeleteMealResponse,Meal,SaveMealInput,SaveMealResponse,UpdateMealInput} from '@formtally/api-contract/meals'
import {http} from './http'
export function saveMeal(input:SaveMealInput,key:string):Promise<SaveMealResponse>{return http('/v1/meals',{method:'POST',headers:{'Idempotency-Key':key},body:JSON.stringify(input)})}
export async function getMeal(id:string):Promise<Meal>{return (await http<{meal:Meal}>(`/v1/meals/${id}`)).meal}
export function updateMeal(id:string,input:UpdateMealInput):Promise<SaveMealResponse>{return http(`/v1/meals/${id}`,{method:'PATCH',body:JSON.stringify(input)})}
export async function removeMealImage(id:string,expectedRevision:number):Promise<Meal>{return (await http<{meal:Meal}>(`/v1/meals/${id}/image?expectedRevision=${expectedRevision}`,{method:'DELETE'})).meal}
export function deleteMeal(id:string,expectedRevision:number):Promise<DeleteMealResponse>{return http(`/v1/meals/${id}?expectedRevision=${expectedRevision}`,{method:'DELETE'})}
