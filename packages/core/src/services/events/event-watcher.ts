/* External Imports */
import { Service, OnStart, OnStop } from '@nestd/core'
import { EventWatcher } from 'watch-eth'

/* Services */
import { SyncDB } from '../db/interfaces/sync-db'
import { ETHProvider } from '../eth/eth-provider'
import { ContractProvider } from '../eth/contract-provider'
import { ConfigService } from '../config.service'
import { EventService } from '../event.service'

/* Internal Imports */
import { CONFIG } from '../../constants'

interface EventWatcherOptions {
  finalityDepth: number
  eventPollInterval: number
}

@Service()
export class EventWatcherService implements OnStart, OnStop {
  public watcher: EventWatcher
  private readonly name = 'eventWatcher'

  constructor(
    private readonly events: EventService,
    private readonly config: ConfigService,
    private readonly eth: ETHProvider,
    private readonly contract: ContractProvider,
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

  public subscribe(event: string): void {
    this.watcher.subscribe(event, (...args: any) => {
      this.events.event(this.name, event, args)
    })
  }

  private options(): EventWatcherOptions {
    return this.config.get(CONFIG.EVENT_WATCHER_OPTIONS)
  }
}
