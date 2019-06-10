/* Internal Imports */
import { EventFilterOptions } from './event-filter-options.interface'
import { EventLog } from './event-log.interface'

export interface FullEventFilter extends EventFilterOptions {
  address: string
  abi: any
  fromBlock: number
  toBlock: number
}

export interface EthProvider {
  connected(): Promise<boolean>
  getCurrentBlock(): Promise<number>
  getEvents(filter: FullEventFilter): Promise<EventLog[]>
}
