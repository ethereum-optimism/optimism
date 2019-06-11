/* Internal Imports */
import { EventDB } from './interfaces'

/**
 * Basic EventDB implementation.
 */
export class DefaultEventDB implements EventDB {
  constructor(private db: any) {}

  /**
   * Returns the last logged block for an event.
   * @param event Event to query.
   * @returns last logged block for that event.
   */
  public async getLastLoggedBlock(event: string): Promise<number> {
    return this.db.get(event)
  }

  /**
   * Sets the last logged block for an event.
   * @param event Event to set.
   * @param block Last logged block for that event.
   */
  public async setLastLoggedBlock(event: string, block: number): Promise<void> {
    await this.db.put(event, block)
  }

  /**
   * Checks whether a given event has already been seen.
   * @param event Event to check
   * @returns `true` if the event has been seen, `false` otherwise.
   */
  public async getEventSeen(event: string): Promise<boolean> {
    return this.db.get(`seen:${event}`)
  }

  /**
   * Sets a given event as seen.
   * @param event Event to set.
   */
  public async setEventSeen(event: string): Promise<void> {
    await this.db.put(`seen:${event}`, true)
  }
}
