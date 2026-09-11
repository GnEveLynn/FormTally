import type {DaySummary} from '@formtally/api-contract/days'
import {reactive} from 'vue'
import * as api from '../api/days'
export function createTodayStore(client:{getDay(date:string):Promise<DaySummary>}=api){const state=reactive<{status:'idle'|'loading'|'empty'|'ready'|'error';day:DaySummary|null;error:string}>({status:'idle',day:null,error:''});return{state,async load(date:string){state.status='loading';state.error='';try{state.day=await client.getDay(date);state.status=state.day.mealGroups.length?'ready':'empty'}catch(error){state.day=null;state.status='error';state.error=error instanceof Error?error.message:'加载失败'}}}}
export const todayStore=createTodayStore()
