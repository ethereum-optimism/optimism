/* External Imports */
import {
  abiEncodeL2ToL1Message,
  L2ToL1Message,
} from '@eth-optimism/rollup-core'
import { getLogger, logError } from '@eth-optimism/core-utils'
import { Contract, Wallet } from 'ethers'

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
  private highestNonceSubmitted: number
  private highestNonceConfirmed: number

  public static async create(
    wallet: Wallet,
    messageReceiverContract: Contract
  ): Promise<DefaultL2ToL1MessageSubmitter> {
    return new DefaultL2ToL1MessageSubmitter(wallet, messageReceiverContract)
  }

  private constructor(
    private readonly wallet: Wallet,
    private readonly messageReceiverContract: Contract
  ) {
    this.highestNonceSubmitted = -1
    this.highestNonceConfirmed = -1
  }

  public async submitMessage(message: L2ToL1Message): Promise<void> {
    log.info(
      `Submitting message number ${message.nonce}: ${JSON.stringify(message)}.`
    )

    let receipt
    try {
      const callData = this.messageReceiverContract.interface.functions.enqueueL2ToL1Message.encode(
        [message]
      )
      receipt = await this.wallet.sendTransaction({
        to: this.messageReceiverContract.address,
        data: callData,
      })

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

    this.highestNonceSubmitted = Math.max(
      this.highestNonceSubmitted,
      message.nonce
    )

    this.messageReceiverContract.provider
      .waitForTransaction(receipt.hash)
      .then((txReceipt: TransactionReceipt) => {
        log.debug(
          `L2ToL1Message with nonce ${message.nonce.toString(
            16
          )} was confirmed on L1!`
        )
        this.highestNonceConfirmed = Math.max(
          this.highestNonceConfirmed,
          message.nonce
        )
      })
      .catch((error) => {
        logError(log, 'Error submitting L2 -> L1 message transaction', error)
      })
  }

  public getHighestNonceSubmitted(): number {
    return this.highestNonceSubmitted
  }

  public getHighestNonceConfirmed(): number {
    return this.highestNonceConfirmed
  }
}
