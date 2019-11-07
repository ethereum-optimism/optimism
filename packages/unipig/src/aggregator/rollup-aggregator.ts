/* External Imports */
import * as AsyncLock from 'async-lock'

import {
  getLogger,
  hexStrToBuf,
  hexBufToStr,
  logError,
  SignatureVerifier,
  DefaultSignatureVerifier,
  serializeObject,
  serializeObjectAsHexString,
  SignatureProvider,
} from '@pigi/core-utils'

import { DB, EthereumListener, EthereumEvent } from '@pigi/core-db'

/* Internal Imports */
import { UnipigAggregator } from '../types/unipig-aggregator'
import {
  Address,
  isFaucetTransaction,
  isSwapTransaction,
  NotSyncedError,
  RollupBlock,
  RollupBlockSubmitter,
  RollupStateMachine,
  RollupTransaction,
  RollupTransition,
  Signature,
  SignedStateReceipt,
  SignedTransaction,
  StateReceipt,
  StateSnapshot,
  StateUpdate,
  SwapTransition,
  Transfer,
  TransferTransition,
} from '../types'
import {
  abiEncodeStateReceipt,
  abiEncodeTransaction,
  abiEncodeTransition,
  parseTransitionFromABI,
} from '../common/serialization'
import {
  EMPTY_AGGREGATOR_SIGNATURE,
  generateTransferTx,
  PIGI_TOKEN_TYPE,
  UNI_TOKEN_TYPE,
  UNISWAP_ADDRESS,
} from '../common'

const log = getLogger('rollup-aggregator')

/*
 * An aggregator implementation which allows for transfers, swaps,
 * balance queries, & faucet requests.
 */
