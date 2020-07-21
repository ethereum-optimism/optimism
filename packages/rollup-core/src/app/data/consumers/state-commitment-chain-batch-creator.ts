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
    private readonly minBatchSize: number = 1,
    private readonly maxbatchSize: number = 100,
    periodMilliseconds = 10_000
  ) {
    super(periodMilliseconds)
  }

  /**
   * @inheritDoc
   */
  public async runTask(): Promise<void> {
    try {
      const isL1Batch: boolean = await this.dataService.wasNextStateCommitmentChainBatchToBuildAppendedOnL1()
      if (isL1Batch) {
        log.debug(
          `Attempting to build state root batch to match appended L1 state roots...`
        )
        const batchNumber: number = await this.dataService.tryBuildStateCommitmentChainBatchToMatchAppendedL1Batch()
        if (batchNumber !== undefined && batchNumber >= 0) {
          log.debug(
            `State root batch matching appended L1 batch number ${batchNumber} was built!`
          )
        }
      } else {
        log.debug(`Attempting to build L2-only state root batch...`)
        const l2OnlyBatchBuilt: number = await this.dataService.tryBuildL2OnlyStateCommitmentChainBatch(
          this.minBatchSize,
          this.maxbatchSize
        )
        if (l2OnlyBatchBuilt !== undefined && l2OnlyBatchBuilt >= 0) {
          log.debug(
            `L2-only state root batch with number ${l2OnlyBatchBuilt} was built!`
          )
        }
      }

      log.debug(`No L2 State Root batch built... sad.`)
    } catch (e) {
      logError(
        log,
        `Error running StateCommitmentChainBatchCreator! Continuing...`,
        e
      )
    }
  }
}
