/* External Imports */
import { getLogger, logError, Md5Hash } from '@pigi/core-utils'
import { ethers, Contract } from 'ethers'
import { LogDescription } from 'ethers/utils'
import { Filter, Log, Provider } from 'ethers/providers'

/* Internal Imports */
import { EthereumEvent, EthereumListener } from '../../types/ethereum'
import { DB } from '../../types/db'

const log = getLogger('ethereum- event-processor')

interface SyncStatus {
  syncCompleted: boolean
  syncInProgress: boolean
}

/**
 * Ethereum EthereumEvent Processor
 * The single class to process and disseminate all Ethereum EthereumEvent subscriptions.
 */
export class EthereumEventProcessor {
  private readonly subscriptions: Map<
    string,
    Set<EthereumListener<EthereumEvent>>
  >
  private currentBlockNumber: number

  private syncStatuses: Map<string, SyncStatus>

  constructor(
    private readonly db: DB,
    private readonly earliestBlock: number = 0
  ) {
    this.subscriptions = new Map<string, Set<EthereumListener<EthereumEvent>>>()
    this.currentBlockNumber = 0

    this.syncStatuses = new Map<string, SyncStatus>()
  }

  /**
   * Subscribes to the event with the provided name for the  provided contract.
   * This will also fetch and send the provided event handler all historical events not in
   * the database unless syncPastEvents is set to false.
   *
   * @param contract The contract of the event
   * @param eventName The event name
   * @param handler The event handler subscribing
   * @param syncPastEvents Whether or not to fetch previous events
   */
  public async subscribe(
    contract: Contract,
    eventName: string,
    handler: EthereumListener<EthereumEvent>,
    syncPastEvents: boolean = true
  ): Promise<void> {
    const eventId: string = this.getEventID(contract.address, eventName)
    log.debug(`Received subscriber for event ${eventName}, ID: ${eventId}`)

    if (!this.subscriptions.has(eventId)) {
      this.subscriptions.set(
        eventId,
        new Set<EthereumListener<EthereumEvent>>([handler])
      )
    } else {
      this.subscriptions.get(eventId).add(handler)
      return
    }

    contract.on(contract.filters[eventName](), async (...data) => {
      log.debug(`Received live event: ${JSON.stringify(data)}`)
      const ethersEvent: ethers.Event = data[data.length - 1]
      const event: EthereumEvent = this.createEventFromEthersEvent(ethersEvent)
      await this.handleEvent(event)
      try {
        await this.db.put(
          Buffer.from(event.eventID),
          Buffer.from(ethersEvent.blockNumber.toString(10))
        )
      } catch (e) {
        logError(
          log,
          `Error storing most recent events block [${ethersEvent.blockNumber}]!`,
          e
        )
      }
    })

    if (syncPastEvents) {
      // If not in progress, create a status, mark it in progress
      if (!this.syncStatuses.has(eventId)) {
        this.syncStatuses.set(eventId, {
          syncInProgress: true,
          syncCompleted: false,
        })
        await this.syncPastEvents(contract, eventName, eventId)
        return
      }

      const syncStatus: SyncStatus = this.syncStatuses.get(eventId)
      // If completed, call callback
      if (syncStatus.syncCompleted) {
        await handler.onSyncCompleted(eventName)
      }
    }
  }

