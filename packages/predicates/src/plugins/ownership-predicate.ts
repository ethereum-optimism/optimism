import BigNum = require('bn.js')

import {
  PredicatePlugin,
  isRangeSubset,
  StateUpdate,
  Transaction,
  StateObject,
} from '@pigi/core'

export class OwnershipPredicatePlugin implements PredicatePlugin {
  public async executeStateTransition(
    previousStateUpdate: StateUpdate,
    transaction: Transaction,
    inBlock: BigNum,
    witness: string
  ): Promise<StateUpdate> {
    await this.validateStateTransition(
      previousStateUpdate,
      transaction,
      witness
    )

    return {
      range: transaction.range,
      stateObject: transaction.body.newState,
      depositAddress: transaction.depositAddress,
      plasmaBlockNumber: inBlock,
    }
  }

  /**
   * Gets the owner of the provided StateObject, if one is present.
   *
   * @returns the owner if one is present, undefined otherwise
   */
  public getOwner(stateObject: StateObject): string | undefined {
    try {
      return stateObject.data.owner
    } catch (e) {
      return undefined
    }
  }

  /**
   * Validates that the provided previous StateUpdate, Transaction, and witness are valid.
   *
   * @param previousStateUpdate the previous StateUpdate upon which the provided Transaction acts
   * @param transaction the Transaction to execute
   * @param inBlock the Block in which this Transaction is being proposed
   * @param witness the signature data for the transaction
   *
   * @throws if the state transition is not valid or input is not of expected format
   */
  private async validateStateTransition(
    previousStateUpdate: StateUpdate,
    transaction: Transaction,
    witness: string
  ): Promise<void> {
    // TODO: Actually check signature stuffs
    if (previousStateUpdate.stateObject.data.owner !== witness) {
      throw new Error(
        `Cannot transition from state [${JSON.stringify(
          previousStateUpdate
        )}] with transaction [${JSON.stringify(
          transaction
        )}] because witness does not match previousStateUpdate owner.`
      )
    }

    if (
      previousStateUpdate.plasmaBlockNumber.gte(transaction.body.originBlock)
    ) {
      throw new Error(
        `Cannot transition from state [${JSON.stringify(
          previousStateUpdate
        )}] with transaction [${JSON.stringify(
          transaction
        )}] because block number [${transaction.body.originBlock.toNumber()}] is not greater than previous state block number.`
      )
    }

    if (!isRangeSubset(transaction.range, previousStateUpdate.range)) {
      throw new Error(
        `Cannot transition from state [${JSON.stringify(
          previousStateUpdate
        )}] with transaction [${JSON.stringify(
          transaction
        )}] because transaction range is not a subset of previous state update range.`
      )
    }
  }
}
