/* External Imports */
import { getLogger, ScheduledTask } from '@eth-optimism/core-utils'

/* Internal Imports */
import {
  VerifierDataService,
  VerificationCandidate,
  FraudProver,
} from '../../../types'

const log = getLogger('verifier')

/**
 * Polls the DB for VerificationCandidates to ensure that L1 rollup Txs match L2 Tx Outputs.
 */
export class FraudDetector extends ScheduledTask {
  private static readonly ALERT_EVERY: number = 6

  private fraudCount: number

  constructor(
    private readonly dataService: VerifierDataService,
    private readonly fraudProver: FraudProver,
    periodMilliseconds = 10_000,
    private readonly reAlertEveryNFailures = FraudDetector.ALERT_EVERY
  ) {
    super(periodMilliseconds)
    this.fraudCount = 0
  }

  public async runTask(): Promise<void> {
    const verifierCandidate: VerificationCandidate = await this.dataService.getNextVerificationCandidate()
    if (!verifierCandidate) {
      log.debug(`No verifier candidate is available, returning...`)
      return
    }

    if (verifierCandidate.batchNumber === undefined) {
      const msg = `Fraud Detector received verification candidate with null batch number! This should never happen!`
      log.error(msg)
      throw Error(msg)
    }

    if (!verifierCandidate.roots || verifierCandidate.roots.length === 0) {
      const msg = `Verification candidate with batch number ${verifierCandidate.batchNumber} has no roots! This should never happen!`
      log.error(msg)
      throw Error(msg)
    }

    for (let i = 0; i < verifierCandidate.roots.length; i++) {
      const root = verifierCandidate.roots[i]
      if (root.l1Root !== root.gethRoot) {
        if (this.fraudCount % FraudDetector.ALERT_EVERY === 0) {
          log.error(
            `Batch #${verifierCandidate.batchNumber} state roots differ at index ${i}! L1 root: ${root.l1Root}, Geth root: ${root.gethRoot}`
          )
          if (this.fraudCount === 0) {
            await this.dataService.markVerificationCandidateFraudulent(
              verifierCandidate.batchNumber
            )
          }
          if (!!this.fraudProver) {
            await this.fraudProver.proveFraud(verifierCandidate.batchNumber, i)
          } else {
            // TODO: take this away and make Fraud Prover mandatory when Fraud Prover exists.
            log.error(`No Fraud Prover Configured!`)
          }
        }
        this.fraudCount++
        return
      }
    }

    if (this.fraudCount > 0) {
      this.fraudCount = 0
      log.info(
        `Fraud has been corrected for batch ${verifierCandidate.batchNumber}`
      )
    }

    log.debug(`Batch #${verifierCandidate.batchNumber} has been verified!`)
    await this.dataService.verifyStateRootBatch(verifierCandidate.batchNumber)
  }
}
