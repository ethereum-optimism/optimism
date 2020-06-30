/* External Imports */
import { getLogger, sleep } from '@eth-optimism/core-utils'

/* Internal Imports */
import { VerifierDataService } from '../types/data'
import { VerificationCandidate } from '../types'

const log = getLogger('verifier')

/**
 * Polls the DB for VerificationCandidates to ensure that L1 rollup Txs match L2 Txs.
 *
 */
export class Verifier {
  private static readonly ALERT_EVERY: number = 6

  private stopped: boolean
  private fraudCount: number

  constructor(
    private readonly dataService: VerifierDataService,
    private readonly fraudProver: any,
    private readonly sleepDelayMillis = 10_000
  ) {
    this.stopped = true
    this.fraudCount = 0
  }

  public async run(): Promise<void> {
    while (!this.stopped) {
      const verifierCandidate: VerificationCandidate = await this.dataService.getVerificationCandidate()
      if (
        !verifierCandidate ||
        verifierCandidate.l1BatchNumber === undefined ||
        verifierCandidate.l2BatchNumber === undefined
      ) {
        await sleep(this.sleepDelayMillis)
        continue
      }

      if (verifierCandidate.l1BatchNumber !== verifierCandidate.l2BatchNumber) {
        log.error(
          `Batch number mismatch! L1 Batch Number: ${verifierCandidate.l1BatchNumber}, L2 Batch Number: ${verifierCandidate.l2BatchNumber}`
        )
      }

      for (let i = 0; i < verifierCandidate.roots.length; i++) {
        const root = verifierCandidate.roots[i]
        if (root.l1Root !== root.l2Root) {
          if (this.fraudCount % Verifier.ALERT_EVERY === 0) {
            log.error(
              `Batch #${verifierCandidate.l1BatchNumber} state roots differ at index ${i}! L1 root: ${root.l1Root}, L2 root: ${root.l2Root}`
            )
            // TODO: Wire up fraud prover
            this.fraudProver.proveFraud(verifierCandidate.l1BatchNumber, i)
          }
          this.fraudCount++
          await sleep(this.sleepDelayMillis)
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
}
