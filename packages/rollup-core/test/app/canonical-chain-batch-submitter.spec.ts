/* External Imports */
import { keccak256FromUtf8, sleep } from '@eth-optimism/core-utils'
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

interface BatchNumberHash {
  batchNumber: number
  txHash: string
}

class MockDataService extends DefaultDataService {
  public readonly nextBatch: TransactionBatchSubmission[] = []
  public readonly txBatchesSubmitted: BatchNumberHash[] = []
  public readonly txBatchesConfirmed: BatchNumberHash[] = []

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

  public async markTransactionBatchConfirmedOnL1(
    batchNumber: number,
    l1TxHash: string
  ): Promise<void> {
    this.txBatchesConfirmed.push({ batchNumber, txHash: l1TxHash })
  }
}

class MockProvider {
  public confirmedTxs: Map<string, TransactionReceipt> = new Map<
    string,
    TransactionReceipt
  >()

  public async waitForTransaction(
    hash: string,
    numConfirms: number
  ): Promise<TransactionReceipt> {
    while (!this.confirmedTxs.get(hash)) {
      await sleep(100)
    }
    return this.confirmedTxs.get(hash)
  }
}

class MockCanonicalTransactionChain {
  public responses: TransactionResponse[] = []

  constructor(public readonly provider: MockProvider) {}

  public async appendSequencerBatch(
    calldata: string,
    timestamp: number
  ): Promise<TransactionResponse> {
    const response: TransactionResponse = this.responses.shift()
    if (!response) {
      throw Error('no response')
    }
    return response
  }
}

class MockStateCommitmentChain {
  public responses: TransactionResponse[] = []

  constructor(public readonly provider: MockProvider) {}

  public async appendStateBatch(
    batches: string[]
  ): Promise<TransactionResponse> {
    const response: TransactionResponse = this.responses.shift()
    if (!response) {
      throw Error('no response')
    }
    return response
  }
}

describe('Canonical Chain Batch Submitter', () => {
  let batchSubmitter: CanonicalChainBatchSubmitter
  let dataService: MockDataService
  let canonicalProvider: MockProvider
  let canonicalTransactionChain: MockCanonicalTransactionChain

  beforeEach(async () => {
    dataService = new MockDataService()
    canonicalProvider = new MockProvider()
    canonicalTransactionChain = new MockCanonicalTransactionChain(
      canonicalProvider
    )
    batchSubmitter = new CanonicalChainBatchSubmitter(
      dataService,
      canonicalTransactionChain as any
    )
  })

  it('should not do anything if there are no batches', async () => {
    await batchSubmitter.runTask()

    dataService.txBatchesSubmitted.length.should.equal(
      0,
      'No tx batches should have been submitted!'
    )
    dataService.txBatchesConfirmed.length.should.equal(
      0,
      'No tx batches should have been confirmed!'
    )
  })

  it('should not do anything if the next batch has an invalid status', async () => {
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

    await batchSubmitter.runTask()

    dataService.txBatchesSubmitted.length.should.equal(
      0,
      'No tx batches should have been submitted!'
    )
    dataService.txBatchesConfirmed.length.should.equal(
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

    await batchSubmitter.runTask()

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

    dataService.txBatchesConfirmed.length.should.equal(
      1,
      'No tx batches confirmed!'
    )
    dataService.txBatchesConfirmed[0].txHash.should.equal(
      hash,
      'Incorrect tx hash confirmed!'
    )
    dataService.txBatchesConfirmed[0].batchNumber.should.equal(
      batchNumber,
      'Incorrect tx batch number confirmed!'
    )
  })
})
