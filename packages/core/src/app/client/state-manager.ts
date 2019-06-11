import {StateManager} from "../../interfaces/client";
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
  executeTransaction(transaction: Transaction): Promise<{ stateUpdate: StateUpdate; validRanges: Range[] }> {
    return undefined;
  }

  ingestHistoryProof(historyProof: HistoryProof): Promise<void> {
    return undefined;
  }

  queryState(query: StateQuery): Promise<StateQueryResult[]> {
    return undefined;
  }

}
