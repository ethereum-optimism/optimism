/* External Imports */
import {
  add0x,
  getLogger,
  isValidHexAddress,
  logError,
  remove0x,
} from '@eth-optimism/core-utils'

import { JsonRpcProvider } from 'ethers/providers'
import { Wallet } from 'ethers'

/* Internal Imports */
import {
  Address,
  L1ToL2TransactionBatch,
  L1ToL2TransactionBatchListener,
} from '../types'
import { CHAIN_ID, GAS_LIMIT } from './constants'

const log = getLogger('l1-to-l2-tx-batch-listener-submitter')

/**
 * Handles L1 To L2 TransactionBatches, submitting them to the configured L2 node.
 */
export class L1ToL2TransactionBatchListenerSubmitter
  implements L1ToL2TransactionBatchListener {
  private readonly submissionToAddress: Address
  private readonly submissionMethodId: string

  constructor(
    private readonly wallet: Wallet,
    private readonly provider: JsonRpcProvider,
    submissionToAddress: Address,
    submissionMethodId: string
  ) {
    if (!submissionMethodId || remove0x(submissionMethodId).length !== 8) {
      throw Error(
        `Invalid Transaction Batch submission method ID: ${remove0x(
          submissionMethodId
        )}. Expected 4 bytes (8 hex chars).`
      )
    }
    if (!isValidHexAddress(submissionToAddress)) {
      throw Error(
        `Invalid Transaction Batch submission to address: ${remove0x(
          submissionToAddress
        )}. Expected 20 bytes (40 hex chars).`
      )
    }
    this.submissionToAddress = add0x(submissionToAddress)
    this.submissionMethodId = add0x(submissionMethodId)
  }

  public async handleTransactionBatch(
    transactionBatch: L1ToL2TransactionBatch
  ): Promise<void> {
    log.debug(
      `Received L1 to L2 Transaction Batch ${JSON.stringify(transactionBatch)}`
    )

    const signedTx = await this.getSignedBatchTransaction(transactionBatch)

    log.debug(
      `sending signed tx for batch. Tx: ${JSON.stringify(
        transactionBatch
      )}. Signed: ${add0x(signedTx)}`
    )
    const receipt = await this.provider.sendTransaction(signedTx)

    log.debug(
      `L1 to L2 Transaction Batch Tx submitted. Tx hash: ${
        receipt.hash
      }. Tx batch: ${JSON.stringify(transactionBatch)}`
    )
    try {
      const txReceipt = await this.provider.waitForTransaction(receipt.hash)
      if (!txReceipt || !txReceipt.status) {
        const msg = `Error processing L1 to L2 Transaction Batch. Tx batch: ${JSON.stringify(
          transactionBatch
        )}, Receipt: ${JSON.stringify(receipt)}`
        log.error(msg)
        throw new Error(msg)
      }
    } catch (e) {
      logError(
        log,
        `Error submitting L1 to L2 Transaction Batch to L2 node. Tx Hash: ${
          receipt.hash
        }, Tx: ${JSON.stringify(transactionBatch)}`,
        e
      )
      throw e
    }
    log.debug(`L1 to L2 Transaction applied to L2. Tx hash: ${receipt.hash}`)
  }

  private async getSignedBatchTransaction(
    transactionBatch: L1ToL2TransactionBatch
  ): Promise<string> {
    const tx = {
      nonce: transactionBatch.nonce,
      gasPrice: 0,
      gasLimit: GAS_LIMIT,
      to: this.submissionToAddress,
      value: 0,
      data: await this.getL2TransactionBatchCalldata(transactionBatch.calldata),
      chainId: CHAIN_ID,
    }

    return this.wallet.sign(tx)
  }

  private async getL2TransactionBatchCalldata(
    l1Calldata: string
  ): Promise<string> {
    const l1CalldataParams =
      !l1Calldata || remove0x(l1Calldata).length < 8
        ? 0
        : remove0x(l1Calldata).substr(8)
    return `${this.submissionMethodId}${l1CalldataParams}`
  }
}
