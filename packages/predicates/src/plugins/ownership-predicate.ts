import BigNum = require('bn.js')

import {
  getOverlappingRange,
  PredicatePlugin,
  StateUpdate,
  Transaction,
} from '@pigi/core'

export class OwnershipPredicatePlugin implements PredicatePlugin {
  public async executeStateTransition(
    previousStateUpdate: StateUpdate,
    transaction: Transaction,
    witness: string
  ): Promise<StateUpdate> {
    // TODO: Actually check signature stuffs
    if (previousStateUpdate.stateObject.data.owner !== witness) {
      throw new Error(
        `Cannot transition from state [${JSON.stringify(
          previousStateUpdate
        )}] with transaction [${transaction}] because witness does not match previousStateUpdate owner.`
      )
    }

    if (
      previousStateUpdate.plasmaBlockNumber.gte(
        transaction.parameters.originBlock
      )
    ) {
      throw new Error(
        `Cannot transition from state [${JSON.stringify(
          previousStateUpdate
        )}] with transaction [${JSON.stringify(
          transaction
        )}] because block number [${transaction.parameters.originBlock.toNumber()}] is not greater than previous state block number.`
      )
    }

    const range = getOverlappingRange(
      previousStateUpdate.range,
      transaction.range
    )
    if (range === undefined) {
      throw new Error(
        `Cannot transition from state [${JSON.stringify(
          previousStateUpdate
        )}] with transaction [${transaction}] because ranges do not overlap.`
      )
    }

    return {
      range,
      stateObject: transaction.parameters.newState,
      depositAddress: transaction.depositAddress,
      plasmaBlockNumber: transaction.parameters.originBlock,
    }
  }
}
