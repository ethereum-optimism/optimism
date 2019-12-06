/* External Imports */
import { DB } from '@pigi/core-db'
import {
  BigNumber,
  getLogger,
  logError,
  ONE,
  runInDomain,
} from '@pigi/core-utils'

import * as AsyncLock from 'async-lock'
import * as domain from 'domain'

/* Internal Imports */

import {
  SignedTransaction,
  RollupStateMachine,
  State,
  TransactionResult,
  TransactionStorage,
} from '../types'
import { NON_EXISTENT_SLOT_INDEX } from './utils'

const log = getLogger('rollup-state-machine')

/**
 * A Rollup State Machine, facilitating state transitions and transactions for Optimistic Rollup.
 */
export class DefaultRollupStateMachine implements RollupStateMachine {
  private static readonly NEXT_TRANSACTION_NUMBER_KEY: Buffer = Buffer.from(
    'tx_num'
  )
  private static readonly lockKey: string = 'lock'
  private static readonly emptyState: State = {
    slotIndex: NON_EXISTENT_SLOT_INDEX,
    balances: {},
  }

  private readonly lock: AsyncLock

  private nextTransactionNumber: BigNumber

  /**
   * Constructs a DefaultRollupStateMachine and initializes it.
   *
   * @param db The DB to use for the RollupStateMachine
   * @returns The new RollupStateMachine
   */
  public static async create(db: DB): Promise<DefaultRollupStateMachine> {
    const rsm: DefaultRollupStateMachine = new DefaultRollupStateMachine(db)
    await rsm.init()

    return rsm
  }

  private constructor(private readonly db: DB) {
    this.lock = new AsyncLock({
      domainReentrant: true,
    })
    this.nextTransactionNumber = ONE
  }

  /**
   * Initializes the RollupStateMachine, loading necessary member variables from storage
   */
  private async init(): Promise<void> {
    const nextTxNumBuffer: Buffer = await this.db.get(
      DefaultRollupStateMachine.NEXT_TRANSACTION_NUMBER_KEY
    )

    if (!nextTxNumBuffer) {
      log.info(`No stored transaction number found. Starting fresh.`)
      return
    }

    this.nextTransactionNumber = new BigNumber(nextTxNumBuffer)
    log.info(
      `Initialized State Machine from DB with next transaction number ${this.nextTransactionNumber.toString()}`
    )
  }

  public async getState(slotIndex: string): Promise<State> {
    // TODO: How are we going to return state?
    return {}
  }

  public async applyTransaction(
    signedTransaction: SignedTransaction,
    d?: domain.Domain
  ): Promise<TransactionResult> {
    return runInDomain(d, async () => {
      log.debug(
        `Acquiring lock to apply transaction: ${JSON.stringify(
          signedTransaction
        )}`
      )
      return this.lock.acquire(DefaultRollupStateMachine.lockKey, async () => {
        log.debug(
          `Lock acquired. Applying transaction: ${JSON.stringify(
            signedTransaction
          )}`
        )
        const result = { signedTransaction }
        // const transaction: RollupTransaction = signedTransaction.transaction
        const modifiedStorage: TransactionStorage[] = []

        // TODO: Run transactions here

        const transactionResult: TransactionResult = {
          transactionNumber: this.nextTransactionNumber,
          signedTransaction,
          modifiedStorage,
        }

        this.nextTransactionNumber = this.nextTransactionNumber.add(ONE)

        try {
          await this.db.put(
            DefaultRollupStateMachine.NEXT_TRANSACTION_NUMBER_KEY,
            this.nextTransactionNumber.toBuffer()
          )
        } catch (e) {
          logError(
            log,
            `Transaction succeeded. Failed to update next transaction number to [${this.nextTransactionNumber.toString()}].`,
            e
          )
          throw e
        }

        return transactionResult
      })
    })
  }

  public async getTransactionResultsSince(
    transactionNumber: BigNumber
  ): Promise<TransactionResult[]> {
    // TODO: Call EVM to get this list.
    return []
  }
}
