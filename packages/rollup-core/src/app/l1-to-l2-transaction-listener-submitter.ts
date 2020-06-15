/* External Imports */
import { add0x, getLogger, logError } from '@eth-optimism/core-utils'

import { JsonRpcProvider } from 'ethers/providers'
import { Wallet } from 'ethers'

/* Internal Imports */
import { L1ToL2Transaction, L1ToL2TransactionListener } from '../types'
import { CHAIN_ID, GAS_LIMIT } from './constants'

const log = getLogger('l1-to-l2-tx-listener')

/**
 * Handles L1 To L2 Transactions, submitting them to the configured L2 node.
 */
export class L1ToL2TransactionListenerSubmitter
  implements L1ToL2TransactionListener {
  constructor(
    private readonly wallet: Wallet,
    private readonly provider: JsonRpcProvider
  ) {}

  public async handleL1ToL2Transaction(
    transaction: L1ToL2Transaction
  ): Promise<void> {
    log.debug(`Received L1 to L2 Transaction ${JSON.stringify(transaction)}`)

    const signedTx = await this.getSignedTransaction(transaction)

    log.debug(
      `sending signed tx. Tx: ${JSON.stringify(transaction)}. Signed: ${add0x(
        signedTx
      )}`
    )
    const receipt = await this.provider.sendTransaction(signedTx)

    log.debug(
      `L1 to L2 Transaction submitted. Tx hash: ${
        receipt.hash
      }. Tx: ${JSON.stringify(transaction)}`
    )
    try {
      const txReceipt = await this.provider.waitForTransaction(receipt.hash)
      if (!txReceipt || !txReceipt.status) {
        const msg = `Error processing L1 to L2 Transaction. Tx: ${JSON.stringify(
          transaction
        )}, Receipt: ${JSON.stringify(receipt)}`
        log.error(msg)
        throw new Error(msg)
      }
    } catch (e) {
      logError(
        log,
        `Error submitting L1 to L2 transaction to L2 node. Tx Hash: ${
          receipt.hash
        }, Tx: ${JSON.stringify(transaction)}`,
        e
      )
      throw e
    }
    log.debug(`L1 to L2 Transaction applied to L2. Tx hash: ${receipt.hash}`)
  }

  private async getSignedTransaction(
    transaction: L1ToL2Transaction
  ): Promise<string> {
    // TODO: change / append to calldata to add sender?

    const tx = {
      nonce: transaction.nonce,
      gasPrice: 0,
      gasLimit: GAS_LIMIT,
      to: transaction.target,
      value: 0,
      data: add0x(transaction.calldata),
      chainId: CHAIN_ID,
    }

    return this.wallet.sign(tx)
  }
}
