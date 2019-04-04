import { hash } from '../utils'

export interface EventLogData {
  transactionHash: string
  logIndex: number
}

/**
 * Represents a single event log.
 */
export class EventLog {
  public data: EventLogData

  constructor(data: EventLogData) {
    this.data = data
  }

  /**
   * Returns a unique hash for this event log.
   */
  get hash(): string {
    return hash(this.data.transactionHash + this.data.logIndex)
  }
}
