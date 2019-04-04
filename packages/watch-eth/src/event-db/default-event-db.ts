import { BaseEventDB } from './base-event-db'

export class DefaultEventDB implements BaseEventDB {
  private lastLogged: { [key: string]: number } = {}
  private seen: { [key: string]: boolean } = {}

  /**
   * Returns the last logged block for an event.
   * @param event Event to query.
   * @returns last logged block for that event.
   */
  public async getLastLoggedBlock(event: string): Promise<number> {
    return this.lastLogged[event] || -1
  }

  /**
   * Sets the last logged block for an event.
   * @param event Event to set.
   * @param block Last logged block for that event.
   */
  public async setLastLoggedBlock(event: string, block: number): Promise<void> {
    this.lastLogged[event] = block
  }

  /**
   * Checks whether a given event has already been seen.
   * @param event Event to check
   * @returns `true` if the event has been seen, `false` otherwise.
   */
  public async getEventSeen(event: string): Promise<boolean> {
    return this.seen[event] || false
  }

  /**
   * Sets a given event as seen.
   * @param event Event to set.
   */
  public async setEventSeen(event: string): Promise<void> {
    this.seen[event] = true
  }
}
