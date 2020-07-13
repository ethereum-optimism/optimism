import { BlockBatches } from './types'

export interface L2NodeService {
  /**
   * Sends the provided BlockBatches to the configured L2 node.
   *
   * @param blockBatches The block batches to send to L2
   */
  sendBlockBatches(blockBatches: BlockBatches): Promise<void>
}
