import { BlockBatches } from './types'

/**
 * Defines the event handler interface for handling L1 Block Batches.
 */
export interface BlockBatchListener {
  handleBlockBatches(transactionBatch: BlockBatches): Promise<void>
}
