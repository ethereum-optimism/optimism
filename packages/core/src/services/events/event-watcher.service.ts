/* External Imports */
import { Service, OnStart, OnStop } from '@nestd/core'
import { EventWatcher } from 'watch-eth'

/* Services */
import { SyncDB } from '../db/interfaces/sync-db'
import { EthDataService } from '../eth/eth-data.service'
import { ContractService } from '../eth/contract.service'
import { ConfigService } from '../config.service'
import { EventService } from '../event.service'

/* Internal Imports */
import { CONFIG } from '../../constants'

interface EventWatcherOptions {
  finalityDepth: number
  eventPollInterval: number
}

/**
 * Service that watches for events from Ethereum.
 */
@Service()
export class EventWatcherService implements OnStart, OnStop {
  public watcher: EventWatcher
  private readonly name = 'eventWatcher'

  constructor(
    private readonly events: EventService,
    private readonly config: ConfigService,
    private readonly eth: EthDataService,
    private readonly contract: ContractService,
    private readonly syncdb: SyncDB
  ) {}

  public async onStart(): Promise<void> {
    if (this.watcher) {
      return
    }

    const address = await this.contract.waitForAddress()
    const abi = this.contract.abi

    this.watcher = new EventWatcher({
      address,
      abi,
      eth: this.eth,
      db: this.syncdb,
      finalityDepth: this.options().finalityDepth,
      pollInterval: this.options().eventPollInterval,
    })
  }

  public async onStop(): Promise<void> {
    await this.watcher.stopPolling()
    this.watcher = undefined
  }

  /**
   * Subscribes to an event.
   * Event will be emitted to the global log.
   * @param event Name of the event to subscribe to.
   */
  public subscribe(event: string): void {
    this.watcher.subscribe(event, (...args: any) => {
      this.events.event(this.name, event, args)
    })
  }

  /**
   * @returns any event watcher options.
   */
  private options(): EventWatcherOptions {
    return this.config.get(CONFIG.EVENT_WATCHER_OPTIONS)
  }
}
