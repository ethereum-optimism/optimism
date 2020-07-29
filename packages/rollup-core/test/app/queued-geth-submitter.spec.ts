/* External Imports */
import { Wallet } from 'ethers'

/* Internal Imports */
import { DefaultDataService, QueuedGethSubmitter } from '../../src/app/data'
import { DefaultL2NodeService } from '../../src/app'
import { GethSubmission } from '../../src/types'
import { keccak256FromUtf8 } from '@eth-optimism/core-utils/build'

class MockL2NodeService extends DefaultL2NodeService {
  public readonly sentSubmissions: GethSubmission[] = []

  constructor() {
    super(Wallet.createRandom())
  }

  public async sendGethSubmission(blockBatches: GethSubmission): Promise<void> {
    this.sentSubmissions.push(blockBatches)
  }
}

class MockL1DataService extends DefaultDataService {
  public readonly submissionsToReturn: GethSubmission[] = []
  public readonly submissionsMarkedSubmitted: number[] = []
  constructor() {
    super(undefined)
  }

  public async getNextQueuedGethSubmission(): Promise<GethSubmission> {
    return this.submissionsToReturn.shift()
  }

  public async markQueuedGethSubmissionSubmittedToGeth(
    batchNumber: number
  ): Promise<void> {
    this.submissionsMarkedSubmitted.push(batchNumber)
  }
}

describe('Optimistic Canonical Chain Batch Submitter', () => {
  let batchSubmitter: QueuedGethSubmitter
  let l1DatService: MockL1DataService
  let l2NodeService: MockL2NodeService

  beforeEach(async () => {
    l1DatService = new MockL1DataService()
    l2NodeService = new MockL2NodeService()
    batchSubmitter = new QueuedGethSubmitter(l1DatService, l2NodeService)
  })

  it('should not submit batch if no fitting L1 batch exists', async () => {
    await batchSubmitter.runTask()

    l1DatService.submissionsMarkedSubmitted.length.should.equal(
      0,
      `No Batches should have been marked as sent!`
    )

    l2NodeService.sentSubmissions.length.should.equal(
      0,
      `No Batches should have been sent!`
    )
  })

  it('should send a batch if a fitting one exists', async () => {
    const blockBatches: GethSubmission = {
      batchNumber: 1,
      timestamp: 1,
      blockNumber: 1,
      rollupTransactions: [
        {
          indexWithinSubmission: 1,
          gasLimit: 0,
          nonce: 0,
          sender: Wallet.createRandom().address,
          target: Wallet.createRandom().address,
          calldata: keccak256FromUtf8('calldata'),
          l1Timestamp: 1,
          l1BlockNumber: 1,
          l1TxHash: keccak256FromUtf8('tx hash'),
          l1TxIndex: 0,
          l1TxLogIndex: 0,
          queueOrigin: 1,
        },
      ],
    }

    l1DatService.submissionsToReturn.push(blockBatches)
    await batchSubmitter.runTask()

    l2NodeService.sentSubmissions.length.should.equal(
      1,
      `1 BlockBatches object should have been submitted!`
    )
    l2NodeService.sentSubmissions[0].should.deep.equal(
      blockBatches,
      `Sent BlockBatches object doesn't match!`
    )

    l1DatService.submissionsMarkedSubmitted.length.should.equal(
      1,
      `1 batch should have been marked submitted!`
    )

    l1DatService.submissionsMarkedSubmitted[0].should.equal(
      1,
      `1 batch should have been marked submitted!`
    )
  })
})
