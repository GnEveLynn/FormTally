import type {DaySummary,DayResponse} from '@formtally/api-contract/days'
import {http} from './http'
export async function getDay(date:string):Promise<DaySummary>{return (await http<DayResponse>(`/v1/days/${date}`)).day}
