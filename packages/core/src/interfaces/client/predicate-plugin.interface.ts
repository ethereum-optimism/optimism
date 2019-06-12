import {StateUpdate, Transaction} from "../common/utils";

export interface PredicatePlugin {
  /**
   * Executes a transaction according the Predicate logic and returns the resulting StateUpdate
   *
   * @param previousStateUpdate the previous StateUpdate upon which the provided Transaction acts
   * @param transaction the Transaction to execute
   * @returns the resulting StateUpdate
   */
  executeStateTransition(previousStateUpdate: StateUpdate, transaction: Transaction): Promise<StateUpdate>

  // TODO: Add other methods when used
}