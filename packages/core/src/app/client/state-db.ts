import {StateDB} from "../../interfaces/client";
import {VerifiedStateUpdate} from "../../interfaces/common/utils";

/**
 * StateDB used to store the state for different ranges.
 *
 * See: http://spec.plasma.group/en/latest/src/05-client-architecture/state-db.html for more details.
 */
export class DefaultStateDB implements StateDB {
  getVerifiedStateUpdates(start: number, end: number): Promise<VerifiedStateUpdate[]> {
    throw Error("DefaultStateDB.getVerifiedStateUpdates is not implemented.")
  }

  putVerifiedStateUpdate(verifiedStateUpdate: VerifiedStateUpdate): Promise<void> {
    throw Error("DefaultStateDB.putVerifiedStateUpdate is not implemented.")
  }

}
