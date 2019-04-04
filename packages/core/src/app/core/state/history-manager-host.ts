import { MessageBus } from '../../../interfaces'
import { BaseRunnable } from '../../common'
import { ChainDbHost } from '../db/chain-db-host'
import { BaseKey } from '../../common/db'
import { PGHistoryManager } from './history-manager'

export class PGHistoryManagerHost extends BaseRunnable {
  private _historyManager: PGHistoryManager

  constructor(
    private messageBus: MessageBus,
    private chainDbHost: ChainDbHost
  ) {
    super()
  }

  get historyManager(): PGHistoryManager {
    return this._historyManager
  }

  public async onStart(): Promise<void> {
    this.messageBus.on('chaindb:ready', this.onChainDbReady.bind(this))
  }

  private onChainDbReady(): void {
    const prefix = new BaseKey('h')
    const db = this.chainDbHost.db.bucket(prefix.encode())
    this._historyManager = new PGHistoryManager(db)
  }
}
