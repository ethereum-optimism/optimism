import BigNum = require('bn.js')

import {
  getOverlappingRange,
  PredicatePlugin,
  StateUpdate,
  SyncManager,
  Transaction,
} from '@pigi/core'

export class OwnershipPredicatePlugin implements PredicatePlugin {
  private readonly syncManager: SyncManager

  public constructor(syncManager: SyncManager) {
    this.syncManager = syncManager
  }

  public async executeStateTransition(
    previousStateUpdate: StateUpdate,
    transaction: Transaction,
    witness: string
  ): Promise<StateUpdate> {
    await this.validateStateTransition(
      previousStateUpdate,
      transaction,
      witness
    )

    const range = getOverlappingRange(
      previousStateUpdate.range,
      transaction.range
    )
    if (range === undefined) {
      throw new Error(
        `Cannot transition from state [${JSON.stringify(
          previousStateUpdate
        )}] with transaction [${JSON.stringify(
          transaction
        )}] because ranges do not overlap.`
      )
    }

    return {
      range,
      stateObject: transaction.parameters.newState,
      depositAddress: transaction.depositAddress,
      plasmaBlockNumber: transaction.parameters.originBlock,
    }
  }

  /**
   * Validates that the provided previous StateUpdate, Transaction, and witness are valid.
   *
   * @param previousStateUpdate the previous StateUpdate upon which the provided Transaction acts
   * @param transaction the Transaction to execute
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

    const lastSyncedBlock: BigNum = await this.syncManager.getLastSyncedBlock(
      previousStateUpdate.stateObject.predicateAddress
    )

    if (lastSyncedBlock.gte(transaction.parameters.targetBlock)) {
      throw new Error(
        `Cannot transition from state [${JSON.stringify(
          previousStateUpdate
        )}] with transaction [${JSON.stringify(
          transaction
        )}] because current block [${lastSyncedBlock.toNumber()}] is >= transaction target block.`
      )
    }
  }
}
