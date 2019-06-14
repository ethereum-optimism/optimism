import BigNum = require('bn.js')

import { isValidTransaction, isValidVerifiedStateUpdate } from '../common/utils'

import {
  HistoryProof,
  PluginManager,
  PredicatePlugin,
  Range,
  StateDB,
  StateManager,
  StateQuery,
  StateQueryResult,
  StateUpdate,
  Transaction,
  VerifiedStateUpdate,
} from '../../interfaces'
import { getOverlappingRange, rangesIntersect } from '../common/utils/range'

/**
 * StateManager that validates transactions and wraps and modifies StateDB as necessary.
 *
 * See: http://spec.plasma.group/en/latest/src/05-client-architecture/state-manager.html for more details.
 */
export class DefaultStateManager implements StateManager {
  private stateDB: StateDB
  private pluginManager: PluginManager

  public constructor(stateDB: StateDB, pluginManager: PluginManager) {
    this.stateDB = stateDB
    this.pluginManager = pluginManager
  }

  public async executeTransaction(
    transaction: Transaction,
    inBlock: number
  ): Promise<{ stateUpdate: StateUpdate; validRanges: Range[] }> {
    const result = {
      stateUpdate: undefined,
      validRanges: [],
    }

    if (!isValidTransaction(transaction)) {
      throw new Error(
        `Cannot execute invalid Transaction: ${JSON.stringify(transaction)}`
      )
    }

    // Get verified updates for range
    const { start, end }: Range = transaction.range
    const verifiedUpdates: VerifiedStateUpdate[] = await this.stateDB.getVerifiedStateUpdates(
      start,
      end
    )

    // Iterate over the verified updates, transition their state, and add their ranges to the return object
    for (const verifiedUpdate of verifiedUpdates) {
      if (!isValidVerifiedStateUpdate(verifiedUpdate)) {
        throw new Error(
          `Cannot process transaction for invalid VerifiedStateUpdate: ${JSON.stringify(
            verifiedUpdate
          )}`
        )
      }

      // If the ranges don't overlap, eagerly exit
      if (!rangesIntersect(verifiedUpdate.range, transaction.range)) {
        throw Error(`VerifiedStateUpdate for range [${start}, ${end}) is outside of range: 
        ${JSON.stringify(
          verifiedUpdate.range
        )}. VerifiedStateUpdate: ${JSON.stringify(verifiedUpdate)}.`)
      }

      if (verifiedUpdate.verifiedBlockNumber + 1 !== inBlock) {
        throw Error(`VerifiedStateUpdate has block ${
          verifiedUpdate.verifiedBlockNumber
        } and ${inBlock - 1} was expected. 
          VerifiedStateUpdate: ${JSON.stringify(verifiedUpdate)}`)
      }

      const predicatePlugin: PredicatePlugin = await this.pluginManager.getPlugin(
        verifiedUpdate.stateUpdate.stateObject.predicate
      )

      const computedState: StateUpdate = await predicatePlugin.executeStateTransition(
        verifiedUpdate.stateUpdate,
        verifiedUpdate.verifiedBlockNumber,
        transaction
      )

      if (
        computedState.plasmaBlockNumber !==
        verifiedUpdate.verifiedBlockNumber + 1
      ) {
        throw new Error(`Transaction resulted in StateUpdate with unexpected block number.
          Expected: ${verifiedUpdate.verifiedBlockNumber + 1}, found: ${
          computedState.plasmaBlockNumber
        }.
          VerifiedStateUpdate transitioned: ${JSON.stringify(verifiedUpdate)}`)
      }

      result.validRanges.push(
        getOverlappingRange(transaction.range, verifiedUpdate.range)
      )

      if (result.stateUpdate === undefined) {
        result.stateUpdate = computedState
      } else if (result.stateUpdate !== computedState) {
        throw new Error(`State transition resulted in two different states: ${JSON.stringify(
          result.stateUpdate
        )} and 
          ${computedState}. Latter differed from former at range ${JSON.stringify(
          result.validRanges.pop()
        )}.`)
      }
    }

    return result
  }

  public ingestHistoryProof(historyProof: HistoryProof): Promise<void> {
    throw Error('DefaultStateManager.ingestHistoryProof is not implemented.')
  }

  public queryState(query: StateQuery): Promise<StateQueryResult[]> {
    throw Error('DefaultStateManager.queryState is not implemented.')
  }
}
