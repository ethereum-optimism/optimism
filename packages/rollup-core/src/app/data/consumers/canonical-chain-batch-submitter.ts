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
  BatchSubmission,
} from '../../../types/data'
import { TransactionReceipt, TransactionResponse } from 'ethers/providers'
import {
  UnexpectedBatchStatus,
  FutureRollupBatchNumberError,
  FutureRollupBatchTimestampError,
  RollupBatchBlockNumberTooOldError,
  RollupBatchTimestampTooOldError,
  RollupBatchSafetyQueueBlockNumberError,
  RollupBatchSafetyQueueBlockTimestampError,
  RollupBatchL1ToL2QueueBlockNumberError,
  RollupBatchL1ToL2QueueBlockTimestampError,
  RollupBatchOvmBlockNumberError,
  RollupBatchOvmTimestampError,
} from '../../../types'

const log = getLogger('canonical-chain-batch-submitter')

/**
 * Polls the DB for L2 batches ready to send to L1 and submits them.
 */
export class CanonicalChainBatchSubmitter extends ScheduledTask {
  constructor(
    private readonly dataService: L2DataService,
    private readonly canonicalTransactionChain: Contract,
    private readonly l1ToL2QueueContract: Contract,
    private readonly safetyQueueContract: Contract,
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

    const batchBlockNumber = await this.getBatchSubmissionBlockNumber()

    await this.validateBatchSubmission(batchSubmission, batchBlockNumber)

    const txHash: string = await this.buildAndSendRollupBatchTransaction(
      batchSubmission,
      batchBlockNumber
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
   * @param batchBlockNumber The BlockNumber for this batch
   * @returns The L1 tx hash.
   */
  private async buildAndSendRollupBatchTransaction(
    l2Batch: TransactionBatchSubmission,
    batchBlockNumber: number
  ): Promise<string> {
    let txHash: string
    try {
      const txsCalldata: string[] = this.getTransactionBatchCalldata(l2Batch)

      // TODO: update this to work with geth-persisted timestamp/block number that updates based on L1 actions
      const timestamp = l2Batch.transactions[0].timestamp

      log.debug(
        `Submitting tx batch ${l2Batch.batchNumber} with ${l2Batch.transactions.length} transactions to canonical chain. Timestamp: ${timestamp}`
      )
      const txRes: TransactionResponse = await this.canonicalTransactionChain.appendSequencerBatch(
        txsCalldata,
        timestamp,
        batchBlockNumber
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

  protected async getBatchSubmissionBlockNumber(): Promise<number> {
    // TODO: This is bad. Make this not suck.
    return (await this.getL1BlockNumber()) - 10
  }

  private async validateBatchSubmission(
    batchSubmission: TransactionBatchSubmission,
    batchBlockNumber: number
  ): Promise<void> {
    let forceInclusionSeconds: number
    let forceInclusionBlocks: number
    let blockNumber: number
    let safetyQueueTimestampSeconds: number
    let safetyQueueBlockNumber: number
    let l1ToL2QueueTimestampSeconds: number
    let l1ToL2QueueBlockNumber: number
    let lastOvmTimestampSeconds: number
    let lastOvmBlockNumber: number
    ;[
      forceInclusionSeconds,
      forceInclusionBlocks,
      blockNumber,
      safetyQueueTimestampSeconds,
      safetyQueueBlockNumber,
      l1ToL2QueueTimestampSeconds,
      l1ToL2QueueBlockNumber,
      lastOvmTimestampSeconds,
      lastOvmBlockNumber,
    ] = await Promise.all([
      this.getForceInclusionPeriodSeconds(),
      this.getForceInclusionPeriodBlocks(),
      this.getL1BlockNumber(),
      this.getMaxSafetyQueueTimestampSeconds(),
      this.getMaxSafetyQueueBlockNumber(),
      this.getMaxL1ToL2QueueTimestampSeconds(),
      this.getMaxL1ToL2QueueBlockNumber(),
      this.getLastOvmTimestampSeconds(),
      this.getLastOvmBlockNumber(),
    ])

    const nowSeconds = Math.round(new Date().getTime() / 1000)
    const batchTimestamp = batchSubmission.transactions[0].timestamp

    if (batchBlockNumber > blockNumber) {
      throw new FutureRollupBatchNumberError(
        `Batch block number cannot be in the future. Batch block number is ${batchBlockNumber} and block number: ${blockNumber}.`
      )
    }

    if (batchTimestamp > nowSeconds) {
      throw new FutureRollupBatchTimestampError(
        `Batch timestamp cannot be in the future. Batch timestamp is ${batchTimestamp} and current timestamp is ${nowSeconds}.`
      )
    }

    if (batchTimestamp + forceInclusionSeconds <= nowSeconds) {
      throw new RollupBatchTimestampTooOldError(
        `Batch is too old. Batch timestamp is ${batchTimestamp}, force inclusion period is ${forceInclusionSeconds}, now is ${nowSeconds}`
      )
    }

    if (batchBlockNumber + forceInclusionBlocks <= blockNumber) {
      throw new RollupBatchBlockNumberTooOldError(
        `Batch is too old. Batch Block # is ${batchTimestamp}, force inclusion blocks is ${forceInclusionBlocks}, L1 block number is ${blockNumber}`
      )
    }

    if (
      safetyQueueTimestampSeconds !== 0 &&
      batchTimestamp > safetyQueueTimestampSeconds
    ) {
      throw new RollupBatchSafetyQueueBlockTimestampError(
        `Safety Queue tx must come first. Safety queue timestamp is ${safetyQueueTimestampSeconds}, batch timestamp is ${batchTimestamp}`
      )
    }

    if (
      safetyQueueBlockNumber !== 0 &&
      batchBlockNumber > safetyQueueBlockNumber
    ) {
      throw new RollupBatchSafetyQueueBlockNumberError(
        `Safety Queue tx must come first. Safety queue blockNumber is ${safetyQueueBlockNumber}, batch blockNumber is ${batchBlockNumber}`
      )
    }

    if (
      l1ToL2QueueTimestampSeconds !== 0 &&
      batchTimestamp > l1ToL2QueueTimestampSeconds
    ) {
      throw new RollupBatchL1ToL2QueueBlockTimestampError(
        `L1 to L2 Queue tx must come first. L1 to L2 Queue timestamp is ${l1ToL2QueueTimestampSeconds}, batch timestamp is ${batchTimestamp}`
      )
    }

    if (
      l1ToL2QueueBlockNumber !== 0 &&
      batchBlockNumber > l1ToL2QueueBlockNumber
    ) {
      throw new RollupBatchL1ToL2QueueBlockNumberError(
        `L1 to L2 Queue tx must come first. L1 to L2 Queue blockNumber is ${l1ToL2QueueBlockNumber}, batch blockNumber is ${batchBlockNumber}`
      )
    }

    if (batchTimestamp < lastOvmTimestampSeconds) {
      throw new RollupBatchOvmTimestampError(
        `Batch timestamp must be > last OVM Timestamp. Batch timestamp is ${batchTimestamp}, last OVM timestamp is ${lastOvmTimestampSeconds}`
      )
    }

    if (batchBlockNumber < lastOvmBlockNumber) {
      throw new RollupBatchOvmBlockNumberError(
        `Batch block number must be > last OVM block number. Batch block number is ${batchBlockNumber}, last OVM block number is ${lastOvmBlockNumber}`
      )
    }
  }

  private forceInclusionPeriodSeconds: number
  private async getForceInclusionPeriodSeconds(): Promise<number> {
    if (this.forceInclusionPeriodSeconds === undefined) {
      this.forceInclusionPeriodSeconds = await this.canonicalTransactionChain.forceInclusionPeriodSeconds()
    }
    return this.forceInclusionPeriodSeconds
  }

  private forceInclusionPeriodBlocks: number
  private async getForceInclusionPeriodBlocks(): Promise<number> {
    if (this.forceInclusionPeriodBlocks === undefined) {
      this.forceInclusionPeriodBlocks = await this.canonicalTransactionChain.forceInclusionPeriodBlocks()
    }
    return this.forceInclusionPeriodBlocks
  }

  private async getL1BlockNumber(): Promise<number> {
    return this.canonicalTransactionChain.provider.getBlockNumber()
  }

  private async getMaxSafetyQueueTimestampSeconds(): Promise<number> {
    return this.safetyQueueContract.peekTimestamp()
  }

  private async getMaxSafetyQueueBlockNumber(): Promise<number> {
    return this.safetyQueueContract.peekBlockNumber()
  }

  private async getMaxL1ToL2QueueTimestampSeconds(): Promise<number> {
    return this.l1ToL2QueueContract.peekTimestamp()
  }

  private async getMaxL1ToL2QueueBlockNumber(): Promise<number> {
    return this.l1ToL2QueueContract.peekBlockNumber()
  }

  private async getLastOvmTimestampSeconds(): Promise<number> {
    return this.canonicalTransactionChain.lastOVMTimestamp()
  }

  private async getLastOvmBlockNumber(): Promise<number> {
    return this.canonicalTransactionChain.lastOVMBlockNumber()
  }
}
