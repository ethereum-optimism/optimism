import {
  DefaultDataService,
  CanonicalChainBatchCreator,
} from '../../src/app/data'

class MockDataService extends DefaultDataService {
  public invoked: boolean = false
  public builtBatch: number = -1

  constructor() {
    super(undefined)
  }

  public async tryBuildCanonicalChainBatchNotPresentOnL1(): Promise<number> {
    this.invoked = true
    if (this.builtBatch < -1) {
      throw Error('error')
    }
    return this.builtBatch
  }
}

describe('Optimistic Canonical Chain Batch Creator', () => {
  let batchCreator: CanonicalChainBatchCreator
  let dataService: MockDataService

  beforeEach(async () => {
    dataService = new MockDataService()
    batchCreator = new CanonicalChainBatchCreator(dataService)
  })

  it('should run successfully when no batch is built', async () => {
    await batchCreator.runTask()
    dataService.invoked.should.equal(
      true,
      'tryBuildCanonicalChainBatchNotPresentOnL1 not invoked!'
    )
  })

  it('should run successfully when Data Service says it built a batch', async () => {
    dataService.builtBatch = 0
    await batchCreator.runTask()

    dataService.invoked.should.equal(
      true,
      'tryBuildCanonicalChainBatchNotPresentOnL1 not invoked!'
    )
  })

  it('should run successfully when Data Service throws', async () => {
    dataService.builtBatch = -2
    await batchCreator.runTask()

    dataService.invoked.should.equal(
      true,
      'tryBuildCanonicalChainBatchNotPresentOnL1 not invoked!'
    )
  })
})
