import '../setup'
/* External Imports */
import {
  BigNumber,
  getLogger,
  IdentityVerifier,
  ONE,
  TestUtils,
  TWO,
} from '@pigi/core-utils'
import { DB, newInMemoryDB } from '@pigi/core-db/'
import {
  Address,
  RollupStateMachine,
  SignatureError,
  SignedTransaction,
  State,
  TransactionResult,
} from '@pigi/rollup-core'

/* Internal Imports */
import { RollupBlockBuilder } from '../../src/types'
import { Aggregator } from '../../src/app/aggregator'

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

const sender: Address = '423Ace7C343094Ed5EB34B0a1838c19adB2BAC92'
const recipient: Address = 'ba3739e8B603cFBCe513C9A4f8b6fFD44312d75E'
const signedTransaction: SignedTransaction = {
  signature: sender,
  transaction: {
    sender,
    recipient,
    tokenType: 0,
    amount: 10,
  },
}

const transactionResult: TransactionResult = {
  transactionNumber: ONE,
  signedTransaction,
  modifiedStorage: [
    {
      contractSlotIndex: 0,
      storageSlotIndex: 0,
      storage: 'First TX Storage 0',
    },
    {
      contractSlotIndex: 0,
      storageSlotIndex: 1,
      storage: 'First TX Storage 1',
    },
  ],
}

const transactionResultTwo: TransactionResult = {
  transactionNumber: TWO,
  signedTransaction,
  modifiedStorage: [
    {
      contractSlotIndex: 1,
      storageSlotIndex: 0,
      storage: 'SECOND TX Storage 0',
    },
    {
      contractSlotIndex: 1,
      storageSlotIndex: 1,
      storage: 'SECOND TX Storage 1',
    },
  ],
}

/*********
 * TESTS *
 *********/

describe('Aggregator', () => {
  let db: DB
  let stateMachine: DummyStateMachine
  let blockBuilder: DummyBlockBuilder
  let aggregator: Aggregator

  beforeEach(async () => {
    db = newInMemoryDB()
    stateMachine = new DummyStateMachine()
    blockBuilder = new DummyBlockBuilder()
  })

  describe('init', () => {
    it('should start fresh properly', async () => {
      aggregator = await Aggregator.create(
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
      await db.put(Aggregator.NEXT_TX_NUMBER_KEY, TWO.toBuffer())

      aggregator = await Aggregator.create(
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
      await db.put(Aggregator.NEXT_TX_NUMBER_KEY, TWO.toBuffer())
      stateMachine.setTransactionsSince([transactionResultTwo])

      aggregator = await Aggregator.create(
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
        transactionResultTwo,
        `BlockBuilder received incorrect TransactionResult!`
      )
    })
  })

  describe('handleTransaction', () => {
    beforeEach(async () => {
      aggregator = await Aggregator.create(
        db,
        stateMachine,
        blockBuilder,
        IdentityVerifier.instance()
      )
    })

    it('should fail for invalid signature', async () => {
      const signedTx: SignedTransaction = { ...signedTransaction }
      signedTx.signature = 'does not match'

      await TestUtils.assertThrowsAsync(
        async () => aggregator.handleTransaction(signedTx),
        SignatureError
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
      stateMachine.setTransactionResult(transactionResultTwo)
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
        transactionResultTwo,
        `BlockBuilder received incorrect TransactionResult!`
      )
    })
  })
})
