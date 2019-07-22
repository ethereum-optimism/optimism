import BigNum = require('bn.js')
import MemDown from 'memdown'
import * as assert from 'assert'

import { should } from '../../setup'
import { KeyValueStore } from '../../../src/types/db'
import { BaseBucket, BaseDB, DEFAULT_PREFIX_LENGTH } from '../../../src/app/db'
import { BlockDB } from '../../../src/types/block-production'
import { DefaultBlockDB } from '../../../src/app/block-production'
import { ONE, stateUpdatesEqual } from '../../../src/app/utils'
import { StateUpdate } from '../../../src/types/serialization'
import { TestUtils } from '../utils/test-utils'

/*******************
 * Mocks & Helpers *
 *******************/

/*********
 * TESTS *
 *********/

describe('DefaultBlockDB', () => {
  let varStore: KeyValueStore
  let blockStore: KeyValueStore
  let blockDB: BlockDB

  beforeEach(async () => {
    varStore = new BaseBucket(
      new BaseDB(new MemDown('') as any, DEFAULT_PREFIX_LENGTH * 2),
      Buffer.from('v')
    )
    blockStore = new BaseBucket(
      new BaseDB(new MemDown('') as any, DEFAULT_PREFIX_LENGTH * 2),
      Buffer.from('b')
    )
    blockDB = new DefaultBlockDB(varStore, blockStore)
  })

  describe('getNextBlockNumber', () => {
    it('returns 1 by default', async () => {
      const nextBlock: BigNum = await blockDB.getNextBlockNumber()
      assert(
        nextBlock.eq(ONE),
        `Next Block Number did not default to 1. Expected ${ONE.toString()}, got ${nextBlock.toString()}`
      )
    })

    it('increases when finalized', async () => {
      await blockDB.finalizeNextBlock()
      const nextBlock: BigNum = await blockDB.getNextBlockNumber()
      const expected: BigNum = new BigNum(2)
      assert(
        nextBlock.eq(expected),
        `Next Block didn't increase after block finalization. Expected ${expected.toString()}, got ${nextBlock.toString()}`
      )
    })
  })

  describe('addPendingStateUpdate / getPendingStateUpdates', () => {
    it('stores the pending StateUpdate(s)', async () => {
      const stateUpdates: StateUpdate[] = TestUtils.generateNSequentialStateUpdates(
        2
      )
      await blockDB.addPendingStateUpdate(stateUpdates[0])

      let returnedUpdates: StateUpdate[] = await blockDB.getPendingStateUpdates()

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

      await blockDB.addPendingStateUpdate(stateUpdates[1])
      returnedUpdates = await blockDB.getPendingStateUpdates()

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

  describe('finalizeNextBlock', () => {
    it('increments next block and clears pending StateUpdates', async () => {
      const previousNextBlockNumber: BigNum = await blockDB.getNextBlockNumber()

      const stateUpdate: StateUpdate = TestUtils.generateNSequentialStateUpdates(
        1
      )[0]
      await blockDB.addPendingStateUpdate(stateUpdate)

      let returnedUpdates: StateUpdate[] = await blockDB.getPendingStateUpdates()
      assert(
        !!returnedUpdates && returnedUpdates.length === 1,
        `getPendingStateUpdates returned undefined or empty list when expecting a single StateUpdate. returned: ${JSON.stringify(
          returnedUpdates
        )}`
      )

      await blockDB.finalizeNextBlock()

      const updatedNextBlockNumber: BigNum = await blockDB.getNextBlockNumber()
      assert(
        updatedNextBlockNumber.eq(previousNextBlockNumber.add(ONE)),
        `Block Number after submitted block is not incremented. Got ${updatedNextBlockNumber.toString()}, expected ${previousNextBlockNumber
          .add(ONE)
          .toString()}`
      )

      returnedUpdates = await blockDB.getPendingStateUpdates()
      assert(
        !!returnedUpdates && returnedUpdates.length === 0,
        `getPendingStateUpdates returned undefined or non-empty list when expecting an empty list. returned: ${JSON.stringify(
          returnedUpdates
        )}`
      )
    })
  })
})
