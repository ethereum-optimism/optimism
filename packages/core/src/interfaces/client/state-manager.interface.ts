import {HistoryProof, Range, StateQuery, StateQueryResult, StateUpdate, Transaction} from "../common/utils";

export interface StateManager {
  /**
   * Executes the provided Transaction against the current verified State.
   *
   * @param transaction the transaction to execute
   * @returns the resulting StateUpdate and a list of Ranges over which the StateUpdate has been validated
   */
  executeTransaction(transaction: Transaction) : Promise<{stateUpdate: StateUpdate, validRanges: Range[]}>

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