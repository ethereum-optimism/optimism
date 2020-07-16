/* External Imports */
import {
  getLogger,
  logError,
  ScheduledTask,
} from '@eth-optimism/core-utils/build'

/* Internal Imports */
import { L1DataService } from '../../../types/data'
import { BlockBatches, L2NodeService } from '../../../types'

const log = getLogger('l2-batch-submitter')

/**
 * Polls the database for new Rollup Transactions that were submitted to L1 that
 * have not yet been processed by L2 and submits them one-by-one to L2.
 */
export class QueuedGethSubmitter extends ScheduledTask {
  constructor(
    private readonly l1DataService: L1DataService,
    private readonly l2NodeService: L2NodeService,
    periodMilliseconds: number = 10_000
  ) {
    super(periodMilliseconds)
  }

  /**
   * @inheritDoc
   */
  public async runTask(): Promise<void> {
    let blockBatches: BlockBatches
    try {
      blockBatches = await this.l1DataService.getNextQueuedGethSubmission()
    } catch (e) {
      logError(log, `Error fetching next batch for L2 submission!`, e)
      return
    }

    if (!blockBatches) {
      log.debug(`No batches ready for submission to L2.`)
      return
    }

    try {
      await this.l2NodeService.sendBlockBatches(blockBatches)
    } catch (e) {
      logError(
        log,
        `Error sending batch to BlockBatchSubmitter! Block Batches: ${JSON.stringify(
          blockBatches
        )}`,
        e
      )
      return
    }

    try {
      await this.l1DataService.markQueuedGethSubmissionSubmittedToGeth(
        blockBatches.batchNumber
      )
    } catch (e) {
      logError(
        log,
        `Error marking L1 Batch as Submitted to L2. L1 Batch Number: ${blockBatches.batchNumber}`,
        e
      )
      return
    }
  }
}
