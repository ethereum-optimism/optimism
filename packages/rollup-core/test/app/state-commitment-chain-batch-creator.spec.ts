import {
  DefaultDataService,
  StateCommitmentChainBatchCreator,
} from '../../src/app/data'

class MockDataService extends DefaultDataService {
  public invokedL2OnlyBatch: boolean = false
  public builtL2OnlyBatch: number = -1

  public invokedL1BatchMatch: boolean = false
  public builtL2BatchToMatchL1: number = -1

  public alreadAppendedOnL1 = false

  constructor() {
    super(undefined)
  }

  public async isNextStateCommitmentChainBatchToBuildAlreadyAppendedOnL1(): Promise<
    boolean
  > {
    return this.alreadAppendedOnL1
  }

  public async tryBuildL2OnlyStateCommitmentChainBatch(
    minBatchSize: number,
    maxBatchSize: number
  ): Promise<number> {
    this.invokedL2OnlyBatch = true
    if (this.builtL2OnlyBatch < -1) {
      throw Error('error')
    }
    return this.builtL2OnlyBatch
  }

  public async tryBuildStateCommitmentChainBatchToMatchAppendedL1Batch(): Promise<
    number
  > {
    this.invokedL1BatchMatch = true
    if (this.builtL2BatchToMatchL1 < -1) {
      throw Error('error')
    }
    return this.builtL2BatchToMatchL1
  }
}

describe('State Commitment Chain Batch Creator', () => {
  let batchCreator: StateCommitmentChainBatchCreator
  let dataService: MockDataService

  beforeEach(async () => {
    dataService = new MockDataService()
    batchCreator = new StateCommitmentChainBatchCreator(dataService, 1, 10)
  })

  it('should try to build L1 batch if already appended on L1', async () => {
    dataService.alreadAppendedOnL1 = true
    await batchCreator.runTask()
    dataService.invokedL1BatchMatch.should.equal(
      true,
      'tryBuildStateCommitmentChainBatchToMatchAppendedL1Batch not invoked!'
    )

    dataService.invokedL2OnlyBatch.should.equal(
      false,
      'tryBuildL2OnlyStateCommitmentChainBatch invoked!'
    )
  })

  it('should try to build L2 batch if not already appended on L1', async () => {
    dataService.alreadAppendedOnL1 = false
    await batchCreator.runTask()
    dataService.invokedL1BatchMatch.should.equal(
      false,
      'tryBuildStateCommitmentChainBatchToMatchAppendedL1Batch invoked!'
    )

    dataService.invokedL2OnlyBatch.should.equal(
      true,
      'tryBuildL2OnlyStateCommitmentChainBatch not invoked!'
    )
  })

  it('should build L2 batch if not already appended on L1', async () => {
    dataService.alreadAppendedOnL1 = false
    dataService.builtL2OnlyBatch = 0
    await batchCreator.runTask()
    dataService.invokedL1BatchMatch.should.equal(
      false,
      'tryBuildStateCommitmentChainBatchToMatchAppendedL1Batch invoked!'
    )
    dataService.invokedL2OnlyBatch.should.equal(
      true,
      'tryBuildL2OnlyStateCommitmentChainBatch not invoked!'
    )
  })

  it('should build L1 batch if already appended on L1', async () => {
    dataService.alreadAppendedOnL1 = true
    dataService.builtL2BatchToMatchL1 = 0
    await batchCreator.runTask()
    dataService.invokedL1BatchMatch.should.equal(
      true,
      'tryBuildStateCommitmentChainBatchToMatchAppendedL1Batch not invoked!'
    )
    dataService.invokedL2OnlyBatch.should.equal(
      false,
      'tryBuildL2OnlyStateCommitmentChainBatch invoked!'
    )
  })

  it('should successfully run when tryBuildStateCommitmentChainBatchToMatchAppendedL1Batch throws', async () => {
    dataService.alreadAppendedOnL1 = true
    dataService.builtL2BatchToMatchL1 = -2
    await batchCreator.runTask()
    dataService.invokedL1BatchMatch.should.equal(
      true,
      'tryBuildStateCommitmentChainBatchToMatchAppendedL1Batch not invoked!'
    )
    dataService.invokedL2OnlyBatch.should.equal(
      false,
      'tryBuildL2OnlyStateCommitmentChainBatch invoked!'
    )
  })

  it('should successfully run when tryBuildL2OnlyStateCommitmentChainBatch throws', async () => {
    dataService.alreadAppendedOnL1 = false
    dataService.builtL2OnlyBatch = -2
    await batchCreator.runTask()
    dataService.invokedL1BatchMatch.should.equal(
      false,
      'tryBuildStateCommitmentChainBatchToMatchAppendedL1Batch invoked!'
    )
    dataService.invokedL2OnlyBatch.should.equal(
      true,
      'tryBuildL2OnlyStateCommitmentChainBatch not invoked!'
    )
  })
})
