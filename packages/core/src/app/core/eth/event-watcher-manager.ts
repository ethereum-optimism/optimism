import { EventWatcher } from '@pigi/watch-eth'
import { compiledPlasmaChain } from '@pigi/contracts'

import { MessageBus, EthClient } from '../../../interfaces'
import { BaseRunnable, BaseKey } from '../../common'
import { ChainDbHost } from '../db'

export class DefaultEventWatcher extends BaseRunnable {
  private watcher: EventWatcher

  constructor(
    private messageBus: MessageBus,
    private ethClient: EthClient,
    private chainDbHost: ChainDbHost
  ) {
    super()
  }

  public async onStart(): Promise<void> {
    this.messageBus.on('chaindb:ready', this.onChainDbReady.bind(this))
  }

  private onChainDbReady(address: string): void {
    const prefix = new BaseKey('p')
    const db = this.chainDbHost.db.bucket(prefix.encode())

    this.watcher = new EventWatcher({
      address,
      abi: compiledPlasmaChain.abi,
      eth: this.ethClient.web3,
      db,
    })

    const events = compiledPlasmaChain.abi.filter((item) => {
      return item.type === 'event'
    })
    for (const event of events) {
      this.watcher.on(event.name, (...args: any[]) => {
        this.messageBus.emit(`ethereum:event:${event.name}`, args)
      })
    }
  }
}
