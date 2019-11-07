import { should } from '../../setup'

/* External Imports */
import { BigNumber, ONE } from '@pigi/core-utils'
import {
  KeyValueStore,
  BaseBucket,
  BaseDB,
  DEFAULT_PREFIX_LENGTH,
} from '@pigi/core-db'

import MemDown from 'memdown'
import * as assert from 'assert'

/* Internal Imports */
import { DefaultBlockDB } from '../../../src/app/block-production'
import { BlockDB } from '../../../src/types/block-production'
import { StateUpdate } from '../../../src/types'
import { TestUtils } from '../test-utils'
import { stateUpdatesEqual } from '../../../src/app/utils'

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
      const nextBlock: BigNumber = await blockDB.getNextBlockNumber()
      assert(
        nextBlock.eq(ONE),
        `Next Block Number did not default to 1. Expected ${ONE.toString()}, got ${nextBlock.toString()}`
      )
    })

    it('increases when finalized', async () => {
      await blockDB.finalizeNextBlock()
      const nextBlock: BigNumber = await blockDB.getNextBlockNumber()
      const expected: BigNumber = new BigNumber(2)
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
        `Added StateUpdate is not the same as returned StateUpdate. 
        Returned: ${JSON.stringify(returnedUpdates[0])},
        Added: ${JSON.stringify(stateUpdates[0])}`
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
      const previousNextBlockNumber: BigNumber = await blockDB.getNextBlockNumber()

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

      const updatedNextBlockNumber: BigNumber = await blockDB.getNextBlockNumber()
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
