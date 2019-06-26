import BigNum = require('bn.js')

export interface SyncManager {
  /**
   * Gets the latest synced block number for the provided Plasma chain
   *
   * @returns the block number
   */
  getLastSyncedBlock(plasmaContract: string): Promise<BigNum>
}
