/* External Imports */
import { getLogger, logError, ScheduledTask } from '@eth-optimism/core-utils'

/* Internal Imports */
import { L1DataService } from '../../../types/data'

const log = getLogger('queued-geth-submitter')

/**
 * Polls the DB to queue L1 Transaction batches for submission to geth.
 */
export class GethSubmissionQueuer extends ScheduledTask {
  constructor(
    private readonly dataService: L1DataService,
    private readonly queueOriginsToSendToGeth: number[],
    periodMilliseconds = 10_000
  ) {
    super(periodMilliseconds)
  }

  /**
   * @inheritDoc
   *
   * Creates Geth Submission Queue batches from L1 Transactions in the DB.
   *
   */
  public async runTask(): Promise<void> {
    // TODO: Leaving this here as a placeholder, but I think we'll implement this in geth
    try {
      const queueIndex = await this.dataService.queueNextGethSubmission(
        this.queueOriginsToSendToGeth
      )
      if (queueIndex < 0) {
        log.debug(`No transactions present to queue for Geth submission.`)
      }
      log.debug(`Queued submission number ${queueIndex} to send to Geth.`)
    } catch (e) {
      logError(log, `Error queueing transactions for submission to Geth`, e)
      // swallow exception.
    }
  }
}
