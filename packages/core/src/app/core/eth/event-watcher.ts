import { EventWatcher } from '@pigi/watch-eth'
import { compiledPlasmaChain } from '@pigi/contracts'

import { MessageBus, EthClient, KeyValueStore } from '../../../interfaces'

export class DefaultEventWatcher {
  private cache: Record<string, boolean>
  private watcher: EventWatcher

  constructor(
    private messageBus: MessageBus,
    private ethClient: EthClient,
    private db: KeyValueStore
  ) {}

  public watch(address: string, abi: any): void {
    if (address in this.cache) {
      return
    }

    this.watcher = new EventWatcher({
      address,
      abi,
      eth: this.ethClient.web3,
      db: this.db,
    })

    const events = compiledPlasmaChain.abi.filter((item) => {
      return item.type === 'event'
    })
    for (const event of events) {
      this.watcher.on(event.name, (...args: any[]) => {
        this.messageBus.emit(`ethereum:event:${event.name}`, args)
      })
    }

    this.cache[address] = true
  }
}
