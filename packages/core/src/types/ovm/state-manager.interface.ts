import BigNum = require('bn.js')

import {
  HistoryProof,
  Range,
  StateQuery,
  StateQueryResult,
  StateUpdate,
  Transaction,
  TransactionResult,
} from '../../types'

export interface StateManager {
  /**
   * Executes the provided Transaction against the current verified State.
   *
   * @param transaction the transaction to execute
   * @param inBlock the block in which this transaction is expected to be executed
   * @param witness the signature data for the transaction in question
   * @returns the resulting StateUpdate and a list of Ranges over which the StateUpdate has been validated
   */
  executeTransaction(
    transaction: Transaction,
    inBlock: BigNum,
    witness: string
  ): Promise<TransactionResult>

  /**
   * Validates the provided HistoryProof.
   *
   * @param historyProof the proof to validate
   * @returns an empty Promise that will resolve successfully if valid and error if not
   */
  ingestHistoryProof(historyProof: HistoryProof): Promise<void>

  /**
   * Executes the provided StateQuery and returns any results.
   *
   * @param query the StateQuery in question
   * @returns The list of StateQueryResults that the query produces.
   */
  queryState(query: StateQuery): Promise<StateQueryResult[]>
}
