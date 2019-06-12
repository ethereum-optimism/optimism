import {StateDB, StateManager} from "../../interfaces/client";
import {
  HistoryProof,
  Range,
  StateQuery,
  StateQueryResult,
  StateUpdate,
  Transaction
} from "../../interfaces/common/utils";

/**
 * StateManager that validates transactions and wraps and modifies StateDB as necessary.
 *
 * See: http://spec.plasma.group/en/latest/src/05-client-architecture/state-manager.html for more details.
 */
export class DefaultStateManager implements StateManager {

  private stateDB: StateDB

  public constructor(stateDB: StateDB) {
    this.stateDB = stateDB
  }

  executeTransaction(transaction: Transaction): Promise<{ stateUpdate: StateUpdate; validRanges: Range[] }> {
    return undefined;
  }

  ingestHistoryProof(historyProof: HistoryProof): Promise<void> {
    throw Error("DefaultStateManager.ingestHistoryProof is not implemented.")
  }

  queryState(query: StateQuery): Promise<StateQueryResult[]> {
    throw Error("DefaultStateManager.queryState is not implemented.")
  }

}
