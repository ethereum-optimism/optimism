/* External Imports */
import { getLogger, logError, ScheduledTask } from '@eth-optimism/core-utils'

/* Internal Imports */
import { DataService, GethSubmissionRecord } from '../../../types/data'

const log = getLogger('l2-batch-creator')

/**
 * Polls the DB to create an Optimistic Canonical Chain batch of L2 Transactions, when one is ready.
 */
export class CanonicalChainBatchCreator extends ScheduledTask {
  constructor(
    private readonly dataService: DataService,
    periodMilliseconds = 10_000
  ) {
    super(periodMilliseconds)
  }

  /**
   * @inheritDoc
   *
   * Creates L2 batches from L2 Transactions in the DB, either when:
   *    1. Unsubmitted & unverified transactions in the L2 tx DB match the oldest unverified L1 batch in size
   *    2. Unsubmitted & unverified transactions in the L2 tx DB have multiple timestamps (multiple batches exist)
   *
   */
  public async runTask(): Promise<void> {
    try {
      const l2OnlyBatchBuilt: number = await this.dataService.tryBuildCanonicalChainBatchNotPresentOnL1()
      if (l2OnlyBatchBuilt !== undefined && l2OnlyBatchBuilt >= 0) {
        log.debug(
          `L2-only Canonical Chain Tx batch with number ${l2OnlyBatchBuilt} was built!`
        )
        return
      }

      log.debug(`No Canonical Chain Tx batches built... sad.`)
    } catch (e) {
      logError(
        log,
        `Error running CanonicalChainBatchCreator! Continuing...`,
        e
      )
    }
  }
}
