/* External Import */
import { BigNumber, getLogger, hexBufToStr, logError } from '@pigi/core-utils'
import { DB, SparseMerkleTree, SparseMerkleTreeImpl } from '@pigi/core-db'
import {
  RollupBlock,
  TransactionResult,
  TransactionStorage,
} from '@pigi/rollup-core'

import AsyncLock from 'async-lock'

/* Internal Import */
import {
  RollupBlockBuilder,
  RollupBlockSubmitter,
  TreeUpdateError,
} from '../types'

// TODO: Actually ABI encode / decode and move into common serialization file if
//  this is the data type in use after merging with Karl's EVM stuff
const parseTransactionResultFromABI = (txResult: string): TransactionResult => {
  const json = JSON.parse(txResult)
  return {
    transactionNumber: new BigNumber(json['transactionNumber'], 'hex'),
    signedTransaction: json['signedTransaction'],
    modifiedStorage: json['modifiedStorage'],
  }
}
const abiEncodeTransactionResult = (txResult: TransactionResult): string => {
  return JSON.stringify({
    transactionNumber: txResult.transactionNumber.toString('hex'),
    signedTransaction: txResult.signedTransaction,
    modifiedStorage: txResult.modifiedStorage,
  })
}

const log = getLogger('rollup-block-builder')

interface PendingBlock {
  blockNumber: number
  transactionResults: TransactionResult[]
}

/**
 *  Default Block Builder implementation. We don't expect others.
 */
export class DefaultRollupBlockBuilder implements RollupBlockBuilder {
  public static readonly LAST_SUBMISSION_DATE_KEY: Buffer = Buffer.from(
    'last_submission'
  )
  public static readonly PENDING_BLOCK_KEY: Buffer = Buffer.from(
    'pending_block_number'
  )
  public static readonly TRANSACTION_COUNT_KEY: Buffer = Buffer.from('tx_count')
  public static readonly TREE_ROOT_KEY: Buffer = Buffer.from('tree_root')

  private static readonly LOCK_KEY: string = 'lock'

  private readonly lock: AsyncLock

  private tree: SparseMerkleTree
  private subtrees: SparseMerkleTree[]
  private lastBlockSubmission: Date
  private pendingBlock: PendingBlock

  public static async create(
    db: DB,
    rollupBlockSubmitter: RollupBlockSubmitter,
    maxTransactionsPerBlock: number = 100,
    maxDelayBetweenBlocksMillis: number = 30_000
  ): Promise<DefaultRollupBlockBuilder> {
    const blockBuilder: DefaultRollupBlockBuilder = new DefaultRollupBlockBuilder(
      db,
      rollupBlockSubmitter,
      maxTransactionsPerBlock,
      maxDelayBetweenBlocksMillis
    )

    await blockBuilder.init()

    return blockBuilder
  }

  constructor(
    private readonly db: DB,
    private readonly rollupBlockSubmitter: RollupBlockSubmitter,
    private readonly maxTransactionsPerBlock: number,
    private readonly maxDelayBetweenBlocksMillis: number
  ) {
    this.pendingBlock = {
      blockNumber: 0,
      transactionResults: [],
    }
    this.lock = new AsyncLock()
  }

