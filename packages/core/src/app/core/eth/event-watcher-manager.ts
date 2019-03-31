import { EventWatcher as EthEventWatcher } from 'watch-eth'

import {
  EventWatcherManager,
  EventWatcher,
  EventWatcherOptions,
  EthClient,
  BaseDB,
} from '../../../interfaces'

export class DefaultEventWatcherManager implements EventWatcherManager {
  constructor(private ethClient: EthClient, private db: BaseDB) {}

  public create(options: EventWatcherOptions): EventWatcher {
    return new EthEventWatcher({
      ...options,
      eth: this.ethClient,
      db: this.db,
    })
  }
}
