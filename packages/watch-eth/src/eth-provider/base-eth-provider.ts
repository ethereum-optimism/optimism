import { EventLog } from '../models/event-log'
import { EventFilterOptions } from '../models'

export interface FullEventFilter extends EventFilterOptions {
  address: string
  abi: any
  fromBlock: number
  toBlock: number
}

export interface BaseEthProvider {
  connected(): Promise<boolean>
  getCurrentBlock(): Promise<number>
  getEvents(filter: FullEventFilter): Promise<EventLog[]>
}
