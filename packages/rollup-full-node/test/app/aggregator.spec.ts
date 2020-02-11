import '../setup'
/* External Imports */
import {
  add0x,
  BigNumber,
  getLogger,
  IdentityVerifier,
  keccak256,
  ONE,
  TWO,
  ZERO,
} from '@eth-optimism/core-utils'
import { DB, newInMemoryDB } from '@eth-optimism/core-db/'
import {
  abiEncodeTransaction,
  Address,
  RollupStateMachine,
  SignedTransaction,
  State,
  StorageSlot,
  StorageValue,
  TransactionResult,
} from '@eth-optimism/rollup-core'

/* Internal Imports */
import { RollupBlockBuilder } from '../../src/types'
import { DefaultAggregator } from '../../src/app/aggregator'

const log = getLogger('block-builder', true)

/*********
 * MOCKS *
 *********/

class DummyStateMachine implements RollupStateMachine {
  private mockedTransactionResult: TransactionResult
  private mockedTransactionsSince: TransactionResult[]

  public setTransactionResult(res: TransactionResult): void {
    this.mockedTransactionResult = res
  }

  public setTransactionsSince(txs: TransactionResult[]): void {
    this.mockedTransactionsSince = txs
  }

  public async applyTransaction(
    signedTx: SignedTransaction
  ): Promise<TransactionResult> {
    return this.mockedTransactionResult
  }

  public async getState(slotIndex: string): Promise<State> {
    return undefined
  }

  public async getTransactionResultsSince(
    transactionNumber: BigNumber
  ): Promise<TransactionResult[]> {
    return this.mockedTransactionsSince || []
  }
}

class DummyBlockBuilder implements RollupBlockBuilder {
  public addedTransactionResults: TransactionResult[] = []
  public async addTransactionResult(
    txResult: TransactionResult
  ): Promise<void> {
    this.addedTransactionResults.push(txResult)
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
const signedTransaction: SignedTransaction = {
  signature: ovmEntrypoint,
  transaction: {
    ovmEntrypoint,
    ovmCalldata,
  },
}

const transactionResult: TransactionResult = {
  transactionNumber: ONE,
  abiEncodedTransaction: abiEncodeTransaction(signedTransaction.transaction),
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
  abiEncodedTransaction: abiEncodeTransaction(signedTransaction.transaction),
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

describe('Aggregator', () => {
  let db: DB
  let stateMachine: DummyStateMachine
  let blockBuilder: DummyBlockBuilder
  let aggregator: DefaultAggregator

  beforeEach(async () => {
    db = newInMemoryDB()
    stateMachine = new DummyStateMachine()
    blockBuilder = new DummyBlockBuilder()
  })

  describe('init', () => {
    it('should start fresh properly', async () => {
      aggregator = await DefaultAggregator.create(
        db,
        stateMachine,
        blockBuilder,
        IdentityVerifier.instance()
      )

      blockBuilder.addedTransactionResults.length.should.equal(
        0,
        `BlockBuilder should not have received any TransactionResults!`
      )
    })

    it('should start properly when up-to-date', async () => {
      await db.put(DefaultAggregator.NEXT_TX_NUMBER_KEY, TWO.toBuffer())

      aggregator = await DefaultAggregator.create(
        db,
        stateMachine,
        blockBuilder,
        IdentityVerifier.instance()
      )

      blockBuilder.addedTransactionResults.length.should.equal(
        0,
        `BlockBuilder should not have received any TransactionResults!`
      )
    })

    it('should query missing transactions and send them to BlockBuilder', async () => {
      await db.put(DefaultAggregator.NEXT_TX_NUMBER_KEY, TWO.toBuffer())
      stateMachine.setTransactionsSince([transactionResult2])

      aggregator = await DefaultAggregator.create(
        db,
        stateMachine,
        blockBuilder,
        IdentityVerifier.instance()
      )

      blockBuilder.addedTransactionResults.length.should.equal(
        1,
        `BlockBuilder should have received TransactionResult, but didn't!`
      )
      blockBuilder.addedTransactionResults[0].should.eql(
        transactionResult2,
        `BlockBuilder received incorrect TransactionResult!`
      )
    })
  })

  describe('handleTransaction', () => {
    beforeEach(async () => {
      aggregator = await DefaultAggregator.create(
        db,
        stateMachine,
        blockBuilder,
        IdentityVerifier.instance()
      )
    })

    it('should execute transaction and send it to BlockBuilder', async () => {
      stateMachine.setTransactionResult(transactionResult)
      await aggregator.handleTransaction(signedTransaction)

      blockBuilder.addedTransactionResults.length.should.equal(
        1,
        `BlockBuilder should have received TransactionResult, but didn't!`
      )
      blockBuilder.addedTransactionResults[0].should.eql(
        transactionResult,
        `BlockBuilder received incorrect TransactionResult!`
      )
    })

    it('should enforce transaction order', async () => {
      stateMachine.setTransactionResult(transactionResult2)
      await aggregator.handleTransaction(signedTransaction)

      blockBuilder.addedTransactionResults.length.should.equal(
        0,
        `BlockBuilder should not have received TransactionResult, but did!`
      )

      stateMachine.setTransactionResult(transactionResult)
      await aggregator.handleTransaction(signedTransaction)

      blockBuilder.addedTransactionResults.length.should.equal(
        2,
        `BlockBuilder should have received TransactionResults, but didn't!`
      )
      blockBuilder.addedTransactionResults[0].should.eql(
        transactionResult,
        `BlockBuilder received incorrect TransactionResult!`
      )
      blockBuilder.addedTransactionResults[1].should.eql(
        transactionResult2,
        `BlockBuilder received incorrect TransactionResult!`
      )
    })
  })
})
