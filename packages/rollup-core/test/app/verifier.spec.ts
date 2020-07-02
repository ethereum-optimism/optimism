/* External Imports */
import { TestUtils } from '@eth-optimism/core-utils'

/* Internal Imports */
import { VerifierDataService } from '../../src/types/data'
import { FraudProver, VerificationCandidate } from '../../src/types'
import { Verifier } from '../../src/app'

class MockVerifierDataService implements VerifierDataService {
  public readonly verificationCandidates: VerificationCandidate[] = []
  public readonly batchesVerified: number[] = []

  public async getVerificationCandidate(): Promise<VerificationCandidate> {
    return this.verificationCandidates.shift()
  }

  public async verifyBatch(batchNumber): Promise<void> {
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

describe('Verifier', () => {
  let dataService: MockVerifierDataService
  let fraudProver: MockFraudProver
  let verifier: Verifier

  beforeEach(() => {
    dataService = new MockVerifierDataService()
    fraudProver = new MockFraudProver()
    verifier = new Verifier(dataService, fraudProver, 10_000, 3)
  })

  it('should verify valid candidates', async () => {
    const roots = [
      { l1Root: 'a', l2Root: 'a' },
      { l1Root: 'b', l2Root: 'b' },
      { l1Root: 'c', l2Root: 'c' },
    ]
    dataService.verificationCandidates.push({
      l1BatchNumber: 1,
      l2BatchNumber: 1,
      roots,
    })

    await verifier.runTask()

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

  it('should verify candidates without roots', async () => {
    dataService.verificationCandidates.push({
      l1BatchNumber: 1,
      l2BatchNumber: 1,
      roots: [],
    })

    await verifier.runTask()

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
    await verifier.runTask()

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
      l1BatchNumber: undefined,
      l2BatchNumber: undefined,
      roots: [{ l1Root: 'a', l2Root: 'a' }],
    })

    await verifier.runTask()

    dataService.batchesVerified.length.should.equal(
      0,
      `Batch should not be verified!`
    )
    fraudProver.provenFraud.length.should.equal(
      0,
      `No fraud should have been proven!`
    )
  })

  it('should not verify candidates without l1 batch number', async () => {
    dataService.verificationCandidates.push({
      l1BatchNumber: undefined,
      l2BatchNumber: 1,
      roots: [{ l1Root: 'a', l2Root: 'a' }],
    })

    await verifier.runTask()

    dataService.batchesVerified.length.should.equal(
      0,
      `Batch should not be verified!`
    )
    fraudProver.provenFraud.length.should.equal(
      0,
      `No fraud should have been proven!`
    )
  })

  it('should not verify candidates without l2 batch number', async () => {
    dataService.verificationCandidates.push({
      l1BatchNumber: 1,
      l2BatchNumber: undefined,
      roots: [{ l1Root: 'a', l2Root: 'a' }],
    })

    await verifier.runTask()

    dataService.batchesVerified.length.should.equal(
      0,
      `Batch should not be verified!`
    )
    fraudProver.provenFraud.length.should.equal(
      0,
      `No fraud should have been proven!`
    )
  })

  it('should throw on batch # mismatch', async () => {
    dataService.verificationCandidates.push({
      l1BatchNumber: 1,
      l2BatchNumber: 2,
      roots: [{ l1Root: 'a', l2Root: 'a' }],
    })

    await TestUtils.assertThrowsAsync(async () => {
      await verifier.runTask()
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
      { l1Root: 'a', l2Root: 'b' },
      { l1Root: 'b', l2Root: 'b' },
      { l1Root: 'c', l2Root: 'c' },
    ]

    dataService.verificationCandidates.push({
      l1BatchNumber: 1,
      l2BatchNumber: 1,
      roots,
    })

    await verifier.runTask()

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
      { l1Root: 'a', l2Root: 'a' },
      { l1Root: 'b', l2Root: 'c' },
      { l1Root: 'c', l2Root: 'c' },
    ]

    dataService.verificationCandidates.push({
      l1BatchNumber: 1,
      l2BatchNumber: 1,
      roots,
    })

    await verifier.runTask()

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
})
