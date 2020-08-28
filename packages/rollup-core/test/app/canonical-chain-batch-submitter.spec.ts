/* External Imports */
import { keccak256FromUtf8, sleep, TestUtils } from '@eth-optimism/core-utils'
import { TransactionReceipt, TransactionResponse } from 'ethers/providers'
import { Wallet } from 'ethers'

/* Internal Imports */
import {
  DefaultDataService,
  CanonicalChainBatchSubmitter,
} from '../../src/app/data'
import {
  TransactionBatchSubmission,
  BatchSubmissionStatus,
} from '../../src/types/data'
import { UnexpectedBatchStatus } from '../../src/types'

interface BatchNumberHash {
  batchNumber: number
  txHash: string
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
    return this.txReceipts.size
  }
}

class MockCanonicalTransactionChain {
  public responses: TransactionResponse[] = []

  constructor(public readonly provider: MockProvider) {}

  public async appendSequencerBatch(
    calldata: string,
    timestamp: number,
    blockNumber: number
  ): Promise<TransactionResponse> {
    const response: TransactionResponse = this.responses.shift()
    if (!response) {
      throw Error('no response')
    }
    return response
  }
}

class MockQueue {
  public timestamp: number = 0
  public blockNumber: number = 0

  public async peekBlockNumber(): Promise<number> {
    return this.blockNumber
  }

  public async peekTimestamp(): Promise<number> {
    return this.blockNumber
  }
}

describe('Canonical Chain Batch Submitter', () => {
  let batchSubmitter: CanonicalChainBatchSubmitter
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
    l1ToL2TransactionQueue = new MockQueue()
    safetyQueue = new MockQueue()
    batchSubmitter = new CanonicalChainBatchSubmitter(
      dataService,
      canonicalTransactionChain as any,
      l1ToL2TransactionQueue as any,
      safetyQueue as any
    )
  })

  it('should not do anything if there are no batches', async () => {
    const res = await batchSubmitter.runTask()

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
      await batchSubmitter.runTask()
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

    const res: boolean = await batchSubmitter.runTask()
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

    const res: boolean = await batchSubmitter.runTask()
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
})
