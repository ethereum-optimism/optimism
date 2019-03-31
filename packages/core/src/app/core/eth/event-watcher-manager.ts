import { EventWatcher as EthEventWatcher } from 'watch-eth'

import {
  EventWatcherManager,
  EventWatcher,
  EventWatcherOptions,
  EthClient,
  BaseDB,
} from '../../../interfaces'

/**
 * Default EventWatcherManager implementation that uses our `watch-eth`
 * library under the hood.
 */
export class DefaultEventWatcherManager implements EventWatcherManager {
  constructor(private ethClient: EthClient, private db: BaseDB) {}

  /**
   * Creates a new `watch-eth` EventWatcher instance.
   * @param options Options for the watcher.
   * @returns the watcher instance.
   */
  public create(options: EventWatcherOptions): EventWatcher {
    return new EthEventWatcher({
      ...options,
      eth: this.ethClient,
      db: this.db,
    })
  }
}
