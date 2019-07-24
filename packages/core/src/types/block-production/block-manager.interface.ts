import { StateUpdate } from '../serialization'
import { BigNumber } from '../../app/utils'

/**
 * Block Manager wrapping Block CRUD operations.
 */
export interface BlockManager {
  getNextBlockNumber(): Promise<BigNumber>
  addPendingStateUpdate(stateUpdate: StateUpdate): Promise<void>
  getPendingStateUpdates(): Promise<StateUpdate[]>
  submitNextBlock(): Promise<void>
}
