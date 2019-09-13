/* External Imports */
import * as AsyncLock from 'async-lock'

import {
  SignatureVerifier,
  DefaultSignatureVerifier,
  SimpleServer,
  serializeObject,
  DefaultSignatureProvider,
  DB,
  objectToBuffer,
  SparseMerkleTree,
} from '@pigi/core'

/* Internal Imports */
import {
  Address,
  SignedTransaction,
  Balances,
  TransactionReceipt,
  UNI_TOKEN_TYPE,
  PIGI_TOKEN_TYPE,
  generateTransferTx,
  AGGREGATOR_API,
  Transaction,
  SignatureProvider,
  UNISWAP_ADDRESS,
  AGGREGATOR_ADDRESS,
  RollupTransition,
  StateUpdate,
  SignedTransactionReceipt,
  isFaucetTransaction,
  RollupBlock,
} from '../index'
import { ethers } from 'ethers'
import { RollupStateMachine } from '../types'

/*
 * Generate two transactions which together send the user some UNI
 * & some PIGI
 */
const generateFaucetTxs = async (
  recipient: Address,
  amount: number,
  aggregatorAddress: string = AGGREGATOR_ADDRESS,
  signatureProvider?: SignatureProvider
): Promise<SignedTransaction[]> => {
  const txOne: Transaction = generateTransferTx(
    recipient,
    UNI_TOKEN_TYPE,
    amount
  )
  const txTwo: Transaction = generateTransferTx(
    recipient,
    PIGI_TOKEN_TYPE,
    amount
  )

  return [
    {
      signature: await signatureProvider.sign(
        aggregatorAddress,
        serializeObject(txOne)
      ),
      transaction: txOne,
    },
    {
      signature: await signatureProvider.sign(
        aggregatorAddress,
        serializeObject(txTwo)
      ),
      transaction: txTwo,
    },
  ]
}

/*
 * A mock aggregator implementation which allows for transfers, swaps,
 * balance queries, & faucet requests
 */
export class MockAggregator extends SimpleServer {
  private static readonly lockKey: string = 'lock'

  private readonly db: DB
  private readonly lock: AsyncLock
  private blockNumber: number
  private transitionNumber: number
  private pendingBlock: RollupBlock
  private readonly rollupStateMachine: RollupStateMachine
  private readonly signatureProvider: SignatureProvider

  constructor(
    db: DB,
    rollupStateMachine: RollupStateMachine,
    hostname: string,
    port: number,
    mnemonic: string,
    signatureVerifier: SignatureVerifier = DefaultSignatureVerifier.instance(),
    middleware?: Function[]
  ) {
    const wallet: ethers.Wallet = ethers.Wallet.fromMnemonic(mnemonic)
    const signatureProvider: SignatureProvider = new DefaultSignatureProvider(
      wallet
    )

    // REST API for our aggregator
    const methods = {
      /*
       * Get balances for some account
       */
      [AGGREGATOR_API.getBalances]: async (
        account: Address
      ): Promise<Balances> => rollupStateMachine.getBalances(account),

      /*
       * Get balances for Uniswap
       */
      [AGGREGATOR_API.getUniswapBalances]: async (): Promise<Balances> =>
        rollupStateMachine.getBalances(UNISWAP_ADDRESS),

      /*
       * Apply either a transfer or swap transaction
       */
      [AGGREGATOR_API.applyTransaction]: async (
        signedTransaction: SignedTransaction
      ): Promise<SignedTransactionReceipt> => {
        const [stateUpdate, transition] = await this.lock.acquire(
          MockAggregator.lockKey,
          async () => {
            const update: StateUpdate = await rollupStateMachine.applyTransaction(
              signedTransaction
            )
            const trans: RollupTransition = await this.addToPendingBlock(
              update,
              signedTransaction
            )
            return [update, trans]
          }
        )

        return this.respond(stateUpdate, transition, signedTransaction)
      },

      /*
       * Request money from a faucet
       */
      [AGGREGATOR_API.requestFaucetFunds]: async (
        signedTransaction: SignedTransaction
      ): Promise<SignedTransactionReceipt> => {
        if (!isFaucetTransaction(signedTransaction.transaction)) {
          throw Error('Cannot handle non-Faucet Request in faucet endpoint')
        }
        const messageSigner: Address = signatureVerifier.verifyMessage(
          serializeObject(signedTransaction.transaction),
          signedTransaction.signature
        )
        if (messageSigner !== signedTransaction.transaction.requester) {
          throw Error('Faucet requests must be signed by the request address')
        }

        // TODO: Probably need to check amount before blindly giving them this amount

        const { requester, amount } = signedTransaction.transaction
        // Generate the faucet txs (one sending uni the other pigi)
        const faucetTxs = await generateFaucetTxs(
          requester,
          amount,
          wallet.address,
          signatureProvider
        )

        const [stateUpdate, transition] = await this.lock.acquire(
          MockAggregator.lockKey,
          async () => {
            // Apply the two txs
            const update: StateUpdate = await rollupStateMachine.applyTransactions(
              faucetTxs
            )

            const trans: RollupTransition = await this.addToPendingBlock(
              update,
              signedTransaction
            )
            return [update, trans]
          }
        )

        return this.respond(stateUpdate, transition, signedTransaction)
      },
    }
    super(methods, hostname, port, middleware)
    this.rollupStateMachine = rollupStateMachine
    this.signatureProvider = signatureProvider
    this.db = db
    this.transitionNumber = 0
    this.blockNumber = 0
    this.pendingBlock = {
      number: ++this.blockNumber,
      transitions: [],
    }
    this.lock = new AsyncLock()
  }

