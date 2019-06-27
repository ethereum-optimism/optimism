import BigNum = require('bn.js')
import { StateUpdate, Transaction } from '../../types'

export interface PredicatePlugin {
  /**
   * Executes a transaction according the Predicate logic and returns the resulting StateUpdate
   *
   * @param previousStateUpdate the previous StateUpdate upon which the provided Transaction acts
   * @param transaction the Transaction to execute
   * @param inBlock the Block in which the Transaction is being proposed
   * @param witness the signature data for the transaction
   * @returns the resulting StateUpdate
   */
  executeStateTransition(
    previousStateUpdate: StateUpdate,
    transaction: Transaction,
    inBlock: BigNum,
    witness: string
  ): Promise<StateUpdate>

  // TODO: Add other methods when used
}
