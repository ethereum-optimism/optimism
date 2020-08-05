/* External Imports */
import {
  getLogger,
  Logger,
  numberToHexString,
  strToHexStr,
} from '@eth-optimism/core-utils'

import { JsonRpcProvider } from 'ethers/providers'
import { Wallet } from 'ethers'

/* Internal Imports */
import { GethSubmission, L2NodeService } from '../types'

const log: Logger = getLogger('block-batch-submitter')

export class DefaultL2NodeService implements L2NodeService {
  // params: [gethSubmissionJSONString, signedGethSubmissionJSONString]
  // -- note all numbers are replaces with hex strings when serialized
  public static readonly sendGethSubmission: string =
    'eth_sendRollupTransactions'

  private readonly l2Provider: JsonRpcProvider

  constructor(private readonly l2Wallet: Wallet) {
    this.l2Provider = l2Wallet.provider as JsonRpcProvider
  }

  /**
   * @inheritDoc
   */
  public async sendGethSubmission(
    gethSubmission: GethSubmission
  ): Promise<void> {
    if (!gethSubmission) {
      const msg = `Received undefined Geth Submission!.`
      log.error(msg)
      throw msg
    }

    if (
      !gethSubmission.rollupTransactions ||
      !gethSubmission.rollupTransactions.length
    ) {
      log.error(
        `Received empty Geth Submission: ${JSON.stringify(gethSubmission)}`
      )
      return
    }

    const payload = JSON.stringify(gethSubmission, (k, v) => {
      if (typeof v === 'number') {
        return v >= 0 ? numberToHexString(v) : undefined
      }
      return v
    })

    const hexPayload: string = strToHexStr(payload)
    const signedPayload: string = await this.l2Wallet.signMessage(hexPayload)

    await this.l2Provider.send(DefaultL2NodeService.sendGethSubmission, [
      [hexPayload, signedPayload],
    ])
  }
}
