import BigNum = require('bn.js')

import { StateUpdate } from '../serialization'

export interface BlockDB {
  getNextBlockNumber(): Promise<BigNum>
  addPendingStateUpdate(stateUpdate: StateUpdate): Promise<void>
  getPendingStateUpdates(): Promise<StateUpdate[]>
  getMerkleRoot(blockNumber: BigNum): Promise<Buffer>
  finalizeNextBlock(): Promise<void>
}
