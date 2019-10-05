import '../setup'

/* External Imports */
import {
  DB,
  DefaultSignatureProvider,
  DefaultSignatureVerifier,
  getLogger,
  hexStrToBuf,
  keccak256,
  newInMemoryDB,
  serializeObject,
  serializeObjectAsHexString,
  SimpleClient,
  sleep,
} from '@pigi/core'

/* Internal Imports */
import { ethers, Wallet } from 'ethers'
import {
  AGGREGATOR_MNEMONIC,
  assertThrowsAsync,
  BOB_ADDRESS,
  DummyBlockSubmitter,
  getGenesisState,
} from '../helpers'
import {
  UNI_TOKEN_TYPE,
  DefaultRollupStateMachine,
  FaucetRequest,
  SignedTransaction,
  AGGREGATOR_ADDRESS,
  AGGREGATOR_API,
  SignedStateReceipt,
  RollupAggregator,
  RollupStateMachine,
  Transfer,
  PIGI_TOKEN_TYPE,
  abiEncodeStateReceipt,
  abiEncodeTransaction,
  TransferTransition,
  AggregatorServer,
  abiEncodeTransition,
} from '../../src'

const log = getLogger('rollup-aggregator', true)
/*********
 * TESTS *
 *********/

describe('RollupAggregator', () => {
  let client
  let aggregatorDB: DB
  let aggregator: RollupAggregator
  let aggregatorServer: AggregatorServer
  let rollupStateMachine: RollupStateMachine
  let dummyBlockSubmitter: DummyBlockSubmitter

  let aliceWallet: ethers.Wallet

  beforeEach(async () => {
    aliceWallet = ethers.Wallet.createRandom()

    rollupStateMachine = await DefaultRollupStateMachine.create(
      getGenesisState(aliceWallet.address),
      newInMemoryDB()
    )

    aggregatorDB = newInMemoryDB()
    dummyBlockSubmitter = new DummyBlockSubmitter()
  })

  describe('server tests', () => {
    beforeEach(async () => {
      aggregator = await RollupAggregator.create(
        aggregatorDB,
        rollupStateMachine,
        dummyBlockSubmitter,
        new DefaultSignatureProvider(Wallet.fromMnemonic(AGGREGATOR_MNEMONIC)),
        DefaultSignatureVerifier.instance(),
        2
      )
      aggregatorServer = new AggregatorServer(aggregator, 'localhost', 3000)

      await aggregatorServer.listen()
      // Connect to the mock aggregator
      client = new SimpleClient('http://127.0.0.1:3000')
    })

    afterEach(async () => {
      // Close the server
      await aggregatorServer.close()
    })

    const sendFromAToB = async (
      senderWallet: ethers.Wallet,
      recipient: string,
      amount: number
    ): Promise<SignedStateReceipt[]> => {
      const transaction: Transfer = {
        sender: senderWallet.address,
        tokenType: UNI_TOKEN_TYPE,
        recipient,
        amount,
      }
      const signature = await senderWallet.signMessage(
        // right now, we are actually signing the hash of our messages to make the contract work.  (See DefaultSignatureProvider)
        hexStrToBuf(ethers.utils.keccak256(abiEncodeTransaction(transaction)))
      )
      const tx = {
        signature,
        transaction,
      }
      return client.handle(AGGREGATOR_API.applyTransaction, tx)
    }

    const sendFromAliceToBob = async (
      amount
    ): Promise<SignedStateReceipt[]> => {
      const beforeState: SignedStateReceipt = await client.handle(
        AGGREGATOR_API.getState,
        BOB_ADDRESS
      )
      log.debug(`Got before state ${serializeObject(beforeState)}`)
      const receipts: SignedStateReceipt[] = await sendFromAToB(
        aliceWallet,
        BOB_ADDRESS,
        amount
      )
      log.debug(`Got tx receipts state ${serializeObject(receipts)}`)
      // Make sure bob got the money!
      const afterState: SignedStateReceipt = await client.handle(
        AGGREGATOR_API.getState,
        BOB_ADDRESS
      )
      log.debug(`Got after state ${serializeObject(afterState)}`)
      if (!!beforeState.stateReceipt.state) {
        const uniDiff =
          afterState.stateReceipt.state.balances[UNI_TOKEN_TYPE] -
          beforeState.stateReceipt.state.balances[UNI_TOKEN_TYPE]
        uniDiff.should.equal(amount)
      } else {
        afterState.stateReceipt.state.balances[UNI_TOKEN_TYPE].should.equal(
          amount
        )
      }

      return receipts
    }

    const requestFaucetFundsForNewWallet = async (
      amount: number
    ): Promise<ethers.Wallet> => {
      const newWallet: ethers.Wallet = ethers.Wallet.createRandom()

      // Request some money for new wallet
      const transaction: FaucetRequest = {
        sender: newWallet.address,
        amount,
      }
      const signature = await newWallet.signMessage(
        // right now, we are actually signing the hash of our messages to make the contract work.  (See DefaultSignatureProvider)
        hexStrToBuf(
          ethers.utils.keccak256(serializeObjectAsHexString(transaction))
        )
      )
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
      newWalletState.stateReceipt.state.balances[UNI_TOKEN_TYPE].should.equal(
        amount
      )
      newWalletState.stateReceipt.state.balances[PIGI_TOKEN_TYPE].should.equal(
        amount
      )

      return newWallet
    }

    describe('getState', () => {
      it('should allow the balance to be queried', async () => {
        await aggregator.onSyncCompleted()

        const response: SignedStateReceipt = await client.handle(
          AGGREGATOR_API.getState,
          aliceWallet.address
        )
        response.stateReceipt.state.balances.should.deep.equal({
          [UNI_TOKEN_TYPE]: 50,
          [PIGI_TOKEN_TYPE]: 50,
        })
      })

      it('should throw if aggregator is not synced', async () => {
        await assertThrowsAsync(async () => {
          await client.handle(AGGREGATOR_API.getState, aliceWallet.address)
        })
      })
    })

    describe('applyTransaction', () => {
      it('should update bobs balance using applyTransaction to send 5 tokens', async () => {
        await aggregator.onSyncCompleted()

        await sendFromAliceToBob(5)

        const transIndex = parseInt(
          (await aggregatorDB.get(
            RollupAggregator.LAST_TRANSITION_KEY
          )).toString(),
          10
        )
        transIndex.should.equal(1)

        aggregator.getPendingBlockNumber().should.equal(1)
      })

      it('should throw if aggregator is not synced', async () => {
        await assertThrowsAsync(async () => {
          await sendFromAliceToBob(5)
        })
      })
    })

    describe('requestFaucetFunds', () => {
      it('should send money to the account who requested', async () => {
        await aggregator.onSyncCompleted()
        await requestFaucetFundsForNewWallet(10)

        const transIndex = parseInt(
          (await aggregatorDB.get(
            RollupAggregator.LAST_TRANSITION_KEY
          )).toString(),
          10
        )

        // It should have submitted a block
        transIndex.should.equal(0)
        dummyBlockSubmitter.submitedBlocks.length.should.equal(1)
        dummyBlockSubmitter.submitedBlocks[0].blockNumber.should.equal(1)
        dummyBlockSubmitter.submitedBlocks[0].transitions.length.should.equal(2)

        aggregator.getPendingBlockNumber().should.equal(2)
      })

      it('should throw if aggregator is not synced', async () => {
        await assertThrowsAsync(async () => {
          await requestFaucetFundsForNewWallet(10)
        })
      })
    })

    describe('RollupTransaction Receipt Tests', () => {
      it('should receive a transaction receipt signed by the aggregator', async () => {
        await aggregator.onSyncCompleted()

        const receipts: SignedStateReceipt[] = await sendFromAliceToBob(5)
        const signer0: string = DefaultSignatureVerifier.instance().verifyMessage(
          abiEncodeStateReceipt(receipts[0].stateReceipt),
          receipts[0].signature
        )

        signer0.should.equal(AGGREGATOR_ADDRESS)

        const signer1: string = DefaultSignatureVerifier.instance().verifyMessage(
          abiEncodeStateReceipt(receipts[1].stateReceipt),
          receipts[1].signature
        )

        signer1.should.equal(AGGREGATOR_ADDRESS)
      })

      it('should have subsequent transactions that build on one another', async () => {
        await aggregator.onSyncCompleted()

        const receiptsOne: SignedStateReceipt[] = await sendFromAliceToBob(5)
        const receiptsTwo: SignedStateReceipt[] = await sendFromAliceToBob(5)

        const blockOne0 = receiptsOne[0].stateReceipt.blockNumber
        const blockTwo0 = receiptsTwo[0].stateReceipt.blockNumber
        blockOne0.should.equal(blockTwo0)

        const blockOne1 = receiptsOne[1].stateReceipt.blockNumber
        const blockTwo1 = receiptsTwo[1].stateReceipt.blockNumber
        blockOne1.should.equal(blockTwo1)

        const indexOne0 = receiptsOne[0].stateReceipt.transitionIndex
        const indexTwo0 = receiptsTwo[0].stateReceipt.transitionIndex
        indexOne0.should.equal(indexTwo0 - 1)

        const indexOne1 = receiptsOne[1].stateReceipt.transitionIndex
        const indexTwo1 = receiptsTwo[1].stateReceipt.transitionIndex
        indexOne1.should.equal(indexTwo1 - 1)
      })
    })
  })

  describe('aggregator init tests', () => {
    let trans1: TransferTransition
    beforeEach(async () => {
      trans1 = {
        stateRoot: keccak256(Buffer.from('trans 1').toString('hex')),
        senderSlotIndex: 1,
        recipientSlotIndex: 0,
        tokenType: 0,
        amount: 10,
        signature: await new DefaultSignatureProvider().sign('trans 1'),
      }
    })

    it('should init without any DB state', async () => {
      aggregator = await RollupAggregator.create(
        aggregatorDB,
        rollupStateMachine,
        dummyBlockSubmitter,
        new DefaultSignatureProvider(Wallet.fromMnemonic(AGGREGATOR_MNEMONIC)),
        DefaultSignatureVerifier.instance(),
        2
      )

      aggregator.getPendingBlockNumber().should.equal(1)
      aggregator.getNextTransitionIndex().should.equal(0)

      dummyBlockSubmitter.submitedBlocks.length.should.equal(0)
    })

    it('should init with pending transition', async () => {
      await aggregatorDB.put(
        RollupAggregator.LAST_TRANSITION_KEY,
        Buffer.from('1')
      )
      await aggregatorDB.put(
        RollupAggregator.getTransitionKey(1),
        hexStrToBuf(abiEncodeTransition(trans1))
      )

      aggregator = await RollupAggregator.create(
        aggregatorDB,
        rollupStateMachine,
        dummyBlockSubmitter,
        new DefaultSignatureProvider(Wallet.fromMnemonic(AGGREGATOR_MNEMONIC)),
        DefaultSignatureVerifier.instance(),
        2
      )

      aggregator.getPendingBlockNumber().should.equal(1)
      aggregator.getNextTransitionIndex().should.equal(1)

      dummyBlockSubmitter.submitedBlocks.length.should.equal(0)
    })

    it('should init with pending transition in pending block > 1', async () => {
      await aggregatorDB.put(
        RollupAggregator.PENDING_BLOCK_KEY,
        Buffer.from('2')
      )
      await aggregatorDB.put(
        RollupAggregator.LAST_TRANSITION_KEY,
        Buffer.from('1')
      )
      await aggregatorDB.put(
        RollupAggregator.getTransitionKey(1),
        hexStrToBuf(abiEncodeTransition(trans1))
      )

      aggregator = await RollupAggregator.create(
        aggregatorDB,
        rollupStateMachine,
        dummyBlockSubmitter,
        new DefaultSignatureProvider(Wallet.fromMnemonic(AGGREGATOR_MNEMONIC)),
        DefaultSignatureVerifier.instance(),
        2
      )

      aggregator.getPendingBlockNumber().should.equal(2)
      aggregator.getNextTransitionIndex().should.equal(1)

      dummyBlockSubmitter.submitedBlocks.length.should.equal(0)
    })
  })

  describe('block submission delay', () => {
    let trans1: TransferTransition
    beforeEach(async () => {
      trans1 = {
        stateRoot: keccak256(Buffer.from('trans 1').toString('hex')),
        senderSlotIndex: 1,
        recipientSlotIndex: 0,
        tokenType: 0,
        amount: 10,
        signature: await new DefaultSignatureProvider().sign('trans 1'),
      }
    })

    it('should init without any DB state', async () => {
      await aggregatorDB.put(
        RollupAggregator.LAST_TRANSITION_KEY,
        Buffer.from('1')
      )
      await aggregatorDB.put(
        RollupAggregator.getTransitionKey(1),
        hexStrToBuf(abiEncodeTransition(trans1))
      )

      aggregator = await RollupAggregator.create(
        aggregatorDB,
        rollupStateMachine,
        dummyBlockSubmitter,
        new DefaultSignatureProvider(Wallet.fromMnemonic(AGGREGATOR_MNEMONIC)),
        DefaultSignatureVerifier.instance(),
        2,
        1_000
      )

      // Block submission delay is set to 1s, so sleep and assert it was submitted.
      await sleep(1_900)

      dummyBlockSubmitter.submitedBlocks.length.should.equal(1)
      dummyBlockSubmitter.submitedBlocks[0].blockNumber.should.equal(1)
      dummyBlockSubmitter.submitedBlocks[0].transitions.length.should.equal(1)
    }).timeout(10_000)
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
