import '../setup'
/* External Imports */
import { getLogger, ONE, sleep, TWO } from '@pigi/core-utils'
import { DB, newInMemoryDB } from '@pigi/core-db/'
import { RollupBlock, TransactionResult } from '@pigi/rollup-core'

/* Internal Imports */
import { RollupBlockSubmitter } from '../../src/types'
import { DefaultRollupBlockBuilder } from '../../src/app'

const log = getLogger('block-builder', true)

/*********
 * MOCKS *
 *********/

class DummyBlockSubmitter implements RollupBlockSubmitter {
  public submittedBlocks: {} = {}

  public async submitBlock(rollupBlock: RollupBlock): Promise<void> {
    this.submittedBlocks[rollupBlock.blockNumber] = rollupBlock
  }
}

const signer: string = '0xabcd'
const transactionResult: TransactionResult = {
  transactionNumber: ONE,
  signedTransaction: {
    signature: signer,
    transaction: {
      sender: signer,
      body: {},
    },
  },
  modifiedStorage: [
    {
      contractSlotIndex: 0,
      storageSlotIndex: 0,
      storage: 'First TX Storage',
    },
  ],
}

const transactionResult2: TransactionResult = {
  transactionNumber: TWO,
  signedTransaction: {
    signature: signer,
    transaction: {
      sender: signer,
      body: {},
    },
  },
  modifiedStorage: [
    {
      contractSlotIndex: 1,
      storageSlotIndex: 1,
      storage: 'Second TX Storage',
    },
  ],
}

/*********
 * TESTS *
 *********/

describe('BlockBuilder', () => {
  let db: DB
  let blockSubmitter: DummyBlockSubmitter
  let blockBuilder: DefaultRollupBlockBuilder

  beforeEach(async () => {
    db = newInMemoryDB()
    blockSubmitter = new DummyBlockSubmitter()
  })

  describe('init', () => {
    it('should load state from DB', async () => {
      blockBuilder = await DefaultRollupBlockBuilder.create(
        db,
        blockSubmitter,
        2,
        30_000
      )

      await blockBuilder.addTransactionResult(transactionResult)
      Object.keys(blockSubmitter.submittedBlocks).length.should.equal(
        0,
        'Block should not have been submitted but was!'
      )

      blockBuilder = await DefaultRollupBlockBuilder.create(
        db,
        blockSubmitter,
        1,
        30_000
      )
      Object.keys(blockSubmitter.submittedBlocks).length.should.equal(
        1,
        'Block should have been submitted but was not!'
      )
    })
  })

  describe('addTransactionOutput', () => {
    beforeEach(async () => {
      blockBuilder = await DefaultRollupBlockBuilder.create(
        db,
        blockSubmitter,
        2,
        30_000
      )
    })

    it('should not submit block - # txs < submission threshold', async () => {
      await blockBuilder.addTransactionResult(transactionResult)
      Object.keys(blockSubmitter.submittedBlocks).length.should.equal(
        0,
        'Block should not have been submitted but was!'
      )
    })

    it('should submit block - # txs = submission threshold', async () => {
      await blockBuilder.addTransactionResult(transactionResult)
      await blockBuilder.addTransactionResult(transactionResult2)

      Object.keys(blockSubmitter.submittedBlocks).length.should.equal(
        1,
        'Block should have been submitted but was not!'
      )
    })

    it('should submit block - timeout passed', async () => {
      blockBuilder = await DefaultRollupBlockBuilder.create(
        db,
        blockSubmitter,
        2,
        500
      )

      await blockBuilder.addTransactionResult(transactionResult)
      Object.keys(blockSubmitter.submittedBlocks).length.should.equal(
        0,
        'Block should not have been submitted but was!'
      )

      await sleep(1_000)

      Object.keys(blockSubmitter.submittedBlocks).length.should.equal(
        1,
        'Block should have been submitted but was not!'
      )
    })
  })
})
