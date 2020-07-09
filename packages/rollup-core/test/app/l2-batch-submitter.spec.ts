/* External Imports */
import { Wallet } from 'ethers'

/* Internal Imports */
import { DefaultDataService, L2BatchSubmitter } from '../../src/app/data'
import { DefaultL2NodeService } from '../../src/app'
import { BlockBatches } from '../../src/types'
import { keccak256FromUtf8 } from '@eth-optimism/core-utils/build'

class MockL2NodeService extends DefaultL2NodeService {
  public readonly sentBlockBatches: BlockBatches[] = []

  constructor() {
    super(Wallet.createRandom())
  }

  public async sendBlockBatches(blockBatches: BlockBatches): Promise<void> {
    this.sentBlockBatches.push(blockBatches)
  }
}

class MockL1DataService extends DefaultDataService {
  public readonly blockBatchesToReturn: BlockBatches[] = []
  public readonly batchesMarkedSubmitted: number[] = []
  constructor() {
    super(undefined)
  }

  public async getNextBatchForL2Submission(): Promise<BlockBatches> {
    return this.blockBatchesToReturn.shift()
  }

  public async markL1BatchSubmittedToL2(batchNumber: number): Promise<void> {
    this.batchesMarkedSubmitted.push(batchNumber)
  }
}

describe('L2 Batch Submitter', () => {
  let batchSubmitter: L2BatchSubmitter
  let l1DatService: MockL1DataService
  let l2NodeService: MockL2NodeService

  beforeEach(async () => {
    l1DatService = new MockL1DataService()
    l2NodeService = new MockL2NodeService()
    batchSubmitter = new L2BatchSubmitter(l1DatService, l2NodeService)
  })

  it('should not submit batch if no fitting L1 batch exists', async () => {
    await batchSubmitter.runTask()

    l1DatService.batchesMarkedSubmitted.length.should.equal(
      0,
      `No Batches should have been marked as sent!`
    )

    l2NodeService.sentBlockBatches.length.should.equal(
      0,
      `No Batches should have been sent!`
    )
  })

  it('should send a batch if a fitting one exists', async () => {
    const blockBatches: BlockBatches = {
      batchNumber: 1,
      timestamp: 1,
      blockNumber: 1,
      batches: [
        [
          {
            batchIndex: 1,
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
      ],
    }

    l1DatService.blockBatchesToReturn.push(blockBatches)
    await batchSubmitter.runTask()

    l2NodeService.sentBlockBatches.length.should.equal(
      1,
      `1 BlockBatches object should have been submitted!`
    )
    l2NodeService.sentBlockBatches[0].should.deep.equal(
      blockBatches,
      `Sent BlockBatches object doesn't match!`
    )

    l1DatService.batchesMarkedSubmitted.length.should.equal(
      1,
      `1 batch should have been marked submitted!`
    )

    l1DatService.batchesMarkedSubmitted[0].should.equal(
      1,
      `1 batch should have been marked submitted!`
    )
  })
})
