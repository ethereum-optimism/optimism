import '../setup'
import MemDown from 'memdown'

/* External Imports */
import {
  BaseDB,
  DB,
  DefaultSignatureVerifier,
  serializeObject,
  SimpleClient,
} from '@pigi/core'

/* Internal Imports */
import { ethers } from 'ethers'
import { MockAggregator } from '../../src/mock'
import { AGGREGATOR_MNEMONIC, getGenesisState } from '../helpers'
import {
  UNI_TOKEN_TYPE,
  DefaultRollupStateMachine,
  FaucetRequest,
  SignedTransaction,
  SignedTransactionReceipt,
  AGGREGATOR_ADDRESS,
} from '../../src'
import { RollupStateMachine } from '../../src/types'

/*********
 * TESTS *
 *********/

describe('MockAggregator', () => {
  let client
  let aggregator: MockAggregator
  let stateDB: DB
  let blockDB: DB

  let aliceWallet: ethers.Wallet

  beforeEach(async () => {
    aliceWallet = ethers.Wallet.createRandom()
    stateDB = new BaseDB(new MemDown('state') as any)
    blockDB = new BaseDB(new MemDown('block') as any, 256)

    const rollupStateMachine: RollupStateMachine = await DefaultRollupStateMachine.create(
      getGenesisState(aliceWallet.address),
      stateDB
    )

    aggregator = new MockAggregator(
      blockDB,
      rollupStateMachine,
      'localhost',
      3000,
      AGGREGATOR_MNEMONIC
    )
    await aggregator.listen()
    // Connect to the mock aggregator
    client = new SimpleClient('http://127.0.0.1:3000')
  })

  afterEach(async () => {
    // Close the server
    await aggregator.close()
    await stateDB.close()
    await blockDB.close()
  })

  const sendFromAliceToBob = async (
    amount
  ): Promise<SignedTransactionReceipt> => {
    const bobBeforeBalances = await client.handle('getBalances', 'bob')
    const transaction = {
      tokenType: UNI_TOKEN_TYPE,
      recipient: 'bob',
      amount,
    }
    const signature = await aliceWallet.signMessage(JSON.stringify(transaction))
    const txAliceToBob = {
      signature,
      transaction,
    }
    // Send some money to bob
    const receipt: SignedTransactionReceipt = await client.handle(
      'applyTransaction',
      txAliceToBob
    )
    // Make sure bob got the money!
    const bobAfterBalances = await client.handle('getBalances', 'bob')
    const uniDiff = bobAfterBalances.uni - bobBeforeBalances.uni
    uniDiff.should.equal(amount)

    return receipt
  }

  describe('getBalances', () => {
    it('should allow the balance to be queried', async () => {
      const response = await client.handle('getBalances', aliceWallet.address)
      response.should.deep.equal({
        uni: 50,
        pigi: 50,
      })
    })
  })

  describe('applyTransaction', () => {
    it('should update bobs balance using applyTransaction to send 5 tokens', async () => {
      await sendFromAliceToBob(5)
    })
  })

  describe('requestFaucetFunds', () => {
    it('should send money to the account who requested', async () => {
      const newWallet: ethers.Wallet = ethers.Wallet.createRandom()

      // Request some money for new wallet
      const transaction: FaucetRequest = {
        requester: newWallet.address,
        amount: 10,
      }
      const signature = await newWallet.signMessage(
        serializeObject(transaction)
      )
      const signedRequest: SignedTransaction = {
        signature,
        transaction,
      }

      await client.handle('requestFaucetFunds', signedRequest)
      // Make sure new wallet got the money!
      const newWalletBalances = await client.handle(
        'getBalances',
        newWallet.address
      )
      newWalletBalances.uni.should.equal(10)
      newWalletBalances.pigi.should.equal(10)
    })
  })

  describe('Transaction Receipt Tests', () => {
    it('should receive a transaction receipt signed by the aggregator', async () => {
      const receipt: SignedTransactionReceipt = await sendFromAliceToBob(5)
      const signer: string = DefaultSignatureVerifier.instance().verifyMessage(
        serializeObject(receipt.transactionReceipt),
        receipt.aggregatorSignature
      )

      signer.should.equal(AGGREGATOR_ADDRESS)
    })

    it('should have subsequent transactions that build on one another', async () => {
      const receiptOne: SignedTransactionReceipt = await sendFromAliceToBob(5)
      const receiptTwo: SignedTransactionReceipt = await sendFromAliceToBob(5)

      const blockOne = receiptOne.transactionReceipt.blockNumber
      const blockTwo = receiptTwo.transactionReceipt.blockNumber
      blockOne.should.equal(blockTwo)

      const indexOne = receiptOne.transactionReceipt.transitionIndex
      const indexTwo = receiptTwo.transactionReceipt.transitionIndex
      indexOne.should.equal(indexTwo - 1)

      const oneEnd = receiptOne.transactionReceipt.endRoot
      const twoStart = receiptTwo.transactionReceipt.startRoot
      oneEnd.should.equal(twoStart)
    })
  })
})