  private async init(): Promise<void> {
    try {
      const [
        lastSubmissionDateBuffer,
        txCountBuffer,
        pendingBlockBuffer,
        treeRoot,
      ]: Buffer[] = await Promise.all([
        this.db.get(DefaultRollupBlockBuilder.LAST_SUBMISSION_DATE_KEY),
        this.db.get(DefaultRollupBlockBuilder.TRANSACTION_COUNT_KEY),
        this.db.get(DefaultRollupBlockBuilder.PENDING_BLOCK_KEY),
        this.db.get(DefaultRollupBlockBuilder.TREE_ROOT_KEY),
      ])

      if (!txCountBuffer) {
        log.info(
          `Init returning -- no stored last transition. This is a fresh start.`
        )
        this.lastBlockSubmission = new Date()
        this.setBlockSubmissionTimeout()
        this.tree = await SparseMerkleTreeImpl.create(this.db, undefined, 16)
        this.subtrees = []
        return
      }

      // TODO: Create int [de]serialization util function(s) so there's no way to mess up radix
      this.lastBlockSubmission = !!lastSubmissionDateBuffer
        ? new Date(parseInt(lastSubmissionDateBuffer.toString(), 10))
        : new Date()

      const transactionCount: number = txCountBuffer
        ? parseInt(txCountBuffer.toString(), 10)
        : 0

      const pendingBlock: number = pendingBlockBuffer
        ? parseInt(pendingBlockBuffer.toString(), 10)
        : 1

      this.tree = await SparseMerkleTreeImpl.create(this.db, treeRoot, 16)
      this.subtrees = []

      const promises: Array<Promise<Buffer>> = []
      for (let i = 1; i <= transactionCount; i++) {
        promises.push(
          this.db.get(DefaultRollupBlockBuilder.getTransactionKey(i))
        )
      }

      const transactionBuffers: Buffer[] = await Promise.all(promises)
      const transactionResults: TransactionResult[] = transactionBuffers.map(
        // x => parseTransactionResultFromABI(hexBufToStr(x))
        (x) => parseTransactionResultFromABI(x.toString())
      )
      this.pendingBlock = {
        blockNumber: pendingBlock,
        transactionResults,
      }

      log.info(
        `Initialized aggregator with pending block: ${JSON.stringify(
          this.pendingBlock
        )} and tree root: ${
          // TODO: THIS
          (await this.tree.getRootHash()).toString('hex')
        }`
      )

      return this.submitBlock()
    } catch (e) {
      logError(log, 'Error initializing aggregator', e)
      throw e
    }
  }

  // Note: Calls to this should be serialized, as it is not safe for multiple async calls at once.
  public async addTransactionResult(
    transactionResult: TransactionResult
  ): Promise<void> {
    // TODO: Protect against duplicates across blocks
    if (
      this.pendingBlock.transactionResults.length > 0 &&
      this.pendingBlock.transactionResults[
        this.pendingBlock.transactionResults.length - 1
      ].transactionNumber.eq(transactionResult.transactionNumber)
    ) {
      log.warn(`Ignoring duplicate TransactionResult. Received [${JSON.stringify(
        transactionResult
      )}], 
        but last transaction is 
        [${
          this.pendingBlock.transactionResults[
            this.pendingBlock.transactionResults.length - 1
          ]
        }].`)
      return
    }

    this.pendingBlock.transactionResults.push(transactionResult)

    log.debug(
      `Received TransactionResult [${JSON.stringify(
        transactionResult
      )}]. Pending block [${this.pendingBlock.blockNumber}] size: [${
        this.pendingBlock.transactionResults.length
      }]`
    )

    await this.db.put(
      DefaultRollupBlockBuilder.getTransactionKey(
        this.pendingBlock.transactionResults.length
      ),
      Buffer.from(abiEncodeTransactionResult(transactionResult))
    )

    await this.db.put(
      DefaultRollupBlockBuilder.TRANSACTION_COUNT_KEY,
      Buffer.from(this.pendingBlock.transactionResults.length.toString(10))
    )

    if (
      this.pendingBlock.transactionResults.length >=
      this.maxTransactionsPerBlock
    ) {
      log.debug(`Submitting block [${this.pendingBlock.blockNumber}]`)
      return this.submitBlock()
    } else {
      log.debug(
        `Not submitting partial block. Pending block [${this.pendingBlock.blockNumber}] is at ${this.pendingBlock.transactionResults.length}/${this.maxTransactionsPerBlock} of its capacity.`
      )
    }
  }

