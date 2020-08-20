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
  BatchSubmissionStatus,
  StateCommitmentBatchSubmission,
} from '../../src/types/data'
import { StateCommitmentChainBatchSubmitter } from '../../src/app/data/consumers/state-commitment-chain-batch-submitter'

interface BatchNumberHash {
  batchNumber: number
  txHash: string
}

class MockDataService extends DefaultDataService {
  public readonly nextBatch: StateCommitmentBatchSubmission[] = []
  public readonly stateRootBatchesSubmitted: BatchNumberHash[] = []
  public readonly stateRootBatchesFinalized: BatchNumberHash[] = []

  constructor() {
    super(undefined)
  }

  public async getNextStateCommitmentBatchToSubmit(): Promise<
    StateCommitmentBatchSubmission
  > {
    return this.nextBatch.shift()
  }

  public async markStateRootBatchSubmittedToL1(
    batchNumber: number,
    l1TxHash: string
  ): Promise<void> {
    this.stateRootBatchesSubmitted.push({ batchNumber, txHash: l1TxHash })
  }

  public async markStateRootBatchFinalOnL1(
    batchNumber: number,
    l1TxHash: string
  ): Promise<void> {
    this.stateRootBatchesFinalized.push({ batchNumber, txHash: l1TxHash })
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
}

class MockStateCommitmentChain {
  public responses: TransactionResponse[] = []

  constructor(public readonly provider: MockProvider) {}

  public async appendStateBatch(
    stateRoots: string[]
  ): Promise<TransactionResponse> {
    const response: TransactionResponse = this.responses.shift()
    if (!response) {
      throw Error('no response')
    }
    return response
  }
}

describe('State Commitment Chain Batch Submitter', () => {
  let batchSubmitter: StateCommitmentChainBatchSubmitter
  let dataService: MockDataService
  let provider: MockProvider
  let stateCommitmentChain: MockStateCommitmentChain

  beforeEach(async () => {
    dataService = new MockDataService()
    provider = new MockProvider()
    stateCommitmentChain = new MockStateCommitmentChain(provider)
    batchSubmitter = new StateCommitmentChainBatchSubmitter(
      dataService,
      stateCommitmentChain as any
    )
  })

  it('should not do anything if there are no batches', async () => {
    const res = await batchSubmitter.runTask()

    res.should.equal(false, 'Incorrect result when there are no batches')

    dataService.stateRootBatchesSubmitted.length.should.equal(
      0,
      'No state root batches should have been submitted!'
    )
    dataService.stateRootBatchesFinalized.length.should.equal(
      0,
      'No state root batches should have been confirmed!'
    )
  })

  it('should throw if the next batch has an invalid status', async () => {
    dataService.nextBatch.push({
      submissionTxHash: undefined,
      status: 'derp' as any,
      batchNumber: 1,
      stateRoots: [keccak256FromUtf8('root 1'), keccak256FromUtf8('root 2')],
    })

    await TestUtils.assertThrowsAsync(async () => {
      await batchSubmitter.runTask()
    })

    dataService.stateRootBatchesSubmitted.length.should.equal(
      0,
      'No state root batches should have been submitted!'
    )
    dataService.stateRootBatchesFinalized.length.should.equal(
      0,
      'No state root batches should have been confirmed!'
    )
  })

  it('should send roots if there is a batch', async () => {
    const hash: string = keccak256FromUtf8('tx hash')
    const stateRoots: string[] = [
      keccak256FromUtf8('root 1'),
      keccak256FromUtf8('root 2'),
    ]
    const batchNumber: number = 1
    dataService.nextBatch.push({
      submissionTxHash: undefined,
      status: BatchSubmissionStatus.QUEUED,
      batchNumber,
      stateRoots,
    })

    stateCommitmentChain.responses.push({ hash } as any)
    provider.txReceipts.set(hash, { status: 1 } as any)

    const res: boolean = await batchSubmitter.runTask()
    res.should.equal(true, `Batch should have been submitted successfully.`)

    dataService.stateRootBatchesSubmitted.length.should.equal(
      1,
      'No state root batches submitted!'
    )
    dataService.stateRootBatchesSubmitted[0].txHash.should.equal(
      hash,
      'Incorrect tx hash submitted!'
    )
    dataService.stateRootBatchesSubmitted[0].batchNumber.should.equal(
      batchNumber,
      'Incorrect tx batch number submitted!'
    )

    dataService.stateRootBatchesFinalized.length.should.equal(
      0,
      'No state root batches should be confirmed!'
    )
  })

  it('should not mark batch as submitted if batch submission tx fails', async () => {
    const hash: string = keccak256FromUtf8('tx hash')
    const stateRoots: string[] = [
      keccak256FromUtf8('root 1'),
      keccak256FromUtf8('root 2'),
    ]
    const batchNumber: number = 1
    dataService.nextBatch.push({
      submissionTxHash: undefined,
      status: BatchSubmissionStatus.QUEUED,
      batchNumber,
      stateRoots,
    })

    stateCommitmentChain.responses.push({ hash } as any)
    provider.txReceipts.set(hash, { status: 0 } as any)

    const res: boolean = await batchSubmitter.runTask()
    res.should.equal(false, `Batch tx should have errored out.`)

    dataService.stateRootBatchesSubmitted.length.should.equal(
      0,
      'No state root batches should have been submitted!'
    )

    dataService.stateRootBatchesFinalized.length.should.equal(
      0,
      'No state root batches should be confirmed!'
    )
  })
})
