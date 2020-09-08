/* External Imports */
import {
  add0x,
  getLogger,
  getSignedTransaction,
  isTxSubmitted,
  keccak256,
  logError,
  ScheduledTask,
} from '@eth-optimism/core-utils'
import { Contract, Wallet } from 'ethers'

/* Internal Imports */
import {
  BatchSubmissionStatus,
  L2DataService,
  StateCommitmentBatchSubmission,
} from '../../../types/data'
import { TransactionReceipt, TransactionResponse } from 'ethers/providers'
import { UnexpectedBatchStatus } from '../../../types'

const log = getLogger('state-commitment-chain-batch-submitter')

/**
 * Polls the DB for L2 batches ready to send to L1 and submits them.
 */
export class StateCommitmentChainBatchSubmitter extends ScheduledTask {
  constructor(
    private readonly dataService: L2DataService,
    private readonly stateCommitmentChain: Contract,
    private readonly submitterWallet: Wallet,
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
    let stateBatch: StateCommitmentBatchSubmission
    try {
      stateBatch = await this.dataService.getNextStateCommitmentBatchToSubmit()
    } catch (e) {
      logError(
        log,
        `Error fetching state root batch for L1 submission! Continuing...`,
        e
      )
      return false
    }

    if (!stateBatch) {
      log.debug(`No state batches ready for submission.`)
      return false
    }

    if (
      stateBatch.status !== BatchSubmissionStatus.QUEUED &&
      stateBatch.status !== BatchSubmissionStatus.SUBMITTING
    ) {
      const msg = `Received state commitment batch to finalize in ${
        stateBatch.status
      } instead of ${BatchSubmissionStatus.QUEUED} or ${
        BatchSubmissionStatus.SUBMITTING
      }. Batch Submission: ${JSON.stringify(stateBatch)}.`
      log.error(msg)
      throw new UnexpectedBatchStatus(msg)
    }

    if (await this.shouldSubmitBatch(stateBatch)) {
      try {
        const txHash: string = await this.buildAndSendRollupBatchTransaction(
          stateBatch
        )
        if (!txHash) {
          return false
        }
        stateBatch.submissionTxHash = txHash
      } catch (e) {
        logError(
          log,
          `Error submitting state root batch number ${stateBatch.batchNumber}.`,
          e
        )
        return false
      }
    }

    try {
      return this.waitForProofThatTransactionSucceeded(
        stateBatch.submissionTxHash,
        stateBatch
      )
    } catch (e) {
      logError(
        log,
        `Error waiting for state batch ${stateBatch.batchNumber} with hash ${stateBatch.submissionTxHash} to succeed!`,
        e
      )
      return false
    }
  }

  protected async shouldSubmitBatch(batchSubmission): Promise<boolean> {
    return (
      batchSubmission.status === BatchSubmissionStatus.QUEUED ||
      !(await isTxSubmitted(
        this.stateCommitmentChain.provider,
        batchSubmission.submissionTxHash
      ))
    )
  }

  protected async getSignedRollupBatchTx(
    stateRoots: string[],
    startIndex: number
  ): Promise<string> {
    return getSignedTransaction(
      this.stateCommitmentChain,
      'appendStateBatch',
      [stateRoots, startIndex],
      this.submitterWallet
    )
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

      log.debug(
        `Appending state root batch number: ${stateRootBatch.batchNumber} with ${stateRoots.length} state roots at index ${stateRootBatch.startIndex}.`
      )

      const signedTx: string = await this.getSignedRollupBatchTx(
        stateRoots,
        stateRootBatch.startIndex
      )

      txHash = keccak256(signedTx, true)
      await this.dataService.markStateRootBatchSubmittingToL1(
        stateRootBatch.batchNumber,
        txHash
      )

      log.debug(
        `Marked tx ${txHash} for state batch ${stateRootBatch.batchNumber} as submitting.`
      )

      const txRes: TransactionResponse = await this.stateCommitmentChain.provider.sendTransaction(
        signedTx
      )

      log.debug(
        `Tx batch ${stateRootBatch.batchNumber} was sent to the state commitment chain! Tx Hash: ${txRes.hash}`
      )

      if (txHash !== txRes.hash) {
        log.warn(
          `Received tx hash not the same as calculated hash! Received: ${txRes.hash}, calculated: ${txHash}`
        )
        txHash = txRes.hash
      }
    } catch (e) {
      logError(
        log,
        `Error submitting State Root batch ${stateRootBatch.batchNumber} to state commitment chain!`,
        e
      )
      return undefined
    }

    return txHash
  }

  /**
   * Waits for a confirm to indicate that the transaction did not fail.
   *
   * @param txHash The tx hash to wait for.
   * @param stateRootBatch The rollup batch in question.
   * @returns true if the tx was successful and false otherwise.
   */
  private async waitForProofThatTransactionSucceeded(
    txHash: string,
    stateRootBatch: StateCommitmentBatchSubmission
  ): Promise<boolean> {
    try {
      const receipt: TransactionReceipt = await this.stateCommitmentChain.provider.waitForTransaction(
        txHash,
        1
      )
      if (!receipt.status) {
        log.error(
          `Error submitting State Root batch # ${stateRootBatch.batchNumber} to L1!. Batch: ${stateRootBatch}`
        )
        return false
      }
    } catch (e) {
      logError(
        log,
        `Error submitting State Root batch # ${stateRootBatch.batchNumber} to L1!. Batch: ${stateRootBatch}`,
        e
      )
      return false
    }

    try {
      log.debug(
        `Marking State Root batch ${stateRootBatch.batchNumber} submitted`
      )
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
      return false
    }
    return true
  }
}
