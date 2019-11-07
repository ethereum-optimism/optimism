/* External Imports */
import { BigNumber } from '@pigi/core-utils'
import { Mutex } from 'async-mutex'

/* Internal Imports */
import {
  BlockDB,
  BlockManager,
  CommitmentContract,
} from '../../types/block-production'
import { StateUpdate } from '../../types'

/**
 * Simple BlockManager implementation.
 */
export class DefaultBlockManager implements BlockManager {
  private readonly blockSubmissionMutex: Mutex

  /**
   * Initializes the manager.
   * @param blockdb BlockDB instance to store/query data from.
   * @param commitmentContract Contract wrapper used to publish block roots.
   */
  constructor(
    private blockdb: BlockDB,
    private commitmentContract: CommitmentContract
  ) {
    this.blockSubmissionMutex = new Mutex()
  }

  /**
   * @returns the next plasma block number.
   */
  public async getNextBlockNumber(): Promise<BigNumber> {
    return this.blockdb.getNextBlockNumber()
  }

  /**
   * Adds a state update to the list of updates to be published in the next
   * plasma block.
   * @param stateUpdate State update to add to the next block.
   * @returns a promise that resolves once the update has been added.
   */
  public async addPendingStateUpdate(stateUpdate: StateUpdate): Promise<void> {
    await this.blockdb.addPendingStateUpdate(stateUpdate)
  }

  /**
   * @returns the state updates to be published in the next block.
   */
  public async getPendingStateUpdates(): Promise<StateUpdate[]> {
    return this.blockdb.getPendingStateUpdates()
  }

  /**
   * Finalizes the next block and submits the block root to Ethereum.
   * @returns a promise that resolves once the block has been published.
   */
  public async submitNextBlock(): Promise<void> {
    await this.blockSubmissionMutex.runExclusive(async () => {
      // Don't submit the block if there are no StateUpdates
      if ((await this.getPendingStateUpdates()).length === 0) {
        return
      }
      const blockNumber = await this.getNextBlockNumber()
      await this.blockdb.finalizeNextBlock()
      const root = await this.blockdb.getMerkleRoot(blockNumber)
      await this.commitmentContract.submitBlock(root)
    })
  }
}
