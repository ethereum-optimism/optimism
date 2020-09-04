/* External Imports */
import {
  keccak256,
  keccak256FromUtf8,
  sleep,
  TestUtils,
} from '@eth-optimism/core-utils'
import { TransactionReceipt, TransactionResponse } from 'ethers/providers'
import { Contract, Wallet } from 'ethers'

/* Internal Imports */
import {
  DefaultDataService,
  CanonicalChainBatchSubmitter,
} from '../../src/app/data'
import {
  BatchSubmissionStatus,
  L2DataService,
  StateCommitmentBatchSubmission,
} from '../../src/types/data'
import { StateCommitmentChainBatchSubmitter } from '../../src/app/data/consumers/state-commitment-chain-batch-submitter'
import { UnexpectedBatchStatus } from '../../src/types'

interface BatchNumberHash {
  batchNumber: number
  txHash: string
}

class TestStateCommitmentChainBatchSubmitter extends StateCommitmentChainBatchSubmitter {
  public signedRollupStateRootBatchTxOverride: string = Buffer.from(
    `signed tx`,
    'utf-8'
  ).toString('hex')

  constructor(
    dataService: L2DataService,
    stateCommitmentChainContract: Contract,
    periodMilliseconds = 10_000
  ) {
    super(
      dataService,
      stateCommitmentChainContract,
      Wallet.createRandom(),
      periodMilliseconds
    )
  }

  protected async getSignedRollupBatchTx(
    stateRoots: string[],
    startIndex: number
  ): Promise<string> {
    return this.signedRollupStateRootBatchTxOverride
  }
}

class MockDataService extends DefaultDataService {
  public readonly nextBatch: StateCommitmentBatchSubmission[] = []
  public readonly stateRootBatchesSubmitting: BatchNumberHash[] = []
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

  public async markStateRootBatchSubmittingToL1(
    batchNumber: number,
    l1TxHash: string
  ): Promise<void> {
    this.stateRootBatchesSubmitting.push({ batchNumber, txHash: l1TxHash })
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
  public readonly submittedTxs: string[] = []
  public txReceipts: Map<string, TransactionReceipt> = new Map<
    string,
    TransactionReceipt
  >()

  public txResponses: Map<string, TransactionResponse> = new Map<
    string,
    TransactionResponse
  >()

  public txExists: boolean = true
  public blockNumberOverride: number

  public async getTransaction(hash: string): Promise<any> {
    return this.txExists ? this.txResponses.get(hash) : false
  }

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

  public async sendTransaction(signedTx: string): Promise<TransactionResponse> {
    const hash: string = keccak256(signedTx)
    this.submittedTxs.push(hash)
    if (!this.txResponses.has(hash)) {
      throw Error(`tx threw`)
    }
    return this.txResponses.get(hash)
  }
}

class MockStateCommitmentChain {
  public responses: TransactionResponse[] = []

  constructor(public readonly provider: MockProvider) {}
}

describe('State Commitment Chain Batch Submitter', () => {
  let batchSubmitter: TestStateCommitmentChainBatchSubmitter
  let dataService: MockDataService
  let provider: MockProvider
  let stateCommitmentChain: MockStateCommitmentChain

  beforeEach(async () => {
    dataService = new MockDataService()
    provider = new MockProvider()
    stateCommitmentChain = new MockStateCommitmentChain(provider)
    batchSubmitter = new TestStateCommitmentChainBatchSubmitter(
      dataService,
      stateCommitmentChain as any
    )
  })

  it('should not do anything if there are no batches', async () => {
    const res = await batchSubmitter.runTask()

    res.should.equal(false, 'Incorrect result when there are no batches')

    provider.submittedTxs.length.should.equal(
      0,
      `No state batches should have been appended!`
    )
    dataService.stateRootBatchesSubmitting.length.should.equal(
      0,
      'No state root batches should have been marked as submitting!'
    )
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
    }, UnexpectedBatchStatus)

    provider.submittedTxs.length.should.equal(
      0,
      `No state batches should have been appended!`
    )
    dataService.stateRootBatchesSubmitting.length.should.equal(
      0,
      'No state root batches should have been marked as submitting!'
    )
    dataService.stateRootBatchesSubmitted.length.should.equal(
      0,
      'No state root batches should have been submitted!'
    )
    dataService.stateRootBatchesFinalized.length.should.equal(
      0,
      'No state root batches should have been confirmed!'
    )
  })

