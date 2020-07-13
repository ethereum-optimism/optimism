import { DefaultDataService, L2BatchCreator } from '../../src/app/data'
import { L1BatchRecord } from '../../src/types/data'

class MockDataService extends DefaultDataService {
  public l2OnlyBatchesBuilt: number = 0
  public l1MatchingBatchesBuilt: number = 0
  public unverifiedL1Batches: L1BatchRecord[] = []

  constructor() {
    super(undefined)
  }

  public async getOldestUnverifiedL1TransactionBatch(): Promise<L1BatchRecord> {
    if (this.unverifiedL1Batches.length > 0) {
      return this.unverifiedL1Batches[0]
    }
    return undefined
  }

  public async tryBuildL2OnlyBatch(): Promise<number> {
    this.l2OnlyBatchesBuilt++
    return
  }

  public async tryBuildL2BatchToMatchL1(
    l1BatchSize: number,
    l1BatchNumber: number
  ): Promise<number> {
    this.l1MatchingBatchesBuilt++
    return this.l1MatchingBatchesBuilt
  }
}

describe('L2 Batch Creator', () => {
  let batchCreator: L2BatchCreator
  let dataService: MockDataService

  beforeEach(async () => {
    dataService = new MockDataService()
    batchCreator = new L2BatchCreator(dataService)
  })

  it('should try to build L2 only batch when no unverified L1 batches exist', async () => {
    await batchCreator.runTask()

    dataService.l2OnlyBatchesBuilt.should.equal(
      1,
      `No L2 only batches should have been attempted!`
    )
    dataService.l1MatchingBatchesBuilt.should.equal(
      0,
      `No L1 matching batches should have been attempted!`
    )
  })

  it('should try to build a matching batch when there is an unverified L1 batch', async () => {
    dataService.unverifiedL1Batches.push({
      batchNumber: 1,
      batchSize: 2,
      blockTimestamp: 3,
    })
    await batchCreator.runTask()

    dataService.l2OnlyBatchesBuilt.should.equal(
      0,
      `No L2 only batches should have been attempted!`
    )
    dataService.l1MatchingBatchesBuilt.should.equal(
      1,
      `1 L1 matching batche should have been attempted!`
    )
  })
})
