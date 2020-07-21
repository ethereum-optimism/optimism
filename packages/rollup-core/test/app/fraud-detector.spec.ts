/* External Imports */
import { TestUtils } from '@eth-optimism/core-utils'

/* Internal Imports */
import { VerifierDataService } from '../../src/types/data'
import { FraudProver, VerificationCandidate } from '../../src/types'
import { FraudDetector } from '../../src/app'

class MockVerifierDataService implements VerifierDataService {
  public readonly verificationCandidates: VerificationCandidate[] = []
  public readonly batchesVerified: number[] = []

  public async getNextVerificationCandidate(): Promise<VerificationCandidate> {
    return this.verificationCandidates.shift()
  }

  public async verifyStateRootBatch(batchNumber): Promise<void> {
    this.batchesVerified.push(batchNumber)
  }
}

interface Fraud {
  batchNumber: number
  batchIndex: number
}
class MockFraudProver implements FraudProver {
  public readonly provenFraud: Fraud[] = []

  public async proveFraud(
    batchNumber: number,
    batchIndex: number
  ): Promise<void> {
    this.provenFraud.push({
      batchNumber,
      batchIndex,
    })
  }
}

describe('Fraud Detector', () => {
  let dataService: MockVerifierDataService
  let fraudProver: MockFraudProver
  let fraudDetector: FraudDetector

  beforeEach(() => {
    dataService = new MockVerifierDataService()
    fraudProver = new MockFraudProver()
    fraudDetector = new FraudDetector(dataService, fraudProver, 10_000, 3)
  })

  it('should verify valid candidates', async () => {
    const roots = [
      { l1Root: 'a', gethRoot: 'a' },
      { l1Root: 'b', gethRoot: 'b' },
      { l1Root: 'c', gethRoot: 'c' },
    ]
    dataService.verificationCandidates.push({
      batchNumber: 1,
      roots,
    })

    await fraudDetector.runTask()

    dataService.batchesVerified.length.should.equal(1, `Batch not verified!`)
    dataService.batchesVerified[0].should.equal(
      1,
      `Batch 1 should be verified!`
    )

    fraudProver.provenFraud.length.should.equal(
      0,
      `No fraud should have been proven!`
    )
  })

  it('should not verify undefined candidates', async () => {
    await fraudDetector.runTask()

    dataService.batchesVerified.length.should.equal(
      0,
      `Batch should not be verified!`
    )
    fraudProver.provenFraud.length.should.equal(
      0,
      `No fraud should have been proven!`
    )
  })

  it('should not verify candidates without roots', async () => {
    dataService.verificationCandidates.push({
      batchNumber: 1,
      roots: [],
    })

    await TestUtils.assertThrowsAsync(async () => {
      await fraudDetector.runTask()
    })

    dataService.batchesVerified.length.should.equal(
      0,
      `Batch should not be verified!`
    )
    fraudProver.provenFraud.length.should.equal(
      0,
      `No fraud should have been proven!`
    )
  })

  it('should not verify candidates without batch number', async () => {
    dataService.verificationCandidates.push({
      batchNumber: undefined,
      roots: [{ l1Root: 'a', gethRoot: 'a' }],
    })

    await TestUtils.assertThrowsAsync(async () => {
      await fraudDetector.runTask()
    })

    dataService.batchesVerified.length.should.equal(
      0,
      `Batch should not be verified!`
    )
    fraudProver.provenFraud.length.should.equal(
      0,
      `No fraud should have been proven!`
    )
  })

  it('should prove fraud on position 0 state root mismatch', async () => {
    const roots = [
      { l1Root: 'a', gethRoot: 'b' },
      { l1Root: 'b', gethRoot: 'b' },
      { l1Root: 'c', gethRoot: 'c' },
    ]

    dataService.verificationCandidates.push({
      batchNumber: 1,
      roots,
    })

    await fraudDetector.runTask()

    dataService.batchesVerified.length.should.equal(
      0,
      `Batch should not be verified!`
    )
    fraudProver.provenFraud.length.should.equal(
      1,
      `Fraud should have been proven!`
    )
    fraudProver.provenFraud[0].batchNumber.should.equal(
      1,
      `Incorrect fraud batch number!`
    )
    fraudProver.provenFraud[0].batchIndex.should.equal(
      0,
      `Incorrect fraud batch index!`
    )
  })

  it('should prove fraud on position 1 state root mismatch', async () => {
    const roots = [
      { l1Root: 'a', gethRoot: 'a' },
      { l1Root: 'b', gethRoot: 'c' },
      { l1Root: 'c', gethRoot: 'c' },
    ]

    dataService.verificationCandidates.push({
      batchNumber: 1,
      roots,
    })

    await fraudDetector.runTask()

    dataService.batchesVerified.length.should.equal(
      0,
      `Batch should not be verified!`
    )
    fraudProver.provenFraud.length.should.equal(
      1,
      `Fraud should have been proven!`
    )
    fraudProver.provenFraud[0].batchNumber.should.equal(
      1,
      `Incorrect fraud batch number!`
    )
    fraudProver.provenFraud[0].batchIndex.should.equal(
      1,
      `Incorrect fraud batch index!`
    )
  })

  it('should only prove fraud once for the same fraud', async () => {
    const roots = [
      { l1Root: 'a', gethRoot: 'a' },
      { l1Root: 'b', gethRoot: 'c' },
      { l1Root: 'c', gethRoot: 'c' },
    ]

    dataService.verificationCandidates.push({
      batchNumber: 1,
      roots,
    })

    await fraudDetector.runTask()
    await fraudDetector.runTask()

    dataService.batchesVerified.length.should.equal(
      0,
      `Batch should not be verified!`
    )
    fraudProver.provenFraud.length.should.equal(
      1,
      `Fraud should have been proven!`
    )
    fraudProver.provenFraud[0].batchNumber.should.equal(
      1,
      `Incorrect fraud batch number!`
    )
    fraudProver.provenFraud[0].batchIndex.should.equal(
      1,
      `Incorrect fraud batch index!`
    )
  })

  it('should recover after proving fraud', async () => {
    const fraudRoots = [
      { l1Root: 'a', gethRoot: 'a' },
      { l1Root: 'b', gethRoot: 'c' },
      { l1Root: 'c', gethRoot: 'c' },
    ]

    dataService.verificationCandidates.push({
      batchNumber: 1,
      roots: fraudRoots,
    })

    await fraudDetector.runTask()

    dataService.verificationCandidates.push({
      batchNumber: 1,
      roots: fraudRoots,
    })
    await fraudDetector.runTask()

    dataService.batchesVerified.length.should.equal(
      0,
      `Batch should not be verified!`
    )
    fraudProver.provenFraud.length.should.equal(
      1,
      `Fraud should have been proven!`
    )
    fraudProver.provenFraud[0].batchNumber.should.equal(
      1,
      `Incorrect fraud batch number!`
    )
    fraudProver.provenFraud[0].batchIndex.should.equal(
      1,
      `Incorrect fraud batch index!`
    )

    const nonFraudRroots = [
      { l1Root: 'a', gethRoot: 'a' },
      { l1Root: 'b', gethRoot: 'b' },
      { l1Root: 'c', gethRoot: 'c' },
    ]

    dataService.verificationCandidates.push({
      batchNumber: 1,
      roots: nonFraudRroots,
    })

    await fraudDetector.runTask()

    dataService.batchesVerified.length.should.equal(
      1,
      `Batch should not be verified!`
    )
    dataService.batchesVerified[0].should.equal(
      1,
      `Batch 1 should be verified!`
    )

    dataService.verificationCandidates.push({
      batchNumber: 2,
      roots: fraudRoots,
    })

    await fraudDetector.runTask()

    dataService.batchesVerified.length.should.equal(
      1,
      `Only batch 1 should not be verified!`
    )
    fraudProver.provenFraud.length.should.equal(
      2,
      `Second fraud should have been proven!`
    )
    fraudProver.provenFraud[1].batchNumber.should.equal(
      2,
      `Incorrect second fraud batch number!`
    )
    fraudProver.provenFraud[1].batchIndex.should.equal(
      1,
      `Incorrect second fraud batch index!`
    )
  })
})
