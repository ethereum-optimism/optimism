/* External Imports */
import { keccak256FromUtf8, sleep, TestUtils } from '@eth-optimism/core-utils'
import { TransactionReceipt, TransactionResponse } from 'ethers/providers'
import { Contract, Wallet } from 'ethers'

/* Internal Imports */
import {
  DefaultDataService,
  CanonicalChainBatchSubmitter,
} from '../../src/app/data'
import {
  TransactionBatchSubmission,
  BatchSubmissionStatus,
  L2DataService,
} from '../../src/types/data'
import {
  FutureRollupBatchNumberError,
  FutureRollupBatchTimestampError,
  RollupBatchBlockNumberTooOldError,
  RollupBatchL1ToL2QueueBlockNumberError,
  RollupBatchL1ToL2QueueBlockTimestampError,
  RollupBatchOvmBlockNumberError,
  RollupBatchOvmTimestampError,
  RollupBatchSafetyQueueBlockNumberError,
  RollupBatchSafetyQueueBlockTimestampError,
  RollupBatchTimestampTooOldError,
  UnexpectedBatchStatus,
} from '../../src/types'

interface BatchNumberHash {
  batchNumber: number
  txHash: string
}

class TestCanonicalChainBatchSubmitter extends CanonicalChainBatchSubmitter {
  public batchSubmissionBlockNumberOverride: number

  constructor(
    dataService: L2DataService,
    canonicalTransactionChain: Contract,
    l1ToL2QueueContract: Contract,
    safetyQueueContract: Contract,
    periodMilliseconds = 10_000
  ) {
    super(
      dataService,
      canonicalTransactionChain,
      l1ToL2QueueContract,
      safetyQueueContract,
      periodMilliseconds
    )
  }

  protected async getBatchSubmissionBlockNumber(): Promise<number> {
    if (this.batchSubmissionBlockNumberOverride) {
      return this.batchSubmissionBlockNumberOverride
    }
    return super.getBatchSubmissionBlockNumber()
  }
}

class MockDataService extends DefaultDataService {
  public readonly nextBatch: TransactionBatchSubmission[] = []
  public readonly txBatchesSubmitted: BatchNumberHash[] = []
  public readonly txBatchesFinalized: BatchNumberHash[] = []

  constructor() {
    super(undefined)
  }

  public async getNextCanonicalChainTransactionBatchToSubmit(): Promise<
    TransactionBatchSubmission
  > {
    return this.nextBatch.shift()
  }

  public async markTransactionBatchSubmittedToL1(
    batchNumber: number,
    l1TxHash: string
  ): Promise<void> {
    this.txBatchesSubmitted.push({ batchNumber, txHash: l1TxHash })
  }

  public async markTransactionBatchFinalOnL1(
    batchNumber: number,
    l1TxHash: string
  ): Promise<void> {
    this.txBatchesFinalized.push({ batchNumber, txHash: l1TxHash })
  }
}

class MockProvider {
  public txReceipts: Map<string, TransactionReceipt> = new Map<
    string,
    TransactionReceipt
  >()
  public blockNumberOverride: number

  public async waitForTransaction(
    hash: string,
    numConfirms: number
  ): Promise<TransactionReceipt> {
    while (!this.txReceipts.get(hash)) {
      await sleep(100)
    }
    return this.txReceipts.get(hash)
  }

  public async getBlockNumber(): Promise<number> {
    return this.blockNumberOverride || this.txReceipts.size
  }
}

class MockCanonicalTransactionChain {
  public responses: TransactionResponse[] = []
  public lastOvmTimestampSeconds: number = 0
  public lastOvmBlock: number = 0

  public forceInclusionSeconds: number = 0
  public forceInclusionBlocks: number = 0

  constructor(public readonly provider: MockProvider) {}

