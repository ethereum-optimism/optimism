import { EventWatcherOptions, EventWatcher } from '../../common'

/**
 * EventWatcherManager creates new EventWatcher instances.
 */
export interface EventWatcherManager {
  /**
   * Creates a new EventWatcher instance.
   * Should cache instances as to not create duplicates.
   * @param options Parameters to the EventWatcher.
   * @returns the EventWatcher instance.
   */
  create(options: EventWatcherOptions): EventWatcher
}
