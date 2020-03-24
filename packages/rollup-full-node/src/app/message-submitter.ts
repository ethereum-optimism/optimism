/* External Imports */
import {
  abiEncodeL2ToL1Message,
  L2ToL1Message,
} from '@eth-optimism/rollup-core'
import { getLogger, logError } from '@eth-optimism/core-utils'
import { Contract } from 'ethers'

import { L2ToL1MessageSubmitter } from '../types'
import { TransactionReceipt } from 'ethers/providers/abstract-provider'

const log = getLogger('rollup-message-submitter')

export class NoOpL2ToL1MessageSubmitter implements L2ToL1MessageSubmitter {
  public async submitMessage(l2ToL1Message: L2ToL1Message): Promise<void> {
    log.debug(
      `L2ToL1Message received by NoOpL2ToL1MessageSubmitter: ${JSON.stringify(
        NoOpL2ToL1MessageSubmitter
      )}`
    )
    return
  }
}

/**
 *  Default Message Submitter implementation. This will be deprecated when message submission works properly.
 */
export class DefaultL2ToL1MessageSubmitter implements L2ToL1MessageSubmitter {
  public static async create(
    messageReceiverContract: Contract
  ): Promise<DefaultL2ToL1MessageSubmitter> {
    return new DefaultL2ToL1MessageSubmitter(messageReceiverContract)
  }

  private constructor(private readonly messageReceiverContract: Contract) {}

  public async submitMessage(message: L2ToL1Message): Promise<void> {
    log.info(
      `Submitting message number ${message.nonce}: ${JSON.stringify(message)}.`
    )

    let receipt
    try {
      receipt = await this.messageReceiverContract.enqueueL2ToL1Message(
        DefaultL2ToL1MessageSubmitter.serializeRollupMessageForSubmission(
          message
        )
      )
      log.debug(
        `Receipt for message ${JSON.stringify(message)}: ${JSON.stringify(
          receipt
        )}`
      )
    } catch (e) {
      logError(
        log,
        `Error submitting rollup message: ${JSON.stringify(message)}`,
        e
      )
      throw e
    }

    this.messageReceiverContract.provider
      .waitForTransaction(receipt)
      .then((txReceipt: TransactionReceipt) => {
        log.debug(
          `L2ToL1Message with nonce ${message.nonce.toString(
            'hex'
          )} was confirmed on L1!`
        )
      })
      .catch((error) => {
        logError(log, 'Error submitting L2 -> L1 message transaction', error)
      })
  }

  public static serializeRollupMessageForSubmission(
    rollupMessage: L2ToL1Message
  ): string {
    return abiEncodeL2ToL1Message(rollupMessage)
  }
}
