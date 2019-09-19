/* External Imports */
import * as AsyncLock from 'async-lock'
import { ethers } from 'ethers'

import {
  SignatureVerifier,
  DefaultSignatureVerifier,
  SimpleServer,
  serializeObject,
  DefaultSignatureProvider,
  DB,
  getLogger,
} from '@pigi/core'

/* Internal Imports */
import {
  Address,
  SignedTransaction,
  Balances,
  UNI_TOKEN_TYPE,
  PIGI_TOKEN_TYPE,
  generateTransferTx,
  AGGREGATOR_API,
  RollupTransaction,
  SignatureProvider,
  UNISWAP_ADDRESS,
  AGGREGATOR_ADDRESS,
  RollupTransition,
  StateUpdate,
  isFaucetTransaction,
  RollupBlock,
  SignedStateReceipt,
  StateSnapshot,
  Signature,
  StateReceipt,
  abiEncodeStateReceipt,
  isSwapTransaction,
  isTransferTransaction,
  Transfer,
  abiEncodeTransition,
  TransferTransition,
  abiEncodeTransaction,
  EMPTY_AGGREGATOR_SIGNATURE,
} from './index'
import { RollupStateMachine } from './types'

const log = getLogger('rollup-aggregator')

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
  const txOne: RollupTransaction = generateTransferTx(
    UNISWAP_ADDRESS,
    recipient,
    UNI_TOKEN_TYPE,
    amount
  )
  const txTwo: RollupTransaction = generateTransferTx(
    UNISWAP_ADDRESS,
    recipient,
    PIGI_TOKEN_TYPE,
    amount
  )

  return [
    {
      signature: await signatureProvider.sign(
        aggregatorAddress,
        abiEncodeTransaction(txOne)
      ),
      transaction: txOne,
    },
    {
      signature: await signatureProvider.sign(
        aggregatorAddress,
        abiEncodeTransaction(txTwo)
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
      ): Promise<SignedStateReceipt[]> =>
        this.applyTransaction(signedTransaction),

      [AGGREGATOR_API.requestFaucetFunds]: async (
        signedTransaction: SignedTransaction
      ): Promise<SignedStateReceipt> =>
        this.requestFaucetFunds(signedTransaction),
    }
    super(methods, hostname, port, middleware)
    this.rollupStateMachine = rollupStateMachine
    this.wallet = ethers.Wallet.fromMnemonic(mnemonic)
    this.signatureVerifier = signatureVerifier
    this.signatureProvider = new DefaultSignatureProvider(this.wallet)
    this.db = db
    this.transitionIndex = 0
    this.blockNumber = 1
    this.pendingBlock = {
      number: this.blockNumber,
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
    try {
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
      let signature: Signature
      if (!!stateReceipt.state) {
        signature = await this.signatureProvider.sign(
          AGGREGATOR_ADDRESS,
          abiEncodeStateReceipt(stateReceipt)
        )
      } else {
        signature = EMPTY_AGGREGATOR_SIGNATURE
      }

      return {
        stateReceipt,
        signature,
      }
    } catch (e) {
      log.error(
        `Error getting state for address [${address}]! ${e.message}, ${e.stack}`
      )
      throw e
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
  ): Promise<SignedStateReceipt[]> {
    try {
      const [
        stateUpdate,
        blockNumber,
        transitionIndex,
      ] = await this.lock.acquire(RollupAggregator.lockKey, async () => {
        const update: StateUpdate = await this.rollupStateMachine.applyTransaction(
          signedTransaction
        )
        await this.addToPendingBlock([update], signedTransaction)
        return [update, this.blockNumber, this.transitionIndex]
      })

      return this.respond(stateUpdate, blockNumber, transitionIndex)
    } catch (e) {
      log.error(
        `Error applying transaction [${serializeObject(signedTransaction)}]! ${
          e.message
        }, ${e.stack}`
      )
      throw e
    }
  }

  /**
   * Requests faucet funds on behalf of the sender and returns the updated
   * state resulting from the faucet allocation, including the guarantee that
   * it will be included in a specific block and transition.
   *
   * @param signedTransaction The faucet transaction
   * @returns The SignedTransactionReceipt
   */
  private async requestFaucetFunds(
    signedTransaction: SignedTransaction
  ): Promise<SignedStateReceipt> {
    try {
      if (!isFaucetTransaction(signedTransaction.transaction)) {
        throw Error('Cannot handle non-Faucet Request in faucet endpoint')
      }
      const messageSigner: Address = this.signatureVerifier.verifyMessage(
        serializeObject(signedTransaction.transaction),
        signedTransaction.signature
      )
      if (messageSigner !== signedTransaction.transaction.sender) {
        throw Error(
          `Faucet requests must be signed by the request address. Signer address: ${messageSigner}, sender: ${signedTransaction.transaction.sender}`
        )
      }

      // TODO: Probably need to check amount before blindly giving them this amount

      const { sender, amount } = signedTransaction.transaction
      // Generate the faucet txs (one sending uni the other pigi)
      const faucetTxs = await generateFaucetTxs(
        sender, // original tx sender... is actually faucet fund recipient
        amount,
        this.wallet.address,
        this.signatureProvider
      )

      const [
        stateUpdate,
        blockNumber,
        transitionIndex,
      ] = await this.lock.acquire(RollupAggregator.lockKey, async () => {
        // Apply the two txs
        const updates: StateUpdate[] = await this.rollupStateMachine.applyTransactions(
          faucetTxs
        )

        await this.addToPendingBlock(updates, signedTransaction)
        return [
          updates[updates.length - 1],
          this.blockNumber,
          this.transitionIndex,
        ]
      })

      return (await this.respond(stateUpdate, blockNumber, transitionIndex))[1]
    } catch (e) {
      log.error(
        `Error handling faucet request [${serializeObject(
          signedTransaction
        )}]! ${e.message}, ${e.stack}`
      )
      throw e
    }
  }

  /**
   * Responds to the provided RollupTransaction according to the provided resulting state
   * update and rollup transition.
   *
   * @param stateUpdate The state update that resulted from this transaction
   * @param blockNumber The block number of this update
   * @param transitionIndex The transition index of this update
   * @returns The signed state receipt objects for the
   */
  private async respond(
    stateUpdate: StateUpdate,
    blockNumber: number,
    transitionIndex: number
  ): Promise<SignedStateReceipt[]> {
    const receipts: SignedStateReceipt[] = []

    const senderReceipt: StateReceipt = {
      slotIndex: stateUpdate.senderSlotIndex,
      stateRoot: stateUpdate.stateRoot,
      state: stateUpdate.senderState,
      inclusionProof: stateUpdate.senderStateInclusionProof,
      blockNumber,
      transitionIndex,
    }
    const senderSignature: string = await this.signatureProvider.sign(
      AGGREGATOR_ADDRESS,
      abiEncodeStateReceipt(senderReceipt)
    )
    receipts.push({
      signature: senderSignature,
      stateReceipt: senderReceipt,
    })

    if (stateUpdate.receiverState.pubKey !== UNISWAP_ADDRESS) {
      const recipientReceipt: StateReceipt = {
        slotIndex: stateUpdate.receiverSlotIndex,
        stateRoot: stateUpdate.stateRoot,
        state: stateUpdate.receiverState,
        inclusionProof: stateUpdate.receiverStateInclusionProof,
        blockNumber,
        transitionIndex,
      }
      const recipientSignature: string = await this.signatureProvider.sign(
        AGGREGATOR_ADDRESS,
        abiEncodeStateReceipt(recipientReceipt)
      )
      receipts.push({
        signature: recipientSignature,
        stateReceipt: recipientReceipt,
      })
    }

    log.debug(`Returning receipts: ${serializeObject(receipts)}`)
    return receipts
  }

  /**
   * Adds and returns the pending transition(s) resulting from
   * the provided StateUpdate(s).
   *
   * @param updates The state updates in question
   * @param transaction The signed transaction received as input
   * @returns The rollup transition
   */
  private async addToPendingBlock(
    updates: StateUpdate[],
    transaction: SignedTransaction
  ): Promise<RollupTransition[]> {
    const transitions: RollupTransition[] = []

    if (isSwapTransaction(transaction.transaction)) {
      const update: StateUpdate = updates[0]
      transitions.push({
        stateRoot: update.stateRoot,
        senderSlotIndex: update.senderSlotIndex,
        uniswapSlotIndex: update.receiverSlotIndex,
        tokenType: transaction.transaction.tokenType,
        inputAmount: transaction.transaction.inputAmount,
        minOutputAmount: transaction.transaction.minOutputAmount,
        timeout: transaction.transaction.timeout,
        signature: transaction.signature,
      })
    } else {
      // It's a transfer -- either faucet or p2p
      for (const u of updates) {
        transitions.push(this.getTransferTransitionFromStateUpdate(u))
      }
    }

    for (const trans of transitions) {
      log.debug(`Adding Transition to pending block: ${serializeObject(trans)}`)
      await this.db
        .bucket(this.getDBKeyFromNumber(this.pendingBlock.number))
        .put(
          this.getDBKeyFromNumber(++this.transitionIndex),
          Buffer.from(abiEncodeTransition(trans))
        )
    }

    return transitions
  }

  /**
   * Creates a TransferTransition from the provided StateUpdate for a Transfer.
   * @param update The state update
   * @returns the TransferTransition
   */
  private getTransferTransitionFromStateUpdate(
    update: StateUpdate
  ): TransferTransition {
    const transfer = update.transaction.transaction as Transfer
    const transition = {
      stateRoot: update.stateRoot,
      senderSlotIndex: update.senderSlotIndex,
      recipientSlotIndex: update.receiverSlotIndex,
      tokenType: transfer.tokenType,
      amount: transfer.amount,
      signature: update.transaction.signature,
    }
    if (update.receiverCreated) {
      transition['createdAccountPubkey'] = update.receiverState.pubKey
    }
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
