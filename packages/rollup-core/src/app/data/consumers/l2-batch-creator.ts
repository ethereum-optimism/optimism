/* External Imports */
import { getLogger, logError, ScheduledTask } from '@eth-optimism/core-utils'

/* Internal Imports */
import { DataService, L1BatchRecord } from '../../../types/data'

const log = getLogger('l2-batch-creator')

/**
 * Polls the DB to create a batch of L2 Transactions, when one is ready.
 */
export class L2BatchCreator extends ScheduledTask {
  constructor(
    private readonly dataService: DataService,
    periodMilliseconds = 10_000
  ) {
    super(periodMilliseconds)
  }

  /**
   * Creates L2 batches from L2 Transactions in the DB, either when:
   *    1. Unsubmitted & unverified transactions in the L2 tx DB match the oldest unverified L1 batch in size
   *    2. Unsubmitted & unverified transactions in the L2 tx DB have multiple timestamps (multiple batches exist)
   *
   * @inheritDoc
   */
  public async runTask(): Promise<void> {
    try {
      const l1BatchRecord: L1BatchRecord = await this.dataService.getOldestUnverifiedL1TransactionBatch()
      if (!l1BatchRecord) {
        const l2OnlyBatchBuilt: number = await this.dataService.tryBuildL2OnlyBatch()
        if (l2OnlyBatchBuilt !== undefined && l2OnlyBatchBuilt >= 0) {
          log.debug(`L2-only batch with number ${l2OnlyBatchBuilt} was built!`)
        }
        return
      }

      const batchBuilt: number = await this.dataService.tryBuildL2BatchToMatchL1(
        l1BatchRecord.batchNumber,
        l1BatchRecord.batchSize
      )
      if (batchBuilt !== undefined && batchBuilt >= 0) {
        log.debug(
          `L2 batch to match L1 batch of size ${l1BatchRecord} was built. Batch number: ${batchBuilt}.`
        )
        return
      }

      log.debug(`No L2 batches built... sad.`)
    } catch (e) {
      logError(log, `Error running L2BatchCreator! Continuing...`, e)
    }
  }
}
