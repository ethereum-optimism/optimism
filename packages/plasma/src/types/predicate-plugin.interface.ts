import { StateObject, StateUpdate, Transaction } from './state.interface'

export interface PredicatePlugin {
  /**
   * Executes a transaction according the Predicate logic and returns the resulting StateUpdate
   *
   * @param previousStateUpdate the previous StateUpdate upon which the provided Transaction acts
   * @param transaction the Transaction to execute
   * @param witness the signature data for the transaction
   * @returns the resulting StateUpdate
   */
  executeStateTransition(
    previousStateUpdate: StateUpdate,
    transaction: Transaction,
    witness: any
  ): Promise<StateObject>

  // TODO: Add other methods when used
}
