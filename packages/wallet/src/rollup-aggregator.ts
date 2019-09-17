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
  SignedStateReceipt,
  StateSnapshot,
  Signature,
  StateReceipt,
} from './index'
import { ethers } from 'ethers'
import { RollupStateMachine } from './types'

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
export class RollupAggregator extends SimpleServer {
  private static readonly lockKey: string = 'lock'

  private readonly db: DB
  private readonly lock: AsyncLock
  private readonly wallet: ethers.Wallet
  private readonly rollupStateMachine: RollupStateMachine
  private readonly signatureProvider: SignatureProvider
  private readonly signatureVerifier: SignatureVerifier

  private blockNumber: number
  private transitionIndex: number
  private pendingBlock: RollupBlock

  constructor(
    db: DB,
    rollupStateMachine: RollupStateMachine,
    hostname: string,
    port: number,
    mnemonic: string,
    signatureVerifier: SignatureVerifier = DefaultSignatureVerifier.instance(),
    middleware?: Function[]
  ) {
    // REST API for our aggregator
    const methods = {
      [AGGREGATOR_API.getState]: async (
        account: Address
      ): Promise<SignedStateReceipt> => this.getState(account),

      [AGGREGATOR_API.getUniswapState]: async (): Promise<SignedStateReceipt> =>
        this.getState(UNISWAP_ADDRESS),

      [AGGREGATOR_API.applyTransaction]: async (
        signedTransaction: SignedTransaction
      ): Promise<SignedTransactionReceipt> =>
        this.applyTransaction(signedTransaction),

      [AGGREGATOR_API.requestFaucetFunds]: async (
        signedTransaction: SignedTransaction
      ): Promise<SignedTransactionReceipt> =>
        this.requestFaucetFunds(signedTransaction),
    }
    super(methods, hostname, port, middleware)
    this.rollupStateMachine = rollupStateMachine
    this.wallet = ethers.Wallet.fromMnemonic(mnemonic)
    this.signatureVerifier = signatureVerifier
    this.signatureProvider = new DefaultSignatureProvider(this.wallet)
    this.db = db
    this.transitionIndex = 0
    this.blockNumber = 0
    this.pendingBlock = {
      number: ++this.blockNumber,
      transitions: [],
    }
    this.lock = new AsyncLock()
  }

  /**
   * Gets the State for the provided address if State exists.
   *
   * @param address The address in question
   * @returns The SignedStateReceipt containing the state and the aggregator
   * guarantee that it exists. If it does not exist, this will include the
   * aggregator guarantee that it does not exist.
   */
  private async getState(address: string): Promise<SignedStateReceipt> {
    const stateReceipt: StateReceipt = await this.lock.acquire(
      RollupAggregator.lockKey,
      async () => {
        const snapshot: StateSnapshot = await this.rollupStateMachine.getState(
          address
        )
        return {
          blockNumber: this.blockNumber,
          transitionIndex: this.transitionIndex,
          ...snapshot,
        }
      }
    )

    const signature: Signature = await this.signatureProvider.sign(
      AGGREGATOR_ADDRESS,
      serializeObject(stateReceipt)
    )

    return {
      stateReceipt,
      signature,
    }
  }

  /**
   * Handles the provided transaction and returns the updated state and block and
   * transition in which it will be updated, guaranteed by the aggregator's signature.
   *
   * @param signedTransaction The transaction to apply
   * @returns The SignedTransactionReceipt
   */
  private async applyTransaction(
    signedTransaction
  ): Promise<SignedTransactionReceipt> {
    const [stateUpdate, transition] = await this.lock.acquire(
      RollupAggregator.lockKey,
      async () => {
        const update: StateUpdate = await this.rollupStateMachine.applyTransaction(
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
  }

  /**
   * Requests faucet funds on behalf of the requester and returns the updated
   * state resulting from the faucet allocation, including the guarantee that
   * it will be included in a specific block and transition.
   *
   * @param signedTransaction The faucet transaction
   * @returns The SignedTransactionReceipt
   */
  private async requestFaucetFunds(
    signedTransaction: SignedTransaction
  ): Promise<SignedTransactionReceipt> {
    if (!isFaucetTransaction(signedTransaction.transaction)) {
      throw Error('Cannot handle non-Faucet Request in faucet endpoint')
    }
    const messageSigner: Address = this.signatureVerifier.verifyMessage(
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
      this.wallet.address,
      this.signatureProvider
    )

    const [stateUpdate, transition] = await this.lock.acquire(
      RollupAggregator.lockKey,
      async () => {
        // Apply the two txs
        const update: StateUpdate = await this.rollupStateMachine.applyTransactions(
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

    const signature: string = await this.signatureProvider.sign(
      AGGREGATOR_ADDRESS,
      serializeObject(transactionReceipt)
    )
    return {
      signature,
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
      number: this.transitionIndex++,
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
    return this.lock.acquire(RollupAggregator.lockKey, async () => {
      const toSubmit = this.pendingBlock

      // TODO: submit block here

      this.pendingBlock = {
        number: ++this.blockNumber,
        transitions: [],
      }
      this.transitionIndex = 0
    })
  }

  private getDBKeyFromNumber(num: number): Buffer {
    const buff = Buffer.alloc(256)
    buff.writeUInt32BE(num, 0)
    return buff
  }
}
