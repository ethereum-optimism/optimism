import BigNum = require('bn.js')

import { StateDB, VerifiedStateUpdate } from '../../types'

/**
 * StateDB used to store the state for different ranges.
 *
 * See: http://spec.plasma.group/en/latest/src/05-client-architecture/state-db.html for more details.
 */
export class DefaultStateDB implements StateDB {
  public async getVerifiedStateUpdates(
    start: BigNum,
    end: BigNum
  ): Promise<VerifiedStateUpdate[]> {
    throw Error('DefaultStateDB.getVerifiedStateUpdates is not implemented.')
  }

  public async putVerifiedStateUpdate(
    verifiedStateUpdate: VerifiedStateUpdate
  ): Promise<void> {
    throw Error('DefaultStateDB.putVerifiedStateUpdate is not implemented.')
  }
}
