import BigNum = require('bn.js')
import * as assert from 'assert'
import { should } from '../../setup'

import { StateUpdate } from '../../../src/types'
import { DefaultBlockManager, ONE, stateUpdatesEqual } from '../../../src/app/'
import { TestUtils } from '../utils/test-utils'
import {
  BlockDB,
  BlockManager,
  CommitmentContract,
} from '../../../src/types/block-production'

/*******************
 * Mocks & Helpers *
 *******************/

class DummyCommitmentContract implements CommitmentContract {
  private throwOnSubmit: boolean = false

  public setThrowOnSubmit() {
    this.throwOnSubmit = true
  }

  public async submitBlock(root: Buffer): Promise<void> {
    if (this.throwOnSubmit) {
      this.throwOnSubmit = false
      throw Error('Simulating error submitting block')
    }
  }
}

class DummyBlockDB implements BlockDB {
  private nextBlockNumber: BigNum = ONE
  private pendingStateUpdates: StateUpdate[] = []
  private throwOnFinalize: boolean = false

  public async addPendingStateUpdate(stateUpdate: StateUpdate): Promise<void> {
    this.pendingStateUpdates.push(stateUpdate)
  }

  public setThrowOnFinalize(): void {
    this.throwOnFinalize = true
  }

  public async finalizeNextBlock(): Promise<void> {
    if (this.throwOnFinalize) {
      this.throwOnFinalize = false
      throw Error('Simulating error finalizing block')
    }
    this.pendingStateUpdates.length = 0
    this.nextBlockNumber = this.nextBlockNumber.add(ONE)
  }

  public async getMerkleRoot(blockNumber: BigNum): Promise<Buffer> {
    return Buffer.from('placeholder')
  }

  public async getNextBlockNumber(): Promise<BigNum> {
    return this.nextBlockNumber
  }

  public async getPendingStateUpdates(): Promise<StateUpdate[]> {
    return this.pendingStateUpdates
  }
}

const addStateUpdateToBlockManager = async (
  blockManager: BlockManager
): Promise<void> => {
  const stateUpdate: StateUpdate = TestUtils.generateNSequentialStateUpdates(
    1
  )[0]
  await blockManager.addPendingStateUpdate(stateUpdate)

  const returnedUpdates: StateUpdate[] = await blockManager.getPendingStateUpdates()
  assert(
    !!returnedUpdates && returnedUpdates.length === 1,
    `getPendingStateUpdates returned undefined or empty list when expecting a single StateUpdate. returned: ${JSON.stringify(
      returnedUpdates
    )}`
  )
}

/*********
 * TESTS *
 *********/

describe('DefaultBlockManager', () => {
  let blockManager: BlockManager
  let blockDB: DummyBlockDB
  let commitmentContract: DummyCommitmentContract

  beforeEach(async () => {
    blockDB = new DummyBlockDB()
    commitmentContract = new DummyCommitmentContract()
    blockManager = new DefaultBlockManager(blockDB, commitmentContract)
  })

  describe('getNextBlockNumber', () => {
    it('is in sync with BlockDB', async () => {
      const blockDBNextBlock: BigNum = await blockDB.getNextBlockNumber()
      const blockManagerNextBlock: BigNum = await blockManager.getNextBlockNumber()

      assert(
        blockManagerNextBlock.eq(blockDBNextBlock),
        'BlockDB and BlockManager are out of sync'
      )
    })
  })

  describe('addPendingStateUpdate / getPendingStateUpdates', () => {
    it('stores the pending StateUpdate(s)', async () => {
      const stateUpdates: StateUpdate[] = TestUtils.generateNSequentialStateUpdates(
        2
      )
      await blockManager.addPendingStateUpdate(stateUpdates[0])

      let returnedUpdates: StateUpdate[] = await blockManager.getPendingStateUpdates()

      assert(
        !!returnedUpdates && returnedUpdates.length === 1,
        `getPendingStateUpdates returned undefined or empty list when expecting a single StateUpdate. returned: ${JSON.stringify(
          returnedUpdates
        )}`
      )
      assert(
        stateUpdatesEqual(returnedUpdates[0], stateUpdates[0]),
        'Added StateUpdate is not the same as returned StateUpdate.'
      )

      await blockManager.addPendingStateUpdate(stateUpdates[1])
      returnedUpdates = await blockManager.getPendingStateUpdates()

      assert(
        !!returnedUpdates && returnedUpdates.length === 2,
        `getPendingStateUpdates returned undefined or empty list when expecting a 2 StateUpdates. returned: ${JSON.stringify(
          returnedUpdates
        )}`
      )
      assert(
        (stateUpdatesEqual(returnedUpdates[0], stateUpdates[0]) ||
          stateUpdatesEqual(returnedUpdates[1], stateUpdates[0])) &&
          (stateUpdatesEqual(returnedUpdates[0], stateUpdates[1]) ||
            stateUpdatesEqual(returnedUpdates[1], stateUpdates[1])),
        'Added StateUpdates are not the same as returned StateUpdates.'
      )
    })
  })

  describe('submitNextBlock', () => {
    it('increments next block and clears pending StateUpdates', async () => {
      const previousNextBlockNumber: BigNum = await blockManager.getNextBlockNumber()

      await addStateUpdateToBlockManager(blockManager)

      await blockManager.submitNextBlock()

      const updatedNextBlockNumber: BigNum = await blockManager.getNextBlockNumber()
      assert(
        updatedNextBlockNumber.eq(previousNextBlockNumber.add(ONE)),
        `Block Number after submitted block is not incremented. Got ${updatedNextBlockNumber.toString()}, expected ${previousNextBlockNumber
          .add(ONE)
          .toString()}`
      )

      const returnedUpdates = await blockManager.getPendingStateUpdates()
      assert(
        !!returnedUpdates && returnedUpdates.length === 0,
        `getPendingStateUpdates returned undefined or non-empty list when expecting an empty list. returned: ${JSON.stringify(
          returnedUpdates
        )}`
      )
    })

    it('does not submit block if there are no updates', async () => {
      const previousNextBlockNumber: BigNum = await blockManager.getNextBlockNumber()
      await blockManager.submitNextBlock()
      const updatedNextBlockNumber: BigNum = await blockManager.getNextBlockNumber()

      assert(
        previousNextBlockNumber.eq(updatedNextBlockNumber),
        'Block was incremented when submission should not have taken place.'
      )
    })

    it('throws if block submission fails', async () => {
      await addStateUpdateToBlockManager(blockManager)

      commitmentContract.setThrowOnSubmit()

      try {
        await blockManager.submitNextBlock()
        assert(false, 'This should have thrown.')
      } catch (e) {
        // This is success
      }
    })

    it('throws if block finalization fails', async () => {
      await addStateUpdateToBlockManager(blockManager)

      blockDB.setThrowOnFinalize()

      try {
        await blockManager.submitNextBlock()
        assert(false, 'This should have thrown.')
      } catch (e) {
        // This is success
      }
    })
  })
})
