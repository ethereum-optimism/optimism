import '../setup'
/* External Imports */
import {
  add0x,
  getLogger,
  keccak256,
  ONE,
  sleep,
  TWO,
  ZERO,
} from '@eth-optimism/core-utils'
import { DB, newInMemoryDB } from '@eth-optimism/core-db/'
import {
  abiEncodeTransaction,
  Address,
  RollupBlock,
  StorageSlot,
  StorageValue,
  Transaction,
  TransactionResult,
} from '@eth-optimism/rollup-core'

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

const ovmEntrypoint: Address = '0x423Ace7C343094Ed5EB34B0a1838c19adB2BAC92'
const ovmCalldata: Address = '0xba3739e8B603cFBCe513C9A4f8b6fFD44312d75E'

const contractAddress: Address = '0xC111937D5f4cF3a9096f38384E5Bd6DCbda1Af71'
const contractAddress2: Address = '0x01F33feD7D584f4bd938B4f7585723Ce00D77fa6'
const storageSlot: StorageSlot = add0x(
  keccak256(Buffer.from('Storage slot').toString('hex'))
)
const storageValue: StorageValue = add0x(
  keccak256(Buffer.from('Storage value').toString('hex'))
)
const storageValue2: StorageValue = add0x(
  keccak256(Buffer.from('Storage value 2').toString('hex'))
)

const transaction: Transaction = {
  ovmEntrypoint,
  ovmCalldata,
}

const transactionResult: TransactionResult = {
  transactionNumber: ONE,
  abiEncodedTransaction: abiEncodeTransaction(transaction),
  updatedStorage: [
    {
      contractAddress,
      storageSlot,
      storageValue,
    },
    {
      contractAddress,
      storageSlot,
      storageValue: storageValue2,
    },
  ],
  updatedContracts: [
    {
      contractAddress,
      contractNonce: TWO,
    },
  ],
  transactionReceipt: undefined,
}

const transactionResult2: TransactionResult = {
  transactionNumber: TWO,
  abiEncodedTransaction: abiEncodeTransaction(transaction),
  updatedStorage: [
    {
      contractAddress: contractAddress2,
      storageSlot,
      storageValue,
    },
    {
      contractAddress: contractAddress2,
      storageSlot,
      storageValue: storageValue2,
    },
  ],
  updatedContracts: [
    {
      contractAddress: contractAddress2,
      contractNonce: TWO,
    },
  ],
  transactionReceipt: undefined,
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
