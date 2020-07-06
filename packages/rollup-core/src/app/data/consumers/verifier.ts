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
 * Polls the DB for VerificationCandidates to ensure that L1 rollup Txs match L2 Txs.
 *
 */
export class Verifier extends ScheduledTask {
  private static readonly ALERT_EVERY: number = 6

  private fraudCount: number

  constructor(
    private readonly dataService: VerifierDataService,
    private readonly fraudProver: FraudProver,
    periodMilliseconds = 10_000,
    private readonly reAlertEveryNFailures = Verifier.ALERT_EVERY
  ) {
    super(periodMilliseconds)
    this.fraudCount = 0
  }

  public async runTask(): Promise<void> {
    const verifierCandidate: VerificationCandidate = await this.dataService.getVerificationCandidate()
    if (
      !verifierCandidate ||
      verifierCandidate.l1BatchNumber === undefined ||
      verifierCandidate.l2BatchNumber === undefined
    ) {
      return
    }

    if (verifierCandidate.l1BatchNumber !== verifierCandidate.l2BatchNumber) {
      const msg: string = `Batch number mismatch! L1 Batch Number: ${verifierCandidate.l1BatchNumber}, L2 Batch Number: ${verifierCandidate.l2BatchNumber}`
      log.error(msg)
      throw Error(msg)
    }

    for (let i = 0; i < verifierCandidate.roots.length; i++) {
      const root = verifierCandidate.roots[i]
      if (root.l1Root !== root.l2Root) {
        if (this.fraudCount % Verifier.ALERT_EVERY === 0) {
          log.error(
            `Batch #${verifierCandidate.l1BatchNumber} state roots differ at index ${i}! L1 root: ${root.l1Root}, L2 root: ${root.l2Root}`
          )
          await this.fraudProver.proveFraud(verifierCandidate.l1BatchNumber, i)
        }
        this.fraudCount++
        return
      }
    }

    if (this.fraudCount > 0) {
      this.fraudCount = 0
      log.info(
        `Fraud has been corrected for batch ${verifierCandidate.l1BatchNumber}`
      )
    }

    log.debug(`Batch #${verifierCandidate.l1BatchNumber} has been verified!`)
    await this.dataService.verifyBatch(verifierCandidate.l1BatchNumber)
  }
}
