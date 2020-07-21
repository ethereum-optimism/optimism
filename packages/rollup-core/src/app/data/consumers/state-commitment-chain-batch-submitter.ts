/* External Imports */
import {
  getLogger,
  logError,
  ScheduledTask,
} from '@eth-optimism/core-utils'
import { Contract } from 'ethers'

/* Internal Imports */
import {
  BatchSubmissionStatus,
  L2DataService, StateCommitmentBatchSubmission,
} from '../../../types/data'
import {TransactionReceipt, TransactionResponse} from 'ethers/providers'

const log = getLogger('state-commitment-chain-batch-submitter')

/**
 * Polls the DB for L2 batches ready to send to L1 and submits them.
 */
export class StateCommitmentChainBatchSubmitter extends ScheduledTask {
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
   *
   */
  public async runTask(): Promise<void> {
    let stateBatch: StateCommitmentBatchSubmission
    try {
      stateBatch = await this.dataService.getNextStateCommitmentBatchToSubmit()
    } catch (e) {
      logError(log, `Error fetching state root batch for L1 submission! Continuing...`, e)
      return
    }

    if (!stateBatch || !stateBatch.stateRoots || !stateBatch.stateRoots.length) {
      log.debug(`No state root batches found for L1 submission.`)
      return
    }

    let rootBatchTxHash: string = stateBatch.submissionTxHash
    switch (stateBatch.status) {
      case BatchSubmissionStatus.QUEUED:
        rootBatchTxHash = await this.buildAndSendRollupBatchTransaction(stateBatch)
        if (!rootBatchTxHash) {
          return
        }
      // Fallthrough on purpose -- this is a workflow
      case BatchSubmissionStatus.SENT:
        await this.waitForStateRootBatchConfirms(rootBatchTxHash, stateBatch.batchNumber)
      // Fallthrough on purpose -- this is a workflow
      case BatchSubmissionStatus.FINALIZED:
        break
      default:
        log.error(
          `Received L1 Batch submission in unexpected tx batch state: ${stateBatch.status}!`
        )
        return
    }
  }

  /**
   * Builds and sends a Rollup State Root Batch transaction to L1, returning its tx hash.
   *
   * @param stateRootBatch The state root batch to send to L1.
   * @returns The L1 tx hash.
   */
  private async buildAndSendRollupBatchTransaction(
    stateRootBatch: StateCommitmentBatchSubmission
  ): Promise<string> {
    let txHash: string
    try {
      const stateRoots: string[] = stateRootBatch.stateRoots

      const txRes: TransactionResponse = await this.stateCommitmentChain.appendStateBatch(stateRoots)
      log.debug(
        `State Root batch ${stateRootBatch.batchNumber} appended with at least one confirmation! Tx Hash: ${txRes.hash}`
      )
      txHash = txRes.hash
    } catch (e) {
      logError(
        log,
        `Error submitting State Root batch ${stateRootBatch.batchNumber} to state commitment chain!`,
        e
      )
      return undefined
    }

    try {
      log.debug(`Marking State Root batch ${stateRootBatch.batchNumber} submitted`)
      await this.dataService.markStateRootBatchSubmittedToL1(
        stateRootBatch.batchNumber,
        txHash
      )
    } catch (e) {
      logError(
        log,
        `Error marking State Root batch ${stateRootBatch.batchNumber} as submitted!`,
        e
      )
      // TODO: Should we return here? Don't want to resubmit, so I think we should update the DB
    }
    return txHash
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