  /**
   * Submits a block to the main chain through the BlockSubmitter, creating a new
   * pending block for future transactions.
   */
  private async submitBlock(): Promise<void> {
    log.debug(
      `Waiting to acquire lock to submit block [${this.pendingBlock.blockNumber}]`
    )
    return this.lock.acquire(DefaultRollupBlockBuilder.LOCK_KEY, async () => {
      log.debug(
        `Lock acquired to submit block [${this.pendingBlock.blockNumber}]`
      )
      if (
        this.pendingBlock.transactionResults.length <
        this.maxTransactionsPerBlock
      ) {
        const millisSinceLastSubmission: number =
          new Date().getTime() - this.lastBlockSubmission.getTime()
        if (millisSinceLastSubmission < this.maxDelayBetweenBlocksMillis) {
          log.debug(
            `Not submitting block: Block tx count [${
              Object.keys(this.pendingBlock.transactionResults).length
            }] less than max per block [${
              this.maxTransactionsPerBlock
            }], and max time between blocks has not elapsed.`
          )
          this.setBlockSubmissionTimeout(
            this.maxDelayBetweenBlocksMillis - millisSinceLastSubmission
          )
          return
        } else if (this.pendingBlock.transactionResults.length === 0) {
          log.info(`Not submitting block: Block is empty.`)
          this.setBlockSubmissionTimeout(this.maxDelayBetweenBlocksMillis)
          return
        }
      }

      log.info(`Building block # [${this.pendingBlock.blockNumber}]`)
      const toSubmit: RollupBlock = await this.buildBlock()
      log.info(`Block built. Submitting block # [${toSubmit.blockNumber}]`)

      await this.rollupBlockSubmitter.submitBlock(toSubmit)
      log.info(`Block # [${toSubmit.blockNumber}] submitted.`)

      await this.db.put(
        DefaultRollupBlockBuilder.TRANSACTION_COUNT_KEY,
        Buffer.from('0')
      )
      await this.db.put(
        DefaultRollupBlockBuilder.PENDING_BLOCK_KEY,
        Buffer.from(this.pendingBlock.blockNumber.toString(10))
      )

      this.lastBlockSubmission = new Date()

      this.setBlockSubmissionTimeout()
    })
  }

  /**
   * Builds the PendingBlock into a RollupBlock that can be submitted.
   * Note: This function creates a new Pending Block!
   *
   * @returns RollupBlock
   */
  private async buildBlock(): Promise<RollupBlock> {
    log.debug(
      `Building Block to submit. Block # [${this.pendingBlock.blockNumber}]`
    )
    // Let next block get appended to while we're building this block.
    const block: PendingBlock = this.pendingBlock
    // TODO: due to asynchrony, the block to build might be too big. Move txs into new block here if necessary.
    //  See: https://github.com/plasma-group/optimistic-rollup/issues/39
    this.pendingBlock = {
      blockNumber: block.blockNumber + 1,
      transactionResults: [],
    }

    // Build Contract Slot ID => Updated Storage Slot IDs map
    const modifiedStorageMap: Map<string, TransactionStorage> = new Map<
      string,
      TransactionStorage
    >()
    for (const res of block.transactionResults) {
      for (const modifiedStorage of res.modifiedStorage) {
        modifiedStorageMap.set(
          `${modifiedStorage.contractSlotIndex}_${modifiedStorage.storageSlotIndex}`,
          modifiedStorage
        )
      }
    }

    // Update all contract storage slots
    const modifiedContractSlotIndexes: Set<number> = new Set()
    const storagePromises: Array<Promise<void>> = []
    for (const modifiedStorage of modifiedStorageMap.values()) {
      storagePromises.push(this.updateStorageSlot(modifiedStorage))
      modifiedContractSlotIndexes.add(modifiedStorage.contractSlotIndex)
    }

    log.debug(
      `Awaiting updateStorageSlot promises. Count: [${storagePromises.length}]`
    )
    // TODO: Figure out how we recover from this when it fails. A new block _may_ already be being built.
    await Promise.all(storagePromises)
    log.debug(`updateStorageSlot promises completed`)

    // Update the base contract tree with the roots of all subtrees
    const blockPromises: Array<Promise<void>> = []
    for (const idx of modifiedContractSlotIndexes.keys()) {
      blockPromises.push(this.updateContractSlot(idx))
    }

    log.debug(
      `Awaiting updateContractSlot promises. Count: [${blockPromises.length}]`
    )
    // TODO: Figure out how we recover from this when it fails. A new block _may_ already be being built.
    await Promise.all(blockPromises)
    log.debug(`updateContractSlot completed`)

    const stateRoot: Buffer = await this.tree.getRootHash()

    return {
      blockNumber: block.blockNumber,
      stateRoot: stateRoot.toString('hex'),
      signedTransactions: block.transactionResults.map(
        (x) => x.signedTransaction
      ),
    }
  }

