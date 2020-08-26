/* External Imports */
import { getLogger, logError } from '@eth-optimism/core-utils'
import { Block, Provider } from 'ethers/providers'

/* Internal Imports */
import { EthereumListener } from '../../types/ethereum'
import { DB } from '../../types/db'
import { Transaction } from 'ethers/utils'

const log = getLogger('ethereum-block-processor')
const blockKey: Buffer = Buffer.from('latestBlock')

/**
 * Ethereum Block Processor
 * Single place through which all block subscriptions are handled.
 */
export class EthereumBlockProcessor {
  private readonly subscriptions: Set<EthereumListener<Block>>
  private currentBlockNumber: number

  private syncInProgress: boolean
  private syncCompleted: boolean

  constructor(
    private readonly db: DB,
    private readonly earliestBlock: number = 0,
    private readonly confirmsUntilFinal: number = 1
  ) {
    this.subscriptions = new Set<EthereumListener<Block>>()
    this.currentBlockNumber = 0

    this.syncInProgress = false
    this.syncCompleted = false
  }

  /**
   * Subscribes to new blocks.
   * This will also fetch and send the provided event handler all historical blocks not in
   * the database unless syncPastBlocks is set to false.
   *
   * @param provider The provider with the connection to the blockchain
   * @param handler The event handler subscribing
   * @param syncPastBlocks Whether or not to fetch previous events
   */
  public async subscribe(
    provider: Provider,
    handler: EthereumListener<Block>,
    syncPastBlocks: boolean = true
  ): Promise<void> {
    this.subscriptions.add(handler)

    provider.on('block', async (blockNumber) => {
      if (blockNumber < this.earliestBlock) {
        log.debug(
          `Received block [${blockNumber}] which is before earliest block [${this.earliestBlock}]. Ignoring...`
        )
        return
      }

      log.debug(`Block [${blockNumber}] was mined!`)

      await this.fetchAndDisseminateBlock(provider, blockNumber)
      this.currentBlockNumber = blockNumber

      if (!syncPastBlocks || this.syncCompleted) {
        await this.storeLastProcessedBlockNumber(this.currentBlockNumber)
      }
    })

    if (syncPastBlocks) {
      if (this.syncCompleted) {
        await handler.onSyncCompleted()
        return
      }

      if (!this.syncInProgress) {
        this.syncInProgress = true
        await this.syncBlocks(provider)
      }
    }
  }

  /**
   * Fetches and broadcasts the Block for the provided block number.
   *
   * @param provider The provider with the connection to the blockchain
   * @param blockNumber The block number
   */
  private async fetchAndDisseminateBlock(
    provider: Provider,
    blockNumber: number
  ): Promise<void> {
    log.debug(`Fetching block [${blockNumber}].`)
    let block: Block = await provider.getBlock(blockNumber, true)
    log.debug(`Received block: ${block.number}.`)

    if (
      this.confirmsUntilFinal > 1 &&
      !!block.transactions &&
      !!block.transactions.length
    ) {
      log.debug(
        `Waiting for ${this.confirmsUntilFinal} confirms before disseminating block ${blockNumber}`
      )
      // TODO: What happens on re-org? I think we're stuck waiting on this confirmation that will never come forever.
      try {
        await provider.waitForTransaction(
          (block.transactions[0] as any).hash,
          this.confirmsUntilFinal
        )
      } catch (e) {
        logError(
          log,
          `Error waiting for ${this.confirmsUntilFinal} confirms on block ${blockNumber}`,
          e
        )
        // TODO: If this is not a re-org, this may require a restart or some other action.
        //  Monitor what actually happens and re-throw here if necessary.
        return
      }

      log.debug(
        `Received ${this.confirmsUntilFinal} confirms for block ${blockNumber}. Refetching block`
      )

      block = await provider.getBlock(blockNumber, true)
    }

    this.subscriptions.forEach((h) => {
      try {
        // purposefully ignore promise
        h.handle(block)
      } catch (e) {
        // should be logged in handler
      }
    })
  }

  /**
   * Fetches historical blocks.
   *
   * @param provider The provider with the connection to the blockchain.
   */
  private async syncBlocks(provider: Provider): Promise<void> {
    log.debug(`Syncing blocks`)
    const syncStart = Math.max(
      (await this.getLastSyncedBlockNumber()) + 1,
      this.earliestBlock
    )
    const blockNumber = await this.getBlockNumber(provider)

    if (blockNumber === syncStart) {
      log.debug(`Up to date, not syncing.`)
      this.finishSync(blockNumber, blockNumber)
      return
    }

    for (let i = syncStart; i <= blockNumber; i++) {
      await this.fetchAndDisseminateBlock(provider, i)
      await this.storeLastProcessedBlockNumber(i)
    }

    this.finishSync(syncStart, blockNumber)
  }

  private finishSync(syncStart: number, currentBlock: number): void {
    this.syncCompleted = true
    this.syncInProgress = false

    if (syncStart !== currentBlock) {
      log.debug(`Synced from block [${syncStart}] to [${currentBlock}]!`)
    } else {
      log.debug(
        `No sync necessary. Last processed and current block are the same block number: ${currentBlock}`
      )
    }

    for (const callback of this.subscriptions) {
      callback.onSyncCompleted().catch((e) => {
        logError(log, 'Error calling Block sync callback', e)
      })
    }
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

    log.debug(`Current block number: ${this.currentBlockNumber}`)
    return this.currentBlockNumber
  }

  /**
   * Gets the last synced block number stored in the DB or earliest block -1 if there is not one.
   *
   * @returns The last synced block number.
   */
  private async getLastSyncedBlockNumber(): Promise<number> {
    const lastSyncedBlockBuffer: Buffer = await this.db.get(blockKey)
    return !!lastSyncedBlockBuffer
      ? parseInt(lastSyncedBlockBuffer.toString(), 10)
      : this.earliestBlock - 1
  }

  /**
   * Stores the provided block number as the last processed block number
   *
   * @param blockNumber The block number to store.
   */
  private async storeLastProcessedBlockNumber(
    blockNumber: number
  ): Promise<void> {
    try {
      await this.db.put(blockKey, Buffer.from(blockNumber.toString()))
    } catch (e) {
      logError(
        log,
        `Error storing most recent block received [${blockNumber}]!`,
        e
      )
    }
  }
}
