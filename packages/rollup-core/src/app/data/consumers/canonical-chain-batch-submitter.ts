/* External Imports */
import {
  getLogger,
  logError,
  numberToHexString,
  remove0x,
  ScheduledTask,
} from '@eth-optimism/core-utils'
import { Contract } from 'ethers'

/* Internal Imports */
import {
  TransactionBatchSubmission,
  BatchSubmissionStatus,
  L2DataService,
} from '../../../types/data'
import { TransactionReceipt, TransactionResponse } from 'ethers/providers'

const log = getLogger('canonical-chain-batch-submitter')

/**
 * Polls the DB for L2 batches ready to send to L1 and submits them.
 */
export class CanonicalChainBatchSubmitter extends ScheduledTask {
  constructor(
    private readonly dataService: L2DataService,
    private readonly canonicalTransactionChain: Contract,
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
    let batchSubmission: TransactionBatchSubmission
    try {
      batchSubmission = await this.dataService.getNextCanonicalChainTransactionBatchToSubmit()
    } catch (e) {
      logError(log, `Error fetching batch for L1 submission! Continuing...`, e)
      return
    }

    if (
      !batchSubmission ||
      !batchSubmission.transactions ||
      !batchSubmission.transactions.length
    ) {
      log.debug(`No batches found for L1 submission.`)
      return
    }

    let txBatchTxHash: string = batchSubmission.submissionTxHash
    switch (batchSubmission.status) {
      case BatchSubmissionStatus.QUEUED:
        txBatchTxHash = await this.buildAndSendRollupBatchTransaction(
          batchSubmission
        )
        if (!txBatchTxHash) {
          return
        }
      // Fallthrough on purpose -- this is a workflow
      case BatchSubmissionStatus.SENT:
        await this.waitForTxBatchConfirms(
          txBatchTxHash,
          batchSubmission.batchNumber
        )
      // Fallthrough on purpose -- this is a workflow
      case BatchSubmissionStatus.FINALIZED:
        break
      default:
        log.error(
          `Received L1 Batch submission in unexpected tx batch state: ${batchSubmission.status}!`
        )
        return
    }
  }

  /**
   * Builds and sends a Rollup Batch transaction to L1, returning its tx hash.
   *
   * @param l2Batch The L2 batch to send to L1.
   * @returns The L1 tx hash.
   */
  private async buildAndSendRollupBatchTransaction(
    l2Batch: TransactionBatchSubmission
  ): Promise<string> {
    let txHash: string
    try {
      const txsCalldata: string[] = this.getTransactionBatchCalldata(l2Batch)

      const timestamp = l2Batch.transactions[0].timestamp
      log.debug(
        `Submitting tx batch ${
          l2Batch.batchNumber
        } to canonical chain. Batch: ${JSON.stringify(
          l2Batch
        )}, txs bytes: ${JSON.stringify(txsCalldata)}, timestamp: ${timestamp}`
      )
      const txRes: TransactionResponse = await this.canonicalTransactionChain.appendSequencerBatch(
        txsCalldata,
        timestamp
      )
      log.debug(
        `Tx batch ${l2Batch.batchNumber} appended with at least one confirmation! Tx Hash: ${txRes.hash}`
      )
      txHash = txRes.hash
    } catch (e) {
      logError(
        log,
        `Error submitting tx batch ${l2Batch.batchNumber} to canonical chain!`,
        e
      )
      return undefined
    }

    try {
      log.debug(`Marking tx batch ${l2Batch.batchNumber} submitted`)
      await this.dataService.markTransactionBatchSubmittedToL1(
        l2Batch.batchNumber,
        txHash
      )
    } catch (e) {
      logError(
        log,
        `Error marking tx batch ${l2Batch.batchNumber} as submitted!`,
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
  private async waitForTxBatchConfirms(
    txHash: string,
    batchNumber: number
  ): Promise<void> {
    if (this.confirmationsUntilFinal > 1) {
      try {
        log.debug(
          `Waiting for ${this.confirmationsUntilFinal} confirmations before treating tx batch ${batchNumber} submission as final.`
        )
        const receipt: TransactionReceipt = await this.canonicalTransactionChain.provider.waitForTransaction(
          txHash,
          this.confirmationsUntilFinal
        )
        log.debug(`Batch submission finalized for tx batch ${batchNumber}!`)
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
      log.debug(`Marking tx batch ${batchNumber} confirmed!`)
      await this.dataService.markTransactionBatchConfirmedOnL1(
        batchNumber,
        txHash
      )
      log.debug(`Tx batch ${batchNumber} marked confirmed!`)
    } catch (e) {
      logError(log, `Error marking tx batch ${batchNumber} as confirmed!`, e)
    }
  }

  /**
   * Gets the calldata bytes for a transaction batch to be submitted by the sequencer.
   * Rollup Transaction Format:
   *    target: 20-byte address    0-20
   *    nonce: 32-byte uint        20-52
   *    gasLimit: 32-byte uint     52-84
   *    signature: 65-byte bytes   84-149
   *    calldata: bytes            149-end
   *
   * @param batch The batch to turn into ABI-encoded calldata bytes.
   * @returns The ABI-encoded bytes[] of the Rollup Transactions in the format listed above.
   */
  private getTransactionBatchCalldata(
    batch: TransactionBatchSubmission
  ): string[] {
    const txs: string[] = []
    for (const tx of batch.transactions) {
      const nonce: string = remove0x(numberToHexString(tx.nonce, 32))
      const gasLimit: string = tx.gasLimit
        ? tx.gasLimit.toString('hex', 64)
        : '00'.repeat(32)
      const signature: string = remove0x(tx.signature)
      const calldata: string = remove0x(tx.calldata)
      txs.push(`${tx.to}${nonce}${gasLimit}${signature}${calldata}`)
    }

    return txs
  }
}
