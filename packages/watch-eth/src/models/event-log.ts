/* Internal Imports */
import { hash } from '../utils'
import { EventLog, EventLogData } from '../interfaces'

/**
 * Represents a single event log.
 */
export class DefaultEventLog implements EventLog {
  public data: EventLogData

  constructor(data: EventLogData) {
    this.data = data
  }

  /**
   * Returns a unique hash for this event log.
   */
  public getHash(): string {
    return hash(this.data.transactionHash + this.data.logIndex)
  }
}
