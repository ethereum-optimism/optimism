import {
  DefaultDataService,
  StateCommitmentChainBatchFinalizer,
} from '../../src/app/data'
import { BatchSubmission, BatchSubmissionStatus } from '../../src/types/data'
import { keccak256FromUtf8, TestUtils } from '@eth-optimism/core-utils/build'
import { JsonRpcProvider, Provider, TransactionReceipt } from 'ethers/providers'
import { sleep } from '@eth-optimism/core-utils/build/src'

class MockDataService extends DefaultDataService {
  public throwOnFinalize: boolean = false
  public getWasInvoked: boolean = false
  public batchSubmissions: BatchSubmission[] = []
  public finalized: Map<number, BatchSubmission> = new Map<
    number,
    BatchSubmission
  >()

  constructor() {
    super(undefined)
  }

  public async getNextStateCommitmentBatchToFinalize(): Promise<
    BatchSubmission
  > {
    this.getWasInvoked = true
    return this.batchSubmissions.shift()
  }

  public async markStateRootBatchFinalOnL1(
    batchNumber: number,
    submissionTxHash: string
  ): Promise<void> {
    if (this.throwOnFinalize) {
      throw Error('You told me to...')
    }

    this.finalized.set(batchNumber, {
      batchNumber,
      submissionTxHash,
      status: BatchSubmissionStatus.FINALIZED,
    })
  }
}

class MockProvider extends JsonRpcProvider {
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
}

describe('State Commitment Chain Batch Finalizer', () => {
  let batchFinalizer: StateCommitmentChainBatchFinalizer
  let dataService: MockDataService
  let provider: MockProvider
  beforeEach(async () => {
    dataService = new MockDataService()
    provider = new MockProvider()
    batchFinalizer = new StateCommitmentChainBatchFinalizer(
      dataService,
      provider,
      1,
      10
    )
  })

  it('should run successfully when no batch is built', async () => {
    const res = await batchFinalizer.runTask()
    res.should.equal(false, 'No batch should have been finalized')
    dataService.getWasInvoked.should.equal(
      true,
      'getNextStateCommitmentBatchToFinalize not invoked!'
    )
  })

  it('should throw if returned batch submission is not SENT', async () => {
    const submissionTxHash: string = keccak256FromUtf8('tx1')
    const batchNumber: number = 0
    dataService.batchSubmissions.push({
      batchNumber,
      submissionTxHash,
      status: BatchSubmissionStatus.QUEUED,
    })

    provider.txReceipts.set(submissionTxHash, { status: 1 } as any)

    TestUtils.assertThrowsAsync(async () => {
      await batchFinalizer.runTask()
    })

    dataService.getWasInvoked.should.equal(
      true,
      'getNextCanonicalChainTransactionBatchToFinalize not invoked!'
    )

    const finalized: boolean = !!dataService.finalized.get(0)
    finalized.should.equal(false, 'Should not be finalized!')
  })

  it('should run successfully when Data Service says it built a batch', async () => {
    const submissionTxHash: string = keccak256FromUtf8('tx1')
    const batchNumber: number = 0
    dataService.batchSubmissions.push({
      batchNumber,
      submissionTxHash,
      status: BatchSubmissionStatus.SENT,
    })

    provider.txReceipts.set(submissionTxHash, { status: 1 } as any)

    const res = await batchFinalizer.runTask()
    res.should.equal(true, 'No batch was finalized!')

    dataService.getWasInvoked.should.equal(
      true,
      'getNextStateCommitmentBatchToFinalize not invoked!'
    )

    const submission: BatchSubmission = dataService.finalized.get(0)
    submission.submissionTxHash.should.equal(
      submissionTxHash,
      'Finalized batch tx mismatch!'
    )
    submission.batchNumber.should.equal(0, 'Finalized batch tx mismatch!')
  })

  it('should return false when tx status returns 0', async () => {
    const submissionTxHash: string = keccak256FromUtf8('tx1')
    const batchNumber: number = 0
    dataService.batchSubmissions.push({
      batchNumber,
      submissionTxHash,
      status: BatchSubmissionStatus.SENT,
    })

    provider.txReceipts.set(submissionTxHash, { status: 0 } as any)

    const res = await batchFinalizer.runTask()
    res.should.equal(false, 'Batch was finalized!')

    dataService.getWasInvoked.should.equal(
      true,
      'getNextStateCommitmentBatchToFinalize not invoked!'
    )

    const submissionExists: boolean = !!dataService.finalized.get(0)
    submissionExists.should.equal(false, 'Submission should not exist!')
  })

  it('should return false when data service throws marking it final', async () => {
    const submissionTxHash: string = keccak256FromUtf8('tx1')
    const batchNumber: number = 0
    dataService.batchSubmissions.push({
      batchNumber,
      submissionTxHash,
      status: BatchSubmissionStatus.SENT,
    })

    provider.txReceipts.set(submissionTxHash, { status: 1 } as any)
    dataService.throwOnFinalize = true

    const res = await batchFinalizer.runTask()
    res.should.equal(false, 'Batch was finalized!')

    dataService.getWasInvoked.should.equal(
      true,
      'getNextStateCommitmentBatchToFinalize not invoked!'
    )

    const submissionExists: boolean = !!dataService.finalized.get(0)
    submissionExists.should.equal(false, 'Submission should not exist!')
  })
})
