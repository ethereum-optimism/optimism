import BigNum = require('bn.js')
import { MerkleSumTree } from '@pigi/utils'

import { Transaction, TransactionProof, ChainDB } from '../../../interfaces'
import { StateProcessor } from '../../common'

/**
 * Checks whether a transaction is a deposit.
 * @param transaction Transaction to check.
 * @returns `true` if it's a deposit, `false` otherwise.
 */
const isDeposit = (transaction: Transaction): boolean => {
  return transaction.witness !== undefined
}

/**
 * StateManager implementation for PG's Plasma Cashflow variant.
 */
export class PGStateManager {
  constructor(private db: ChainDB) {}

  /**
   * Applies a single transaction to the local state.
   * @param transaction Transaction to apply.
   * @param transactionProof Additional proof information.
   */
  public async applyTransaction(
    transaction: Transaction,
    transactionProof: TransactionProof
  ): Promise<void> {
    /*
    if (!(await this.checkTransactionProof(transaction, transactionProof))) {
      throw new Error('Invalid transaction proof')
    }

    const processor = await this.loadState()
    const newTransactions: Transaction[] = []
    for (const proofElement of transactionProof) {
      if (!processor.hasStateUpdate(proofElement.stateUpdate)) {
        newTransactions.push(proofElement)
        processor.addStateUpdate(proofElement.stateUpdate)
      }
    }

    // TODO: Lock each range before saving head state.
    await this.saveState(processor)
    // TODO: Write new transactions to the historical state.
    */
  }

  /**
   * Checks a transaction proof. Uses local state
   * and public information (e.g. plasma blocks).
   * @param transaction Transaction to check.
   * @param transactionProof Proof to check.
   * @returns `true` if the proof is valid, `false` otherwise.
   */
  public async checkTransactionProof(
    transaction: Transaction,
    transactionProof: TransactionProof
  ): Promise<boolean> {
    return

    // TODO: Figure out whether transaction should be added to TransactionProof.
    // TODO: Figure out whether this function should return the latest state.

    /*
    const processor = await this.loadState()

    const deposits: Transaction[] = []
    const transactions: Transaction[] = []
    for (const proofElement of transactionProof) {
      if (isDeposit(proofElement)) {
        deposits.push(proofElement)
      } else {
        transactions.push(proofElement)
      }
    }

    for (const deposit of deposits) {
      if (!(await this.isValidDeposit(deposit))) {
        return false
      }

      processor.addStateUpdate(deposit.stateUpdate)
    }

    for (const tx of transactions) {
      try {
        const { implicitStart, implicitEnd } = await this.checkInclusionProof(
          transaction
        )
        // TODO: Add implicit ends somehow.
      } catch {
        return false
      }

      const oldStates = processor.getOldStates(tx.stateUpdate)
      for (const oldState of oldStates) {
        // TODO: Check that the state transition is valid.
      }

      processor.applyStateUpdate(tx.stateUpdate)
    }

    if (!(await processor.hasStateUpdate(transaction.stateUpdate))) {
      return false
    }

    return true
    */
  }

  /**
   * Checks whether a deposit is valid.
   * @param deposit Deposit to check.
   * @returns `true` if the deposit is valid, `false` otherwise.
   */
  private async isValidDeposit(deposit: Transaction): Promise<boolean> {
    // TODO: Implement this check.
    return
  }

  private async checkInclusionProof(
    transaction: Transaction
  ): Promise<{ implicitStart: BigNum; implicitEnd: BigNum }> {
    /*
    // TODO: Figure out where to get the block root.
    const root = null
    if (root === null) {
      throw new Error(
        `Received transaction for non-existent block #${transaction.block}`
      )
    }

    // TODO: Figure out where to put the inclusion proof.
    const tree = new MerkleSumTree()
    return tree.verify(
      {
        end: transaction.stateUpdate.end,
        data: transaction.newState.encoded,
      },
      0,
      transaction.inclusionProof,
      root + 'ffffffffffffffffffffffffffffffff'
    )
    */
    return
  }

  /**
   * @returns the current head state as a Processor.
   */
  private async loadState(): Promise<StateProcessor> {
    /**
    const state = await this.db.get(Buffer.from('state'))
    return new StateProcessor(JSON.parse(state.toString('utf8')))
    */
    return
  }

  /**
   * Saves the current head state.
   * @param processor Processor to save from.
   */
  private async saveState(processor: StateProcessor): Promise<void> {
    /**
    const state = Buffer.from(JSON.stringify(processor.state), 'utf8')
    await this.db.put(Buffer.from('state'), state)
    */
  }
}