  public async appendSequencerBatch(
    calldata: string,
    timestamp: number,
    blockNumber: number,
    startIndex: number
  ): Promise<TransactionResponse> {
    const response: TransactionResponse = this.responses.shift()
    if (!response) {
      throw Error('no response')
    }
    return response
  }

  public async lastOVMTimestamp(): Promise<number> {
    return this.lastOvmTimestampSeconds
  }

  public async lastOVMBlockNumber(): Promise<number> {
    return this.lastOvmBlock
  }

  public async forceInclusionPeriodSeconds(): Promise<number> {
    return this.forceInclusionSeconds
  }

  public async forceInclusionPeriodBlocks(): Promise<number> {
    return this.forceInclusionBlocks
  }
}

class MockQueue {
  public timestamp: number
  public blockNumber: number

  public async peekBlockNumber(): Promise<number> {
    if (this.blockNumber === undefined) {
      throw Error(`Queue is empty`)
    }
    return this.blockNumber
  }

  public async peekTimestamp(): Promise<number> {
    if (this.timestamp === undefined) {
      throw Error(`Queue is empty`)
    }
    return this.timestamp
  }
}

describe('Canonical Chain Batch Submitter', () => {
  let batchSubmitter: TestCanonicalChainBatchSubmitter
  let dataService: MockDataService
  let canonicalProvider: MockProvider
  let canonicalTransactionChain: MockCanonicalTransactionChain
  let l1ToL2TransactionQueue: MockQueue
  let safetyQueue: MockQueue

  beforeEach(async () => {
    dataService = new MockDataService()
    canonicalProvider = new MockProvider()
    canonicalTransactionChain = new MockCanonicalTransactionChain(
      canonicalProvider
    )
    canonicalTransactionChain.forceInclusionSeconds = 100_000_000_000
    canonicalTransactionChain.forceInclusionBlocks = 100_000_000_000

    l1ToL2TransactionQueue = new MockQueue()
    safetyQueue = new MockQueue()
    batchSubmitter = new TestCanonicalChainBatchSubmitter(
      dataService,
      canonicalTransactionChain as any,
      l1ToL2TransactionQueue as any,
      safetyQueue as any
    )
    canonicalProvider.blockNumberOverride = 1
    batchSubmitter.batchSubmissionBlockNumberOverride = 1
  })

  it('should not do anything if there are no batches', async () => {
    const res = await batchSubmitter.runTask(true)

    res.should.equal(false, 'Incorrect result when there are no batches')

    dataService.txBatchesSubmitted.length.should.equal(
      0,
      'No tx batches should have been submitted!'
    )
    dataService.txBatchesFinalized.length.should.equal(
      0,
      'No tx batches should have been confirmed!'
    )
  })

  it('should throw if the next batch has an invalid status', async () => {
    dataService.nextBatch.push({
      submissionTxHash: undefined,
      status: 'derp' as any,
      batchNumber: 1,
      transactions: [
        {
          timestamp: 1,
          blockNumber: 2,
          transactionHash: keccak256FromUtf8('l2 tx hash'),
          transactionIndex: 0,
          to: Wallet.createRandom().address,
          from: Wallet.createRandom().address,
          nonce: 1,
          calldata: keccak256FromUtf8('some calldata'),
          stateRoot: keccak256FromUtf8('l2 state root'),
          signature: 'ab'.repeat(65),
        },
      ],
    })

    await TestUtils.assertThrowsAsync(async () => {
      await batchSubmitter.runTask(true)
    }, UnexpectedBatchStatus)

    dataService.txBatchesSubmitted.length.should.equal(
      0,
      'No tx batches should have been submitted!'
    )
    dataService.txBatchesFinalized.length.should.equal(
      0,
      'No tx batches should have been confirmed!'
    )
  })

  it('should send txs if there is a batch', async () => {
    const hash: string = keccak256FromUtf8('l1 tx hash')
    const batchNumber: number = 1
    dataService.nextBatch.push({
      submissionTxHash: undefined,
      status: BatchSubmissionStatus.QUEUED,
      batchNumber,
      transactions: [
        {
          timestamp: 1,
          blockNumber: 2,
          transactionHash: keccak256FromUtf8('l2 tx hash'),
          transactionIndex: 0,
          to: Wallet.createRandom().address,
          from: Wallet.createRandom().address,
          nonce: 1,
          calldata: keccak256FromUtf8('some calldata'),
          stateRoot: keccak256FromUtf8('l2 state root'),
          signature: 'ab'.repeat(65),
        },
      ],
    })

    canonicalTransactionChain.responses.push({ hash } as any)
    canonicalProvider.txReceipts.set(hash, { status: 1 } as any)

    const res: boolean = await batchSubmitter.runTask(true)
    res.should.equal(true, `Batch should have been submitted successfully.`)

    dataService.txBatchesSubmitted.length.should.equal(
      1,
      'No tx batches submitted!'
    )
    dataService.txBatchesSubmitted[0].txHash.should.equal(
      hash,
      'Incorrect tx hash submitted!'
    )
    dataService.txBatchesSubmitted[0].batchNumber.should.equal(
      batchNumber,
      'Incorrect tx batch number submitted!'
    )

    dataService.txBatchesFinalized.length.should.equal(
      0,
      'No tx batches should be confirmed!'
    )
  })

  it('should not mark txs as sent if batch submission tx fails', async () => {
    const hash: string = keccak256FromUtf8('l1 tx hash')
    const batchNumber: number = 1
    dataService.nextBatch.push({
      submissionTxHash: undefined,
      status: BatchSubmissionStatus.QUEUED,
      batchNumber,
      transactions: [
        {
          timestamp: 1,
          blockNumber: 2,
          transactionHash: keccak256FromUtf8('l2 tx hash'),
          transactionIndex: 0,
          to: Wallet.createRandom().address,
          from: Wallet.createRandom().address,
          nonce: 1,
          calldata: keccak256FromUtf8('some calldata'),
          stateRoot: keccak256FromUtf8('l2 state root'),
          signature: 'ab'.repeat(65),
        },
      ],
    })

    canonicalTransactionChain.responses.push({ hash } as any)
    canonicalProvider.txReceipts.set(hash, { status: 0 } as any)

    const res: boolean = await batchSubmitter.runTask(true)
    res.should.equal(false, `Batch tx should have errored out.`)

    dataService.txBatchesSubmitted.length.should.equal(
      0,
      'No tx batches submitted!'
    )

    dataService.txBatchesFinalized.length.should.equal(
      0,
      'No tx batches should be confirmed!'
    )
  })

  describe('Smart Contract Logic Pre-checks', () => {
    const setUpTask = (batchTimestamp?: number) => {
      const hash: string = keccak256FromUtf8('l1 tx hash')
      const batchNumber: number = 1
      dataService.nextBatch.push({
        submissionTxHash: undefined,
        status: BatchSubmissionStatus.QUEUED,
        batchNumber,
        transactions: [
          {
            timestamp: batchTimestamp || 1,
            blockNumber: 2,
            transactionHash: keccak256FromUtf8('l2 tx hash'),
            transactionIndex: 0,
            to: Wallet.createRandom().address,
            from: Wallet.createRandom().address,
            nonce: 1,
            calldata: keccak256FromUtf8('some calldata'),
            stateRoot: keccak256FromUtf8('l2 state root'),
            signature: 'ab'.repeat(65),
          },
        ],
      })

      canonicalTransactionChain.responses.push({ hash } as any)
      canonicalProvider.txReceipts.set(hash, { status: 1 } as any)
    }

    it('should throw if the batch block number is greater than the L1 block number', async () => {
      batchSubmitter.batchSubmissionBlockNumberOverride = 2
      canonicalProvider.blockNumberOverride = 1

      setUpTask()

      await TestUtils.assertThrowsAsync(async () => {
        await batchSubmitter.runTask(true)
      }, FutureRollupBatchNumberError)
    })

    it('should throw if the batch timestamp is greater than the L1 timestamp', async () => {
      const nowSeconds = Math.round(new Date().getTime() / 1000)

      setUpTask(nowSeconds + 20)

      await TestUtils.assertThrowsAsync(async () => {
        await batchSubmitter.runTask(true)
      }, FutureRollupBatchTimestampError)
    })

    it('should throw if not included within force inclusion period seconds', async () => {
      canonicalTransactionChain.forceInclusionSeconds = 1

      const nowSeconds = Math.round(new Date().getTime() / 1000)

      setUpTask(nowSeconds - 5)

      await TestUtils.assertThrowsAsync(async () => {
        await batchSubmitter.runTask(true)
      }, RollupBatchTimestampTooOldError)
    })

    it('should throw if not included within force inclusion period blocks', async () => {
      canonicalProvider.blockNumberOverride = 5
      canonicalTransactionChain.forceInclusionBlocks = 1
      batchSubmitter.batchSubmissionBlockNumberOverride = 4

      setUpTask()

      await TestUtils.assertThrowsAsync(async () => {
        await batchSubmitter.runTask(true)
      }, RollupBatchBlockNumberTooOldError)
    })

    it('should throw if older tx in safety queue', async () => {
      const nowSeconds = Math.round(new Date().getTime() / 1000)

      safetyQueue.timestamp = nowSeconds - 2

      setUpTask(nowSeconds - 1)

      await TestUtils.assertThrowsAsync(async () => {
        await batchSubmitter.runTask(true)
      }, RollupBatchSafetyQueueBlockTimestampError)
    })

    it('should throw if older tx in l1 to L2 queue', async () => {
      const nowSeconds = Math.round(new Date().getTime() / 1000)

      l1ToL2TransactionQueue.timestamp = nowSeconds - 2

      setUpTask(nowSeconds - 1)

      await TestUtils.assertThrowsAsync(async () => {
        await batchSubmitter.runTask(true)
      }, RollupBatchL1ToL2QueueBlockTimestampError)
    })

    it('should throw if older tx block in safety queue', async () => {
      safetyQueue.blockNumber = 4
      batchSubmitter.batchSubmissionBlockNumberOverride = 5
      canonicalProvider.blockNumberOverride = 6

      setUpTask()

      await TestUtils.assertThrowsAsync(async () => {
        await batchSubmitter.runTask(true)
      }, RollupBatchSafetyQueueBlockNumberError)
    })

    it('should throw if older tx block in l1 to L2 queue', async () => {
      l1ToL2TransactionQueue.blockNumber = 4
      batchSubmitter.batchSubmissionBlockNumberOverride = 5
      canonicalProvider.blockNumberOverride = 6

      setUpTask()

      await TestUtils.assertThrowsAsync(async () => {
        await batchSubmitter.runTask(true)
      }, RollupBatchL1ToL2QueueBlockNumberError)
    })

    it('should throw if older tx block in safety queue', async () => {
      const nowSeconds = Math.round(new Date().getTime() / 1000)

      canonicalTransactionChain.lastOvmTimestampSeconds = nowSeconds - 3

      setUpTask(nowSeconds - 5)

      await TestUtils.assertThrowsAsync(async () => {
        await batchSubmitter.runTask(true)
      }, RollupBatchOvmTimestampError)
    })

    it('should throw if older tx block in l1 to L2 queue', async () => {
      canonicalProvider.blockNumberOverride = 6
      canonicalTransactionChain.lastOvmBlock = 5
      batchSubmitter.batchSubmissionBlockNumberOverride = 4

      setUpTask()

      await TestUtils.assertThrowsAsync(async () => {
        await batchSubmitter.runTask(true)
      }, RollupBatchOvmBlockNumberError)
    })
  })
})
