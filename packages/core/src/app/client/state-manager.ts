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
    transaction: Transaction
  ): Promise<{ stateUpdate: StateUpdate; validRanges: Range[] }> {
    const result = {
      stateUpdate: undefined,
      validRanges: [],
    }

    if (!isValidTransaction(transaction)) {
      // log here?
      return result
    }

    // Get Verified updates for range
    const { start, end }: Range = transaction.stateUpdate.id
    const verifiedUpdates: VerifiedStateUpdate[] = await this.stateDB.getVerifiedStateUpdates(
      start,
      end
    )

    for (const verifiedUpdate of verifiedUpdates) {
      if (!isValidVerifiedStateUpdate(verifiedUpdate)) {
        // log here?
        continue
      }

      const {
        start: verifiedStart,
        end: verifiedEnd,
      }: Range = verifiedUpdate.stateUpdate.id
      if (
        transaction.block !== verifiedUpdate.verifiedBlockNumber + 1 ||
        verifiedEnd.lte(start) ||
        verifiedStart.gte(end)
      ) {
        // log here?
        continue
      }

      const predicatePlugin: PredicatePlugin = await this.pluginManager.getPlugin(
        verifiedUpdate.stateUpdate.newState.predicate
      )

      const computedState: StateUpdate = await predicatePlugin.executeStateTransition(
        verifiedUpdate.stateUpdate,
        transaction
      )
      result.validRanges.push({
        start: BigNum.max(start, verifiedStart),
        end: BigNum.min(end, verifiedEnd),
      })

      if (result.stateUpdate === undefined) {
        result.stateUpdate = computedState
      } else if (result.stateUpdate !== computedState) {
        throw new Error(`State transition resulted in two different states: ${
          result.stateUpdate
        } and 
          ${computedState}. Latter differed from former at range ${result.validRanges.pop()}.`)
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
