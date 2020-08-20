/* External Imports */
import { getLogger, logError, ScheduledTask } from '@eth-optimism/core-utils'
import { Contract } from 'ethers'

/* Internal Imports */
import {
  BatchSubmission,
  BatchSubmissionStatus,
  L2DataService,
  StateCommitmentBatchSubmission,
} from '../../../types/data'
import { TransactionReceipt, TransactionResponse } from 'ethers/providers'

const log = getLogger('state-commitment-chain-batch-finalizer')

/**
 * Polls the DB for L2 batches ready to send to L1 and submits them.
 */
export class StateCommitmentChainBatchFinalizer extends ScheduledTask {
  constructor(
    private readonly dataService: L2DataService,
    private readonly stateCommitmentChain: Contract,
    private readonly confirmationsUntilFinal: number = 1,
    periodMilliseconds = 10_000
  ) {
    super(periodMilliseconds)
  }

  /**
   * @inheritDoc
   *
   * Submits L2 batches from L2 Transactions in the DB whenever there is a batch that is ready.
   */
  public async runTask(): Promise<boolean> {
    let batchToFinalize: BatchSubmission
    try {
      batchToFinalize = await this.dataService.getNextStateCommitmentBatchToFinalize()
    } catch (e) {
      logError(
        log,
        `Error fetching state root batch for L1 finalization! Continuing...`,
        e
      )
      return false
    }

    if (
      !batchToFinalize ||
      batchToFinalize.batchNumber === null ||
      batchToFinalize.batchNumber === undefined ||
      !batchToFinalize.submissionTxHash
    ) {
      log.debug(`No tx batches found to finalize.`)
      return false
    }

    if (batchToFinalize.status !== BatchSubmissionStatus.SENT) {
      const msg = `Received state commitment batch to finalize in ${
        batchToFinalize.status
      } instead of ${
        BatchSubmissionStatus.SENT
      }. Batch Submission: ${JSON.stringify(batchToFinalize)}.`
      log.error(msg)
      throw msg
    }

    await this.waitForStateRootBatchConfirms(
      batchToFinalize.submissionTxHash,
      batchToFinalize.batchNumber
    )
    return true
  }

  /**
   * Waits for the configured number of confirms for the provided rollup tx transaction hash and
   * marks the tx as
   *
   * @param txHash The tx hash to wait for.
   * @param batchNumber The rollup batch number in question.
   */
  private async waitForStateRootBatchConfirms(
    txHash: string,
    batchNumber: number
  ): Promise<void> {
    if (this.confirmationsUntilFinal > 1) {
      try {
        log.debug(
          `Waiting for ${this.confirmationsUntilFinal} confirmations before treating state root batch ${batchNumber} submission as final.`
        )
        const receipt: TransactionReceipt = await this.stateCommitmentChain.provider.waitForTransaction(
          txHash,
          this.confirmationsUntilFinal
        )
        log.debug(
          `State root batch submission finalized for batch ${batchNumber}!`
        )
      } catch (e) {
        logError(
          log,
          `Error waiting for necessary block confirmations until final!`,
          e
        )
        // TODO: Should we return here? Don't want to resubmit, so I think we should update the DB
      }
    }

    try {
      log.debug(`Marking state root batch ${batchNumber} confirmed!`)
      await this.dataService.markStateRootBatchFinalOnL1(batchNumber, txHash)
      log.debug(`State root batch ${batchNumber} marked confirmed!`)
    } catch (e) {
      logError(log, `Error marking batch ${batchNumber} as confirmed!`, e)
    }
  }
}
