/* External Imports */
import { Mutex } from 'async-mutex'

import { BaseKey, BaseRangeBucket } from '../db'
import { BlockDB } from '../../types/block-production'
import { KeyValueStore, RangeStore } from '../../types/db'
import { StateUpdate } from '../../types/serialization'
import { BIG_ENDIAN, BigNumber, MAX_BIG_NUM, ONE, ZERO } from '../utils'
import { GenericMerkleIntervalTree } from './merkle-interval-tree'
import { deserializeStateUpdate, serializeStateUpdate } from '../serialization'

const KEYS = {
  NEXT_BLOCK: Buffer.from('nextblock'),
  BLOCK: new BaseKey('b', ['buffer']),
}

/**
 * Simple BlockDB implementation.
 */
export class DefaultBlockDB implements BlockDB {
  private readonly blockMutex: Mutex

  /**
   * Initializes the database wrapper.
   * @param vars the KeyValueStore to store variables in
   * @param blocks the KeyValueStore to store Blocks in
   */
  constructor(
    private readonly vars: KeyValueStore,
    private readonly blocks: KeyValueStore
  ) {
    this.blockMutex = new Mutex()
  }

  /**
   * @returns the next plasma block number.
   */
  public async getNextBlockNumber(): Promise<BigNumber> {
    // TODO: Cache this when it makes sense
    const buf = await this.vars.get(KEYS.NEXT_BLOCK)
    return !buf ? ONE : new BigNumber(buf, 'hex', BIG_ENDIAN)
  }

  /**
   * Adds a state update to the list of updates to be published in the next
   * plasma block.
   * @param stateUpdate State update to publish in the next block.
   * @returns a promise that resolves once the update has been added.
   */
  public async addPendingStateUpdate(stateUpdate: StateUpdate): Promise<void> {
    await this.blockMutex.runExclusive(async () => {
      const block = await this.getNextBlockStore()
      const start = stateUpdate.range.start
      const end = stateUpdate.range.end

      if (await block.hasDataInRange(start, end)) {
        throw new Error(
          'Block already contains a state update over that range.'
        )
      }

      const value = Buffer.from(serializeStateUpdate(stateUpdate))
      await block.put(start, end, value)
    })
  }

  /**
   * @returns the list of state updates waiting to be published in the next
   * plasma block.
   */
  public async getPendingStateUpdates(): Promise<StateUpdate[]> {
    const blockNumber = await this.getNextBlockNumber()
    return this.getStateUpdates(blockNumber)
  }

  /**
   * Computes the Merkle Interval Tree root of a given block.
   * @param blockNumber Block to compute a root for.
   * @returns the root of the block.
   */
  public async getMerkleRoot(blockNumber: BigNumber): Promise<Buffer> {
    const stateUpdates = await this.getStateUpdates(blockNumber)

    const leaves = stateUpdates.map((stateUpdate) => {
      // TODO: Actually encode this.
      const encodedStateUpdate = serializeStateUpdate(stateUpdate)
      return {
        start: stateUpdate.range.start,
        end: stateUpdate.range.end,
        data: encodedStateUpdate,
      }
    })
    const tree = new GenericMerkleIntervalTree(leaves)
    return tree.root().hash
  }

  /**
   * Finalizes the next plasma block so that it can be published.
   *
   * Note: The execution of this function is serialized internally,
   * but to be of use, the caller will most likely want to serialize
   * their calls to it as well.
   */
  public async finalizeNextBlock(): Promise<void> {
    await this.blockMutex.runExclusive(async () => {
      const prevBlockNumber: BigNumber = await this.getNextBlockNumber()
      const nextBlockNumber: Buffer = prevBlockNumber
        .add(ONE)
        .toBuffer(BIG_ENDIAN)

      await this.vars.put(KEYS.NEXT_BLOCK, nextBlockNumber)
    })
  }

  /**
   * Opens the RangeDB for a specific block.
   * @param blockNumber Block to open the RangeDB for.
   * @returns the RangeDB instance for the given block.
   */
  private async getBlockStore(blockNumber: BigNumber): Promise<RangeStore> {
    const key = KEYS.BLOCK.encode([blockNumber.toBuffer(BIG_ENDIAN)])
    const bucket = this.blocks.bucket(key)
    return new BaseRangeBucket(bucket.db, bucket.prefix)
  }

  /**
   * @returns the RangeDB instance for the next block to be published.
   *
   * IMPORTANT: This function itself is safe from concurrency issues, but
   * if the caller is modifying the returned RangeStore or needs to
   * guarantee the returned next RangeStore is not stale, both the call
   * to this function AND any subsequent reads / writes should be run with
   * the blockMutex lock held to guarantee the expected behavior.
   */
  private async getNextBlockStore(): Promise<RangeStore> {
    const blockNumber = await this.getNextBlockNumber()
    return this.getBlockStore(blockNumber)
  }

  /**
   * Queries all of the state updates within a given block.
   * @param blockNumber Block to query state updates for.
   * @returns the list of state updates for that block.
   */
  private async getStateUpdates(
    blockNumber: BigNumber
  ): Promise<StateUpdate[]> {
    const block = await this.getBlockStore(blockNumber)
    const values = await block.get(ZERO, MAX_BIG_NUM)
    return values.map((value) => {
      return deserializeStateUpdate(value.value.toString())
    })
  }
}
