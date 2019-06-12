import BigNum = require('bn.js')

import {PluginManager, PredicatePlugin, StateDB, StateManager} from "../../interfaces/client";
import {
  HistoryProof,
  Range,
  StateQuery,
  StateQueryResult,
  StateUpdate,
  Transaction, VerifiedStateUpdate
} from "../../interfaces/common/utils";

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

  public async executeTransaction(transaction: Transaction): Promise<{ stateUpdate: StateUpdate; validRanges: Range[] }> {
    const result = {
      stateUpdate: undefined,
      validRanges: []
    }

    if ( ! this.validateTransaction(transaction)) {
      return result
    }

    // TODO: Check that transaction.block < SyncManager.getLastSyncedBlock()

    const {start, end}: Range = transaction.stateUpdate.id
    const verifiedUpdates: VerifiedStateUpdate[] = await this.stateDB.getVerifiedStateUpdates(start, end)

    verifiedUpdates.forEach((verifiedUpdate: VerifiedStateUpdate) => {
      if ( ! this.validateVerifiedStateUpdate(verifiedUpdate)) {
        // log here?
        return
      }

      const {start: verifiedStart, end: verifiedEnd}: Range = verifiedUpdate.stateUpdate.id
      if (transaction.block !== verifiedUpdate.verifiedBlockNumber + 1 || verifiedEnd <= start || verifiedStart >= end) {
        // log here?
        return
      }

      const predicatePlugin: PredicatePlugin = await this.pluginManager.getPlugin(verifiedUpdate.stateUpdate.newState.predicate)
      if (!predicatePlugin) {
        // log here?
        return
      }

      const computedState: StateUpdate = await predicatePlugin.executeStateTransition(verifiedUpdate.stateUpdate, transaction)
      result.validRanges.push({
        start: BigNum.max(start, verifiedStart),
        end: BigNum.min(end, verifiedEnd)
      })


      if (result.stateUpdate === undefined) {
        result.stateUpdate = computedState
      } else if (result.stateUpdate !== computedState) {
        throw new Error(`State transition resulted in two different states: ${result.stateUpdate} and 
          ${computedState}. Latter differed from former range ${result.validRanges.pop()}.`)
      }
    })

    return result;
  }

  public ingestHistoryProof(historyProof: HistoryProof): Promise<void> {
    throw Error("DefaultStateManager.ingestHistoryProof is not implemented.")
  }

  public queryState(query: StateQuery): Promise<StateQueryResult[]> {
    throw Error("DefaultStateManager.queryState is not implemented.")
  }

  /**
   * Validates that a transaction has all of the necessary fields populated for it to be useful to StateManager.
   * Mostly just guards against NPEs.
   *
   * @param transaction the Transaction to inspect
   * @returns true if valid, false otherwise
   */
  private validateTransaction(transaction: Transaction): boolean {
    return !!transaction
      && !!transaction.stateUpdate
      && !!transaction.stateUpdate.id
      && !!transaction.stateUpdate.newState
      && !!transaction.block
      && !!transaction.witness
      && transaction.stateUpdate.id.start.gte(new BigNum(0))
      && transaction.stateUpdate.id.end.gt(transaction.stateUpdate.id.start)
  }

  /**
   * Validates that a VerifiedStateUpdate has all of the necessary fields populated for it to be useful to StateManager.
   * Mostly just guards against NPEs.
   *
   * @param verifiedUpdate the VerifiedStateUpdate to inspect
   * @returns true if valid, false otherwise
   */
  private validateVerifiedStateUpdate(verifiedUpdate: VerifiedStateUpdate) : boolean {
    return !!verifiedUpdate
      && !!verifiedUpdate.stateUpdate
      && !!verifiedUpdate.stateUpdate.newState
      && verifiedUpdate.start.lt(verifiedUpdate.end)
  }

}
