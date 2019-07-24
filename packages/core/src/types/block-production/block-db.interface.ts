import { StateUpdate } from '../serialization'
import { BigNumber } from '../../app/utils'

export interface BlockDB {
  getNextBlockNumber(): Promise<BigNumber>
  addPendingStateUpdate(stateUpdate: StateUpdate): Promise<void>
  getPendingStateUpdates(): Promise<StateUpdate[]>
  getMerkleRoot(blockNumber: BigNumber): Promise<Buffer>
  finalizeNextBlock(): Promise<void>
}