export class RollupAggregator
  implements EthereumListener<EthereumEvent>, UnipigAggregator {
  public static readonly PENDING_BLOCK_KEY: Buffer = Buffer.from(
    'pending_block_number'
  )
  public static readonly LAST_TRANSITION_KEY: Buffer = Buffer.from(
    'last_transition'
  )
  public static readonly TRANSACTION_COUNT_KEY: Buffer = Buffer.from('tx_count')
  public static readonly TX_COUNT_STORAGE_THRESHOLD: number = 7

  private static readonly lockKey: string = 'lock'

  private readonly lock: AsyncLock

  private synced: boolean

  private transactionCount: number
  private pendingBlock: RollupBlock
  private lastBlockSubmission: Date

  public static async create(
    db: DB,
    rollupStateMachine: RollupStateMachine,
    rollupBlockSubmitter: RollupBlockSubmitter,
    signatureProvider: SignatureProvider,
    signatureVerifier: SignatureVerifier = DefaultSignatureVerifier.instance(),
    blockSubmissionTransitionCount: number = 100,
    blockSubmissionIntervalMillis: number = 300_000,
    authorizedFaucetAddress?: Address
  ): Promise<RollupAggregator> {
    const aggregator = new RollupAggregator(
      db,
      rollupStateMachine,
      rollupBlockSubmitter,
      signatureProvider,
      signatureVerifier,
      blockSubmissionTransitionCount,
      blockSubmissionIntervalMillis,
      authorizedFaucetAddress
    )

    await aggregator.init()

    return aggregator
  }

  private constructor(
    private readonly db: DB,
    private readonly rollupStateMachine: RollupStateMachine,
    private readonly rollupBlockSubmitter: RollupBlockSubmitter,
    private readonly signatureProvider: SignatureProvider,
    private readonly signatureVerifier: SignatureVerifier = DefaultSignatureVerifier.instance(),
    private readonly blockSubmissionTransitionCount: number = 100,
    private readonly blockSubmissionIntervalMillis: number = 300_000,
    private readonly authorizedFaucetAddress?: Address
  ) {
    this.pendingBlock = {
      blockNumber: 1,
      transitions: [],
    }
    this.lock = new AsyncLock()
    this.synced = false
  }

  /**
   * Initialize method, required for the Aggregator to load existing state before
   * it can handle requests.
   */
  private async init(): Promise<void> {
    try {
      const [
        pendingBlockNumberBuffer,
        lastTransitionBuffer,
        txCountBuffer,
      ] = await Promise.all([
        this.db.get(RollupAggregator.PENDING_BLOCK_KEY),
        this.db.get(RollupAggregator.LAST_TRANSITION_KEY),
        this.db.get(RollupAggregator.TRANSACTION_COUNT_KEY),
      ])

      // Fresh start -- nothing in the DB
      if (!lastTransitionBuffer) {
        log.info(`Init returning -- no stored last transition.`)
        this.transactionCount = 0
        this.lastBlockSubmission = new Date()
        return
      }

      this.transactionCount = txCountBuffer
        ? parseInt(txCountBuffer.toString(), 10)
        : 0

      const pendingBlock: number = pendingBlockNumberBuffer
        ? parseInt(pendingBlockNumberBuffer.toString(), 10)
        : 1

      const lastTransition: number = parseInt(
        lastTransitionBuffer.toString(),
        10
      )

      const promises: Array<Promise<Buffer>> = []
      for (let i = 1; i <= lastTransition; i++) {
        promises.push(this.db.get(RollupAggregator.getTransitionKey(i)))
      }

      const transitionBuffers: Buffer[] = await Promise.all(promises)
      const transitions: RollupTransition[] = transitionBuffers.map((x) =>
        parseTransitionFromABI(hexBufToStr(x))
      )

      this.pendingBlock = {
        blockNumber: pendingBlock,
        transitions,
      }

      log.info(
        `Initialized aggregator with pending block: ${JSON.stringify(
          this.pendingBlock
        )}`
      )

      this.lastBlockSubmission = new Date()
      this.setBlockSubmissionTimeout()
    } catch (e) {
      logError(log, 'Error initializing aggregator', e)
      throw e
    }
  }

  public async onSyncCompleted(syncIdentifier?: string): Promise<void> {
    this.synced = true
  }

  public async handle(event: EthereumEvent): Promise<void> {
    log.debug(`Aggregator received event: ${JSON.stringify(event)}`)
    if (!!event && !!event.values && 'blockNumber' in event.values) {
      await this.rollupBlockSubmitter.handleNewRollupBlock(
        (event.values['blockNumber'] as any).toNumber()
      )
    }
  }

  public async getTransactionCount(): Promise<number> {
    return this.transactionCount
  }

  public async getState(address: string): Promise<SignedStateReceipt> {
    if (!this.synced) {
      throw new NotSyncedError()
    }

    try {
      const stateReceipt: StateReceipt = await this.lock.acquire(
        RollupAggregator.lockKey,
        async () => {
          const snapshot: StateSnapshot = await this.rollupStateMachine.getState(
            address
          )
          return {
            blockNumber: this.pendingBlock.blockNumber,
            transitionIndex: this.pendingBlock.transitions.length,
            ...snapshot,
          }
        }
      )
      let signature: Signature
      if (!!stateReceipt.state) {
        signature = await this.signatureProvider.sign(
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

  public async applyTransaction(
    signedTransaction: SignedTransaction
  ): Promise<SignedStateReceipt[]> {
    if (!this.synced) {
      throw new NotSyncedError()
    }

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
        return [
          update,
          this.pendingBlock.blockNumber,
          this.pendingBlock.transitions.length,
        ]
      })

      if (
        this.pendingBlock.transitions.length >=
        this.blockSubmissionTransitionCount
      ) {
        this.submitBlock()
      }

      await this.incrementTxCount()

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

  public async requestFaucetFunds(
    signedTransaction: SignedTransaction
  ): Promise<SignedStateReceipt> {
    if (!this.synced) {
      throw new NotSyncedError()
    }

    try {
      if (!isFaucetTransaction(signedTransaction.transaction)) {
        throw Error('Cannot handle non-Faucet Request in faucet endpoint')
      }
      const messageSigner: Address = this.signatureVerifier.verifyMessage(
        serializeObjectAsHexString(signedTransaction.transaction),
        signedTransaction.signature
      )
      const requiredSigner = !!this.authorizedFaucetAddress
        ? this.authorizedFaucetAddress
        : signedTransaction.transaction.sender
      if (messageSigner !== requiredSigner) {
        throw Error(
          `Faucet requests must be signed by the authorized faucet requester address. Signer address: ${messageSigner}, required address: ${requiredSigner}`
        )
      }

      // TODO: Probably need to check amount before blindly giving them this amount

      const { sender, amount } = signedTransaction.transaction
      // Generate the faucet txs (one sending uni the other pigi)
      const faucetTxs = await this.generateFaucetTxs(
        sender, // original tx sender... is actually faucet fund recipient
        amount
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
          this.pendingBlock.blockNumber,
          this.pendingBlock.transitions.length,
        ]
      })

      if (
        this.pendingBlock.transitions.length >=
        this.blockSubmissionTransitionCount
      ) {
        this.submitBlock()
      }

      await this.incrementTxCount()

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
      abiEncodeStateReceipt(senderReceipt)
    )
    receipts.push({
      signature: senderSignature,
      stateReceipt: senderReceipt,
    })

    if (stateUpdate.receiverState.pubkey !== UNISWAP_ADDRESS) {
      const recipientReceipt: StateReceipt = {
        slotIndex: stateUpdate.receiverSlotIndex,
        stateRoot: stateUpdate.stateRoot,
        state: stateUpdate.receiverState,
        inclusionProof: stateUpdate.receiverStateInclusionProof,
        blockNumber,
        transitionIndex,
      }
      const recipientSignature: string = await this.signatureProvider.sign(
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
      const transition: SwapTransition = {
        stateRoot: update.stateRoot,
        senderSlotIndex: update.senderSlotIndex,
        uniswapSlotIndex: update.receiverSlotIndex,
        tokenType: transaction.transaction.tokenType,
        inputAmount: transaction.transaction.inputAmount,
        minOutputAmount: transaction.transaction.minOutputAmount,
        timeout: transaction.transaction.timeout,
        signature: transaction.signature,
      }
      this.pendingBlock.transitions.push(transition)
      await this.db.put(
        RollupAggregator.getTransitionKey(this.pendingBlock.transitions.length),
        hexStrToBuf(abiEncodeTransition(transition))
      )
    } else {
      // It's a transfer -- either faucet or p2p
      for (const u of updates) {
        const transition: TransferTransition = this.getTransferTransitionFromStateUpdate(
          u
        )
        this.pendingBlock.transitions.push(transition)
        await this.db.put(
          RollupAggregator.getTransitionKey(
            this.pendingBlock.transitions.length
          ),
          hexStrToBuf(abiEncodeTransition(transition))
        )
      }
    }

    await this.db.put(
      RollupAggregator.LAST_TRANSITION_KEY,
      Buffer.from(this.pendingBlock.transitions.length.toString(10))
    )

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
      transition['createdAccountPubkey'] = update.receiverState.pubkey
    }
    return transition
  }

  /**
   * Submits a block to the main chain, creating a new pending block for future
   * transitions.
   */
  private async submitBlock(): Promise<void> {
    return this.lock.acquire(RollupAggregator.lockKey, async () => {
      if (
        this.pendingBlock.transitions.length <
        this.blockSubmissionTransitionCount
      ) {
        const millisSinceLastSubmission: number =
          new Date().getTime() - this.lastBlockSubmission.getTime()
        if (millisSinceLastSubmission < this.blockSubmissionIntervalMillis) {
          this.setBlockSubmissionTimeout(
            this.blockSubmissionIntervalMillis - millisSinceLastSubmission
          )
          return
        } else if (this.pendingBlock.transitions.length === 0) {
          this.setBlockSubmissionTimeout(this.blockSubmissionIntervalMillis)
          return
        }
      }

      const toSubmit = this.pendingBlock

      await this.rollupBlockSubmitter.submitBlock(toSubmit)
      this.pendingBlock = {
        blockNumber: toSubmit.blockNumber + 1,
        transitions: [],
      }

      await this.db.put(RollupAggregator.LAST_TRANSITION_KEY, Buffer.from('0'))
      await this.db.put(
        RollupAggregator.PENDING_BLOCK_KEY,
        Buffer.from(this.pendingBlock.blockNumber.toString(10))
      )

      this.lastBlockSubmission = new Date()

      this.setBlockSubmissionTimeout()
    })
  }

  private setBlockSubmissionTimeout(timeoutMillis?: number): void {
    setTimeout(async () => {
      await this.submitBlock()
    }, timeoutMillis || this.blockSubmissionIntervalMillis)
  }

  public static getTransitionKey(transIndex: number): Buffer {
    return Buffer.from(`TRANS_${transIndex}`)
  }

  /**
   * Generates two transactions which together send the user some UNI
   * & some PIGI.
   *
   * @param recipient The address to receive the faucet tokens
   * @param amount The amount to receive
   * @returns The signed faucet transactions
   */
  private async generateFaucetTxs(
    recipient: Address,
    amount: number
  ): Promise<SignedTransaction[]> {
    const address: string = await this.signatureProvider.getAddress()
    const txOne: RollupTransaction = generateTransferTx(
      address,
      recipient,
      UNI_TOKEN_TYPE,
      amount
    )
    const txTwo: RollupTransaction = generateTransferTx(
      address,
      recipient,
      PIGI_TOKEN_TYPE,
      amount
    )

    return [
      {
        signature: await this.signatureProvider.sign(
          abiEncodeTransaction(txOne)
        ),
        transaction: txOne,
      },
      {
        signature: await this.signatureProvider.sign(
          abiEncodeTransaction(txTwo)
        ),
        transaction: txTwo,
      },
    ]
  }

  /**
   * Increments the total transaction count that this aggregator has processed,
   * saving periodically
   */
  private async incrementTxCount(): Promise<void> {
    if (
      ++this.transactionCount % RollupAggregator.TX_COUNT_STORAGE_THRESHOLD ===
      0
    ) {
      try {
        await this.db.put(
          RollupAggregator.TRANSACTION_COUNT_KEY,
          Buffer.from(this.transactionCount.toString(10))
        )
      } catch (e) {
        logError(log, 'Error saving transaction count!', e)
      }
    }
  }

  /***********
   * GETTERS *
   ***********/

  public getPendingBlockNumber(): number {
    return this.pendingBlock.blockNumber
  }

  public getNextTransitionIndex(): number {
    return this.pendingBlock.transitions.length
  }
}
