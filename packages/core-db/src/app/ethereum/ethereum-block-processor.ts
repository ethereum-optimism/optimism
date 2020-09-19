/* External Imports */
import { getLogger, logError, sleep } from '@eth-optimism/core-utils'
import { Block, Provider, TransactionReceipt } from 'ethers/providers'

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
  private currentFinalizedBlockNumber: number

  private syncInProgress: boolean
  private syncCompleted: boolean

  constructor(
    private readonly db: DB,
    private readonly earliestBlock: number = 0,
    private readonly confirmsUntilFinal: number = 1
  ) {
    this.subscriptions = new Set<EthereumListener<Block>>()
    this.currentFinalizedBlockNumber = 0

    this.syncInProgress = false
    this.syncCompleted = false
    if (earliestBlock < 0) {
      throw Error('Earliest block must be >= 0')
    }
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
      try {
        const finalizedBlockNumber = this.getBlockFinalizedBy(blockNumber)

        if (finalizedBlockNumber < this.earliestBlock) {
          log.debug(
            `Received block [${blockNumber}] which finalizes a block ${finalizedBlockNumber}, before earliest block [${this.earliestBlock}]. Ignoring...`
          )
          return
        }

        log.debug(
          `Block [${blockNumber}] was mined! Finalizing block ${finalizedBlockNumber}`
        )

        await this.fetchAndDisseminateBlock(provider, finalizedBlockNumber)
        this.currentFinalizedBlockNumber = finalizedBlockNumber

        if (!syncPastBlocks || this.syncCompleted) {
          await this.storeLastProcessedBlockNumber(
            this.currentFinalizedBlockNumber
          )
        }
      } catch (e) {
        logError(
          log,
          `Error thrown processing block ${blockNumber}, finalizing block ${this.getBlockFinalizedBy(
            blockNumber
          )}. Exiting since throwing will not be caught.`,
          e
        )
        process.exit(1)
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
   * Fetches the Block, waits for finalization, and broadcasts the Block for the provided block number.
   *
   * @param provider The provider with the connection to the blockchain
   * @param blockNumber The block number
   */
  private async fetchAndDisseminateBlock(
    provider: Provider,
    blockNumber: number
  ): Promise<void> {
    log.debug(`Fetching block [${blockNumber}].`)
    const block: Block = await provider.getBlock(blockNumber, true)
    log.debug(`Received block: ${block.number}.`)

    this.subscriptions.forEach((h) => {
      try {
        // purposefully ignore promise
        h.handle(block)
      } catch (e) {
        // Cannot silently fail here because syncing will move on as if the block was processed
        logError(
          log,
          `Error in subscriber handling block number ${blockNumber}. Re-throwing because we cannot proceed skipping a block.`,
          e
        )
        throw e
      }
    })
  }

  /**
   * Fetches historical blocks.
   *
   * @param provider The provider with the connection to the blockchain.
   */
  private async syncBlocks(provider: Provider): Promise<void> {
    log.debug(`Syncing blocks.`)
    const lastSynced = await this.getLastSyncedBlockNumber()
    const syncStart = Math.max(lastSynced + 1, this.earliestBlock)

    log.debug(
      `Starting sync with block ${syncStart}. Last synced: ${lastSynced}, earliest block: ${this.earliestBlock}.`
    )

    const mostRecentFinalBlock = this.getBlockFinalizedBy(
      await this.getBlockNumber(provider)
    )

    if (mostRecentFinalBlock <= syncStart) {
      log.debug(`Up to date, not syncing.`)
      this.finishSync(mostRecentFinalBlock, mostRecentFinalBlock)
      return
    }

    for (let i = syncStart; i <= mostRecentFinalBlock; i++) {
      try {
        await this.fetchAndDisseminateBlock(provider, i)
      } catch (e) {
        logError(log, `Error fetching and disseminating block. Retrying...`, e)
        i--
        continue
      }
      await this.storeLastProcessedBlockNumber(i)
    }

    this.finishSync(syncStart, mostRecentFinalBlock)
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
    if (this.currentFinalizedBlockNumber === 0) {
      this.currentFinalizedBlockNumber = await provider.getBlockNumber()
    }

    log.debug(`Current block number: ${this.currentFinalizedBlockNumber}`)
    return this.currentFinalizedBlockNumber
  }

  /**
   * Gets the block number finalized by the block with the provided number.
   *
   * @param finalizingBlock The block number that finalizes the returned block number
   * @returns The block number finalized by the provided block number.
   */
  private getBlockFinalizedBy(finalizingBlock: number): number {
    if (this.confirmsUntilFinal <= 1) {
      return finalizingBlock
    }
    return finalizingBlock - (this.confirmsUntilFinal - 1)
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
