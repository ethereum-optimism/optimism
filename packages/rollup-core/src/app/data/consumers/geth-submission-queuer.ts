/* External Imports */
import { getLogger, logError, ScheduledTask } from '@eth-optimism/core-utils'

/* Internal Imports */
import {GethSubmissionRecord, L1DataService} from '../../../types/data'

const log = getLogger('l2-batch-creator')

/**
 * Polls the DB to create a batch of L1 Transactions, when one is ready.
 */
export class GethSubmissionQueuer extends ScheduledTask {
  constructor(
    private readonly dataService: L1DataService,
    periodMilliseconds = 10_000
  ) {
    super(periodMilliseconds)
  }

  /**
   * @inheritDoc
   *
   * Creates L1 batches from L1 Transactions in the DB.
   *
   */
  public async runTask(): Promise<void> {
    // TODO: Implement this
    // try {
    //   const l1BatchRecord: L1BatchRecord = await this.dataService.get()
    //   if (!l1BatchRecord) {
    //     const l2OnlyBatchBuilt: number = await this.dataService.tryBuildL2OnlyBatch()
    //     if (l2OnlyBatchBuilt !== undefined && l2OnlyBatchBuilt >= 0) {
    //       log.debug(`L2-only batch with number ${l2OnlyBatchBuilt} was built!`)
    //     }
    //     return
    //   }
    //
    //   const batchBuilt: number = await this.dataService.tryBuildL2BatchToMatchL1(
    //     l1BatchRecord.batchNumber,
    //     l1BatchRecord.batchSize
    //   )
    //   if (batchBuilt !== undefined && batchBuilt >= 0) {
    //     log.debug(
    //       `L2 batch to match L1 batch of size ${l1BatchRecord} was built. Batch number: ${batchBuilt}.`
    //     )
    //     return
    //   }
    //
    //   log.debug(`No L2 batches built... sad.`)
    // } catch (e) {
    //   logError(log, `Error running L2BatchCreator! Continuing...`, e)
    // }
  }
}