  /**
   * Fetches historical events for the provided contract with the provided event name.
   *
   * @param contract The contract for the events.
   * @param eventName The event name.
   * @param eventId The local event ID to identify the event in this class.
   */
  private async syncPastEvents(
    contract: Contract,
    eventName: string,
    eventId: string
  ): Promise<void> {
    log.debug(`Syncing events for event ${eventName}`)
    const blockNumber = await this.getBlockNumber(contract.provider)

    const lastSyncedBlockBuffer: Buffer = await this.db.get(
      Buffer.from(eventId)
    )
    const lastSyncedNumber: number = !!lastSyncedBlockBuffer
      ? parseInt(lastSyncedBlockBuffer.toString(), 10)
      : this.earliestBlock

    if (blockNumber === lastSyncedNumber) {
      log.debug(`Up to date, not syncing.`)
      this.finishSync(eventId, eventName, 0)
      return
    }

    const filter: Filter = contract.filters[eventName]()
    filter.fromBlock = lastSyncedNumber + 1
    filter.toBlock = 'latest'

    const logs: Log[] = await contract.provider.getLogs(filter)
    const events: EthereumEvent[] = logs.map((l) => {
      const logDesc: LogDescription = contract.interface.parseLog(l)
      return EthereumEventProcessor.createEventFromLogDesc(logDesc, eventId)
    })

    for (const event of events) {
      await this.handleEvent(event)
    }

    this.finishSync(eventId, eventName, events.length)
  }

  private finishSync(
    eventId: string,
    eventName: string,
    numEvents: number
  ): void {
    const status: SyncStatus = this.syncStatuses.get(eventId)
    status.syncCompleted = true
    status.syncInProgress = false

    log.debug(
      `Synced events for event ${eventName}, ${eventId}. Found ${numEvents} events`
    )

    for (const subscription of this.subscriptions.get(eventId)) {
      subscription.onSyncCompleted(eventId).catch((e) => {
        logError(log, 'Error calling EthereumEvent sync callback', e)
      })
    }
  }

  /**
   * Handles an event, whether live or historical, and passes it to all subscribers.
   *
   * @param event The event to disseminate.
   */
  private async handleEvent(event: EthereumEvent): Promise<void> {
    log.debug(`Handling event ${JSON.stringify(event)}`)
    const subscribers: Set<
      EthereumListener<EthereumEvent>
    > = this.subscriptions.get(event.eventID)

    subscribers.forEach((s) => {
      try {
        // purposefully ignore promise
        s.handle(event)
      } catch (e) {
        // should be logged in subscriber
      }
    })
  }

  /**
   * Fetches the current block number from the given provider.
   *
   * @param provider The provider connected to a node
   * @returns The current block number
   */
  private async getBlockNumber(provider: Provider): Promise<number> {
    if (this.currentBlockNumber === 0) {
      this.currentBlockNumber = await provider.getBlockNumber()
    }

    return this.currentBlockNumber
  }

  /**
   * Creates a local EthereumEvent from the provided Ethers LogDesc.
   *
   * @param logDesc The LogDesc in question
   * @param eventID The local event ID
   * @returns The local EthereumEvent
   */
  private static createEventFromLogDesc(
    logDesc: LogDescription,
    eventID: string
  ): EthereumEvent {
    const values = EthereumEventProcessor.getLogValues(logDesc.values)
    return {
      eventID,
      name: logDesc.name,
      signature: logDesc.signature,
      values,
    }
  }

  /**
   * Creates a local EthereumEvent from the provided Ethers event.
   *
   * @param event The event in question
   * @returns The local EthereumEvent
   */
  private createEventFromEthersEvent(event: ethers.Event): EthereumEvent {
    const values = EthereumEventProcessor.getLogValues(event.args)
    return {
      eventID: this.getEventID(event.address, event.event),
      name: event.event,
      signature: event.eventSignature,
      values,
    }
  }

  /**
   * Creates a JS object of key-value pairs for event fields and values.
   *
   * @param logArgs The args from the log event, including extra fields
   * @returns The values.
   */
  private static getLogValues(logArgs: {}): {} {
    const values = { ...logArgs }

    for (let i = 0; i < logArgs['length']; i++) {
      delete values[i.toString()]
    }
    delete values['length']

    return values
  }

  /**
   * Gets a unique ID for the event with the provided address and name.
   *
   * @param address The address of the event
   * @param eventName The name of the event
   * @returns The unique ID string.
   */
  private getEventID(address: string, eventName: string): string {
    return Md5Hash(`${address}${eventName}`)
  }
}