  it('should send roots if there is a batch in QUEUED state', async () => {
    const hash: string = keccak256(
      batchSubmitter.signedRollupStateRootBatchTxOverride
    )
    const stateRoots: string[] = [
      keccak256FromUtf8('root 1'),
      keccak256FromUtf8('root 2'),
    ]
    const batchNumber: number = 1
    dataService.nextBatch.push({
      submissionTxHash: hash,
      status: BatchSubmissionStatus.QUEUED,
      batchNumber,
      stateRoots,
    })

    stateCommitmentChain.responses.push({ hash } as any)
    provider.txReceipts.set(hash, { status: 1 } as any)
    provider.txResponses.set(hash, { hash } as any)

    const res: boolean = await batchSubmitter.runTask()
    res.should.equal(true, `Batch should have been submitted successfully.`)

    provider.submittedTxs.length.should.equal(
      1,
      `1 State batch should have been appended!`
    )
    dataService.stateRootBatchesSubmitting.length.should.equal(
      1,
      'No state root batches marked as submitting!'
    )
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

  it('should wait for tx confirmation there is a batch in SUBMITTING status that has been submitted', async () => {
    const hash: string = keccak256(
      batchSubmitter.signedRollupStateRootBatchTxOverride
    )
    const stateRoots: string[] = [
      keccak256FromUtf8('root 1'),
      keccak256FromUtf8('root 2'),
    ]
    const batchNumber: number = 1
    dataService.nextBatch.push({
      submissionTxHash: hash,
      status: BatchSubmissionStatus.SUBMITTING,
      batchNumber,
      stateRoots,
    })

    stateCommitmentChain.responses.push({ hash } as any)
    provider.txReceipts.set(hash, { status: 1 } as any)
    provider.txResponses.set(hash, { hash } as any)

    const res: boolean = await batchSubmitter.runTask()
    res.should.equal(true, `Batch should have been submitted successfully.`)

    provider.submittedTxs.length.should.equal(
      0,
      `Batch should not be re-submitted!`
    )
    dataService.stateRootBatchesSubmitting.length.should.equal(
      0,
      'Batch should not be marked as submitting again!'
    )
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

  it('should wait for tx confirmation there is a batch in SUBMITTING status that has not been submitted', async () => {
    const hash: string = keccak256(
      batchSubmitter.signedRollupStateRootBatchTxOverride
    )
    const stateRoots: string[] = [
      keccak256FromUtf8('root 1'),
      keccak256FromUtf8('root 2'),
    ]
    const batchNumber: number = 1
    dataService.nextBatch.push({
      submissionTxHash: hash,
      status: BatchSubmissionStatus.SUBMITTING,
      batchNumber,
      stateRoots,
    })

    stateCommitmentChain.responses.push({ hash } as any)
    provider.txReceipts.set(hash, { status: 1 } as any)
    provider.txResponses.set(hash, { hash } as any)
    provider.txExists = false

    const res: boolean = await batchSubmitter.runTask()
    res.should.equal(true, `Batch should have been submitted successfully.`)

    provider.submittedTxs.length.should.equal(
      1,
      `Batch should not be re-submitted!`
    )
    dataService.stateRootBatchesSubmitting.length.should.equal(
      1,
      'Batch should not be marked as submitting again!'
    )
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
    const hash: string = keccak256(
      batchSubmitter.signedRollupStateRootBatchTxOverride
    )
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
    provider.txResponses.set(hash, { hash } as any)

    const res: boolean = await batchSubmitter.runTask()
    res.should.equal(false, `Batch tx should have errored out.`)

    provider.submittedTxs.length.should.equal(
      1,
      `1 State batch should have been appended!`
    )
    provider.submittedTxs[0].should.equal(hash, `Incorrect tx submitted!`)
    dataService.stateRootBatchesSubmitting.length.should.equal(
      1,
      'No state root batches marked as submitting!'
    )
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
