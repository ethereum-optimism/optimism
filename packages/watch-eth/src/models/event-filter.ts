/* Internal Imports */
import { hash } from '../utils'
import { EventFilterOptions } from '../interfaces'

/**
 * Represents an event filter.
 */
export class EventFilter {
  public options: EventFilterOptions

  constructor(options: EventFilterOptions) {
    this.options = options
  }

  /**
   * @returns the unique hash for this filter.
   */
  get hash(): string {
    return hash(JSON.stringify(this.options))
  }
}
