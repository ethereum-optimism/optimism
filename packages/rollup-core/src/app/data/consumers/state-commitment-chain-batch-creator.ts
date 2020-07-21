/* External Imports */
import { getLogger, logError, ScheduledTask } from '@eth-optimism/core-utils'

/* Internal Imports */
import { DataService, GethSubmissionRecord } from '../../../types/data'

const log = getLogger('l2-batch-creator')

/**
 * Polls the DB to create an State Commitment Chain batch of L2 Transaction State Roots, when one is ready.
 */
export class StateCommitmentChainBatchCreator extends ScheduledTask {
  constructor(
    private readonly dataService: DataService,
    periodMilliseconds = 10_000
  ) {
    super(periodMilliseconds)
  }

  /**
   * @inheritDoc
   */
  public async runTask(): Promise<void> {
    try {
      const l2OnlyBatchBuilt: number = await this.dataService.tryBuildStateCommitmentChainBatchToMatchAppendedL1Batch()
      if (l2OnlyBatchBuilt !== undefined && l2OnlyBatchBuilt >= 0) {
        log.debug(`L2-only OCC tx batch with number ${l2OnlyBatchBuilt} was built!`)
        return
      }
      log.debug(`No L2 OCC tx batches built... sad.`)
    } catch (e) {
      logError(log, `Error running OptimisticCanonicalChainBatchCreator! Continuing...`, e)
    }
  }
}
