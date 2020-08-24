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
  StateCommitmentBatchSubmission,
} from '../../../types/data'
import { TransactionReceipt, TransactionResponse } from 'ethers/providers'
import { UnexpectedBatchStatus } from '../../../types'

const log = getLogger('canonical-chain-batch-submitter')

/**
 * Polls the DB for L2 batches ready to send to L1 and submits them.
 */
export class CanonicalChainBatchSubmitter extends ScheduledTask {
  constructor(
    private readonly dataService: L2DataService,
    private readonly canonicalTransactionChain: Contract,
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
    let batchSubmission: TransactionBatchSubmission
    try {
      batchSubmission = await this.dataService.getNextCanonicalChainTransactionBatchToSubmit()
    } catch (e) {
      logError(
        log,
        `Error fetching tx batch for L1 submission! Continuing...`,
        e
      )
      return false
    }

    if (
      !batchSubmission ||
      !batchSubmission.transactions ||
      !batchSubmission.transactions.length
    ) {
      log.debug(`No tx batches found for L1 submission.`)
      return false
    }

    if (batchSubmission.status !== BatchSubmissionStatus.QUEUED) {
      const msg = `Received tx batch to send in ${
        batchSubmission.status
      } instead of ${
        BatchSubmissionStatus.QUEUED
      }. Batch Submission: ${JSON.stringify(batchSubmission)}.`
      log.error(msg)
      throw new UnexpectedBatchStatus(msg)
    }

    const txHash: string = await this.buildAndSendRollupBatchTransaction(
      batchSubmission
    )
    if (!txHash) {
      return false
    }
    return this.waitForProofThatTransactionSucceeded(txHash, batchSubmission)
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

      // TODO: update this to work with geth-persisted timestamp/block number that updates based on L1 actions
      const timestamp = l2Batch.transactions[0].timestamp
      const blocknumber =
        (await this.canonicalTransactionChain.provider.getBlockNumber()) - 10 // broken for any prod setting but works for now

      log.debug(
        `Submitting tx batch ${l2Batch.batchNumber} with ${l2Batch.transactions.length} transactions to canonical chain. Timestamp: ${timestamp}`
      )
      const txRes: TransactionResponse = await this.canonicalTransactionChain.appendSequencerBatch(
        txsCalldata,
        timestamp,
        blocknumber
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

    return txHash
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

  /**
   * Waits for a confirm to indicate that the transaction did not fail.
   *
   * @param txHash The tx hash to wait for.
   * @param txBatch The rollup batch in question.
   * @returns true if the tx was successful and false otherwise.
   */
  private async waitForProofThatTransactionSucceeded(
    txHash: string,
    txBatch: TransactionBatchSubmission
  ): Promise<boolean> {
    try {
      const receipt: TransactionReceipt = await this.canonicalTransactionChain.provider.waitForTransaction(
        txHash,
        1
      )
      if (!receipt.status) {
        log.error(
          `Error submitting tx batch # ${txBatch.batchNumber} to L1!. Batch: ${txBatch}`
        )
        return false
      }
    } catch (e) {
      logError(
        log,
        `Error submitting tx batch # ${txBatch.batchNumber} to L1!. Batch: ${txBatch}`,
        e
      )
      return false
    }

    try {
      log.debug(`Marking tx batch ${txBatch.batchNumber} submitted`)
      await this.dataService.markTransactionBatchSubmittedToL1(
        txBatch.batchNumber,
        txHash
      )
    } catch (e) {
      logError(
        log,
        `Error marking tx batch ${txBatch.batchNumber} as submitted!`,
        e
      )
      // TODO: Should we return here? Don't want to resubmit, so I think we should update the DB
      return false
    }
    return true
  }
}
