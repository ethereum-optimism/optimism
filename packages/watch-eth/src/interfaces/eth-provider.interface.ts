/* Internal Imports */
import { EventFilterOptions } from './event-filter'

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
