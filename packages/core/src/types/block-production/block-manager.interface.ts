import BigNum = require('bn.js')

import { StateUpdate } from '../serialization'

/**
 * Block Manager wrapping Block CRUD operations.
 */
export interface BlockManager {
  getNextBlockNumber(): Promise<BigNum>
  addPendingStateUpdate(stateUpdate: StateUpdate): Promise<void>
  getPendingStateUpdates(): Promise<StateUpdate[]>
  submitNextBlock(): Promise<void>
}