  /**
   * Responds to the provided Transaction according to the provided resulting state
   * update and rollup transition.
   *
   * @param stateUpdate The state update that resulted from this transaction
   * @param transition The rollup transition for this transaction
   * @param transaction The transaction
   * @returns The signed transaction response
   */
  private async respond(
    stateUpdate: StateUpdate,
    transition: RollupTransition,
    transaction: SignedTransaction
  ): Promise<SignedTransactionReceipt> {
    const transactionReceipt: TransactionReceipt = {
      blockNumber: transition.blockNumber,
      transitionIndex: transition.number,
      transaction,
      startRoot: transition.startRoot,
      endRoot: transition.endRoot,
      updatedState: stateUpdate.updatedState,
      updatedStateInclusionProof: stateUpdate.updatedStateInclusionProof,
    }

    const aggregatorSignature: string = await this.signatureProvider.sign(
      AGGREGATOR_ADDRESS,
      serializeObject(transactionReceipt)
    )
    return {
      aggregatorSignature,
      transactionReceipt,
    }
  }

  /**
   * Adds and returns the pending transition resulting from the provided StateUpdate.
   *
   * @param update The state update in question
   * @param transaction The signed transaction received as input
   * @returns The rollup transition
   */
  private async addToPendingBlock(
    update: StateUpdate,
    transaction: SignedTransaction
  ): Promise<RollupTransition> {
    const transition: RollupTransition = {
      number: this.transitionNumber++,
      blockNumber: this.pendingBlock.number,
      transactions: [transaction],
      startRoot: update.startRoot,
      endRoot: update.endRoot,
    }

    await this.db
      .bucket(this.getDBKeyFromNumber(this.pendingBlock.number))
      .put(
        this.getDBKeyFromNumber(transition.number),
        objectToBuffer(transition)
      )

    return transition
  }

  /**
   * Submits a block to the main chain, creating a new pending block for future
   * transitions.
   */
  private async submitBlock(): Promise<void> {
    return this.lock.acquire(MockAggregator.lockKey, async () => {
      const toSubmit = this.pendingBlock

      // TODO: submit block here

      this.pendingBlock = {
        number: ++this.blockNumber,
        transitions: [],
      }
      this.transitionNumber = 0
    })
  }

  private getDBKeyFromNumber(num: number): Buffer {
    const buff = Buffer.alloc(256)
    buff.writeUInt32BE(num, 0)
    return buff
  }
}
