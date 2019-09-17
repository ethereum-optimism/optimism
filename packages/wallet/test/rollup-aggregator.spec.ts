import './setup'
import MemDown from 'memdown'

/* External Imports */
import {
  BaseDB,
  BigNumber,
  DB,
  DefaultSignatureVerifier,
  serializeObject,
  SimpleClient,
  SparseMerkleTree,
  SparseMerkleTreeImpl,
} from '@pigi/core'

/* Internal Imports */
import { ethers } from 'ethers'
import { AGGREGATOR_MNEMONIC, getGenesisState } from './helpers'
import {
  UNI_TOKEN_TYPE,
  DefaultRollupStateMachine,
  FaucetRequest,
  SignedTransaction,
  SignedTransactionReceipt,
  AGGREGATOR_ADDRESS,
  AGGREGATOR_API,
  SignedStateReceipt,
  RollupAggregator,
  RollupStateMachine,
} from '../src'

/*********
 * TESTS *
 *********/

describe('RollupAggregator', () => {
  let client
  let aggregator: RollupAggregator
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

    aggregator = new RollupAggregator(
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

  const sendFromAToB = async (
    senderWallet: ethers.Wallet,
    recipient: string,
    amount: number
  ): Promise<SignedTransactionReceipt> => {
    const transaction = {
      tokenType: UNI_TOKEN_TYPE,
      recipient,
      amount,
    }
    const signature = await senderWallet.signMessage(
      JSON.stringify(transaction)
    )
    const tx = {
      signature,
      transaction,
    }
    return client.handle(AGGREGATOR_API.applyTransaction, tx)
  }

  const sendFromAliceToBob = async (
    amount
  ): Promise<SignedTransactionReceipt> => {
    const bobAddress: string = 'bob'
    const beforeState: SignedStateReceipt = await client.handle(
      AGGREGATOR_API.getState,
      bobAddress
    )
    const receipt = await sendFromAToB(aliceWallet, bobAddress, amount)

    // Make sure bob got the money!
    const afterState: SignedStateReceipt = await client.handle(
      AGGREGATOR_API.getState,
      bobAddress
    )
    if (!!beforeState.stateReceipt.state) {
      const uniDiff =
        afterState.stateReceipt.state[bobAddress].balances.uni -
        beforeState.stateReceipt.state[bobAddress].balances.uni
      uniDiff.should.equal(amount)
    } else {
      afterState.stateReceipt.state[bobAddress].balances.uni.should.equal(
        amount
      )
    }

    return receipt
  }

  const requestFaucetFundsForNewWallet = async (
    amount: number
  ): Promise<ethers.Wallet> => {
    const newWallet: ethers.Wallet = ethers.Wallet.createRandom()

    // Request some money for new wallet
    const transaction: FaucetRequest = {
      requester: newWallet.address,
      amount,
    }
    const signature = await newWallet.signMessage(serializeObject(transaction))
    const signedRequest: SignedTransaction = {
      signature,
      transaction,
    }

    await client.handle(AGGREGATOR_API.requestFaucetFunds, signedRequest)
    // Make sure new wallet got the money!
    const newWalletState: SignedStateReceipt = await client.handle(
      AGGREGATOR_API.getState,
      newWallet.address
    )
    newWalletState.stateReceipt.state[
      newWallet.address
    ].balances.uni.should.equal(amount)
    newWalletState.stateReceipt.state[
      newWallet.address
    ].balances.pigi.should.equal(amount)

    return newWallet
  }

  describe('getState', () => {
    it('should allow the balance to be queried', async () => {
      const response: SignedStateReceipt = await client.handle(
        AGGREGATOR_API.getState,
        aliceWallet.address
      )
      response.stateReceipt.state[
        aliceWallet.address
      ].balances.should.deep.equal({
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
      await requestFaucetFundsForNewWallet(10)
    })
  })

  describe('Transaction Receipt Tests', () => {
    it('should receive a transaction receipt signed by the aggregator', async () => {
      const receipt: SignedTransactionReceipt = await sendFromAliceToBob(5)
      const signer: string = DefaultSignatureVerifier.instance().verifyMessage(
        serializeObject(receipt.transactionReceipt),
        receipt.signature
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

  // describe('benchmarks', () => {
  //   const runTransactionTest = async (numTxs: number): Promise<void> => {
  //     const wallet: ethers.Wallet = await requestFaucetFundsForNewWallet(numTxs)
  //     const promises: Array<Promise<SignedTransactionReceipt>> = []
  //
  //     const startTime = +new Date()
  //
  //     for (let i = 0; i < numTxs; i++) {
  //       promises.push(sendFromAToB(wallet, 'does not matter', 1))
  //     }
  //
  //     await Promise.all(promises)
  //
  //     const finishTime = +new Date()
  //     const durationInMiliseconds = finishTime - startTime
  //     // tslint:disable-next-line:no-console
  //     console.log(
  //       'Duration:',
  //       durationInMiliseconds,
  //       ', Aggregator TPS: ',
  //       numTxs / (durationInMiliseconds / 1_000.0)
  //     )
  //   }
  //
  //   it('Applies 100 Aggregator Transactions', async () => {
  //     await runTransactionTest(100)
  //   }).timeout(20_000)
  // })
})