  /**
   * Updates the tree with the storage value of the provided TransactionStorage object.
   *
   * @param transactionStorage The TransactionStorage object.
   */
  private async updateStorageSlot(
    transactionStorage: TransactionStorage
  ): Promise<void> {
    log.debug(
      `Updating contract storage [${transactionStorage.contractSlotIndex}, ${transactionStorage.storageSlotIndex}] to [${transactionStorage.storage}].`
    )
    const contractSlotBN: BigNumber = new BigNumber(
      transactionStorage.contractSlotIndex,
      10
    )

    const subtreeRoot: Buffer = await this.tree.getLeaf(contractSlotBN)
    if (!subtreeRoot) {
      log.debug(
        `Creating contract slot index [${transactionStorage.contractSlotIndex}].`
      )
      this.subtrees[
        transactionStorage.contractSlotIndex
      ] = await SparseMerkleTreeImpl.create(this.db, undefined, 32)
    } else if (!this.subtrees[transactionStorage.contractSlotIndex]) {
      log.info(
        `Subtree at index [${
          transactionStorage.contractSlotIndex
        }] exists with root [${subtreeRoot.toString(
          'hex'
        )}] but is not in the subtree array. Creating it.`
      )
      this.subtrees[
        transactionStorage.contractSlotIndex
      ] = await SparseMerkleTreeImpl.create(this.db, subtreeRoot, 32)
    }

    const storageSlotBN: BigNumber = new BigNumber(
      transactionStorage.storageSlotIndex,
      10
    )

    let updated: boolean
    try {
      updated = await this.subtrees[
        transactionStorage.contractSlotIndex
      ].update(storageSlotBN, Buffer.from(transactionStorage.storage))
    } catch (e) {
      logError(
        log,
        `Error updating contract storage [${transactionStorage.contractSlotIndex}, ${transactionStorage.storageSlotIndex}] to [${transactionStorage.storage}].`,
        e
      )
      throw e
    }

    if (!updated) {
      const msg: string = `Error updating contract storage [${transactionStorage.contractSlotIndex}, ${transactionStorage.storageSlotIndex}] to [${transactionStorage.storage}].`
      log.error(msg)
      throw new TreeUpdateError(msg)
    }
    log.debug(
      `Updated contract storage [${transactionStorage.contractSlotIndex}, ${transactionStorage.storageSlotIndex}] to [${transactionStorage.storage}].`
    )
  }

  /**
   * Updates the provided contract slot index from the state root of the associated subtree.
   *
   * @param contractSlotIndex
   */
  private async updateContractSlot(contractSlotIndex: number): Promise<void> {
    log.debug(
      `Updating contract slot index [${contractSlotIndex}] with subtree hash.`
    )

    const subtreeHash: Buffer = await this.subtrees[
      contractSlotIndex
    ].getRootHash()
    const slotBN: BigNumber = new BigNumber(contractSlotIndex, 10)

    const updated: boolean = await this.tree.update(slotBN, subtreeHash)

    if (!updated) {
      const msg: string = `Error updating contract slot index [${contractSlotIndex}] with new tree root`
      log.error(msg)
      throw new TreeUpdateError(msg)
    }
    log.debug(
      `Updated Contract Slot Index [${contractSlotIndex}] to [${subtreeHash.toString(
        'hex'
      )}].`
    )
  }

  /**
   * Sets the timeout for submitting a block if the max delay between blocks passes.
   *
   * @param timeoutMillis The number of millis until the timeout should fire.
   */
  private setBlockSubmissionTimeout(timeoutMillis?: number): void {
    setTimeout(async () => {
      await this.submitBlock()
    }, timeoutMillis || this.maxDelayBetweenBlocksMillis)
  }

  /**
   * Gets the transaction key associated with the provided transaction number in the DB.
   *
   * @param txNumber The number of the transaction within the pending block.
   * @returns the key that can be used to save/fetch the transaction.
   */
  private static getTransactionKey(txNumber: number): Buffer {
    return Buffer.from(`tx${txNumber.toString(10)}`)
  }
}
