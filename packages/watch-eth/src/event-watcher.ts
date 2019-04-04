import { EventEmitter } from 'events'
import { sleep } from './utils'
import { EventFilter, EventFilterOptions, EventLog } from './models'
import { BaseEthProvider } from './eth-provider/base-eth-provider'
import { BaseEventDB } from './event-db/base-event-db'
import { DefaultEventDB } from './event-db/default-event-db'
import { DefaultEthProvider } from './eth-provider/default-eth-provider'

interface EventSubscription {
  filter: EventFilter
  listeners: Array<(...args: any) => any>
}

export interface EventWatcherOptions {
  address: string
  abi: any
  finalityDepth?: number
  pollInterval?: number
  eth?: BaseEthProvider
  db?: BaseEventDB
}

const defaultOptions = {
  finalityDepth: 12,
  pollInterval: 10000,
}

/**
 * Watches for events on a given contract.
 */
export class EventWatcher extends EventEmitter {
  private options: EventWatcherOptions
  private eth: BaseEthProvider
  private db: BaseEventDB
  private polling = false
  private subscriptions: { [key: string]: EventSubscription } = {}

  constructor(options: EventWatcherOptions) {
    super()

    options = {
      ...defaultOptions,
      ...options,
    }

    this.eth = options.eth || new DefaultEthProvider()
    this.db = options.db || new DefaultEventDB()
    this.options = options
  }

  /**
   * @returns `true` if polling, `false` otherwise.
   */
  get isPolling(): boolean {
    return this.polling
  }

  /**
   * Starts the polling loop.
   * Can only be called once.
   */
  public startPolling(): void {
    if (this.polling) {
      return
    }

    this.polling = true
    this.pollEvents()
  }

  /**
   * Stops the polling loop.
   */
  public stopPolling(): void {
    this.polling = false
  }

  /**
   * Subscribes to an event with a given callback.
   * @param options Event filter to subscribe to.
   * @param listener Function to be called when the event is triggered.
   */
  public subscribe(
    options: EventFilterOptions | string,
    listener: (...args: any) => any
  ): void {
    // Start polling if we haven't already.
    this.startPolling()

    const filter =
      typeof options === 'string'
        ? new EventFilter({
            event: options,
          })
        : new EventFilter(options)

    // Initialize the subscriber if it doesn't exist.
    if (!(filter.hash in this.subscriptions)) {
      this.subscriptions[filter.hash] = {
        filter,
        listeners: [],
      }
    }

    // Register the event.
    this.subscriptions[filter.hash].listeners.push(listener)
  }

  /**
   * Unsubscribes from an event with a given callback.
   * @param options Event filter to unsubscribe from.
   * @param listener Function that was used to subscribe.
   */
  public unsubscribe(
    options: EventFilterOptions | string,
    listener: (...args: any) => any
  ): void {
    const filter =
      typeof options === 'string'
        ? new EventFilter({
            event: options,
          })
        : new EventFilter(options)
    const subscription = this.subscriptions[filter.hash]

    // Can't unsubscribe if we aren't subscribed in the first place.
    if (subscription === undefined) {
      return
    }

    // Remove the listener.
    subscription.listeners = subscription.listeners.filter((l) => {
      return l !== listener
    })

    // No more listeners on this event, can remove the filter.
    if (subscription.listeners.length === 0) {
      delete this.subscriptions[filter.hash]
    }

    // No more subscriptions, can stop polling.
    if (Object.keys(this.subscriptions).length === 0) {
      this.polling = false
    }
  }

  /**
   * Polling loop.
   * Checks events then sleeps before calling itself again.
   * Stops polling if the service is stopped.
   */
  private async pollEvents(): Promise<void> {
    if (!this.polling) {
      return
    }

    try {
      await this.checkEvents()
    } finally {
      await sleep(this.options.pollInterval)
      this.pollEvents()
    }
  }

  /**
   * Checks for new events and triggers any listeners on those events.
   * Will only check for events that are currently being listened to.
   */
  private async checkEvents(): Promise<void> {
    const connected = await this.eth.connected()
    if (!connected) {
      return
    }

    // We only want to query final blocks, so we look a few blocks in the past.
    const block = await this.eth.getCurrentBlock()
    const lastFinalBlock = Math.max(-1, block - this.options.finalityDepth)

    // Check all subscribed events.
    await Promise.all(
      Object.values(this.subscriptions).map((subscription) =>
        this.checkEvent(subscription.filter, lastFinalBlock)
      )
    )
  }

  /**
   * Checks for new instances of an event.
   * @param filter Event filter to check.
   * @param lastFinalBlock Number of the latest block known to be final.
   */
  private async checkEvent(
    filter: EventFilter,
    lastFinalBlock: number
  ): Promise<void> {
    // Figure out the last block we've seen.
    const lastLoggedBlock = await this.db.getLastLoggedBlock(filter.hash)
    const firstUnsyncedBlock = lastLoggedBlock + 1

    // Don't do anything if we've already seen the latest final block.
    if (firstUnsyncedBlock > lastFinalBlock) {
      return
    }

    // Pull new events from the contract.
    const events = await this.eth.getEvents({
      ...filter.options,
      address: this.options.address,
      abi: this.options.abi,
      fromBlock: firstUnsyncedBlock,
      toBlock: lastFinalBlock,
    })

    // Filter out events that we've already seen.
    const unique = await this.getUniqueEvents(events)

    // Emit the events.
    await this.emitEvents(filter.hash, unique)

    // Update the last block that we've seen based on what we just queried.
    await this.db.setLastLoggedBlock(filter.hash, lastFinalBlock)
  }

  /**
   * Filters out any events we've already seen.
   * @param events A series of Ethereum events.
   * @returns any events we haven't seen already.
   */
  private async getUniqueEvents(events: EventLog[]): Promise<EventLog[]> {
    // Filter out duplicated events.
    events = events.filter((event, index, self) => {
      return (
        index ===
        self.findIndex((e) => {
          return e.hash === event.hash
        })
      )
    })

    // Filter out events we've already seen.
    const isUnique = await Promise.all(
      events.map(async (event) => {
        return !(await this.db.getEventSeen(event.hash))
      })
    )
    return events.filter((_, i) => isUnique[i])
  }

  /**
   * Emits events for a given event name.
   * @param filterHash Hash of the event filter to emit.
   * @param events Event objects for that event.
   */
  private async emitEvents(
    filterHash: string,
    events: EventLog[]
  ): Promise<void> {
    // Nothing to emit.
    if (events.length === 0) {
      return
    }

    // Mark these events as seen.
    for (const event of events) {
      await this.db.setEventSeen(event.hash)
    }

    // Alert any listeners.
    for (const listener of this.subscriptions[filterHash].listeners) {
      try {
        listener(events)
      } catch {}
    }
  }
}
