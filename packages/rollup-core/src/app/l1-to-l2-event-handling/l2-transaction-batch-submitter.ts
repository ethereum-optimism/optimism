/* External Imports */
import { getLogger, Logger, numberToHexString } from '@eth-optimism/core-utils'

/* Internal Imports */
import {
  L1ToL2TransactionBatch,
  L1ToL2TransactionBatchListener,
} from '../../types'
import { JsonRpcProvider } from 'ethers/providers'
import { Wallet } from 'ethers'

const log: Logger = getLogger('l1-to-l2-transition-synchronizer')

// params: [timestampHex, transactionsArrayJSON, signedTransactionsArrayJSON]
const sendL1ToL2TransactionsMethod: string = 'optimism_sendL1ToL2Transactions'

export class L2TransactionBatchSubmitter
  implements L1ToL2TransactionBatchListener {
  private readonly l2Provider: JsonRpcProvider

  constructor(private readonly l2Wallet: Wallet) {
    this.l2Provider = l2Wallet.provider as JsonRpcProvider
  }

  /**
   * @inheritDoc
   */
  public async handleL1ToL2TransactionBatch(
    transactionBatch: L1ToL2TransactionBatch
  ): Promise<void> {
    if (!transactionBatch) {
      const msg = `Received undefined Transaction Batch!.`
      log.error(msg)
      throw msg
    }

    if (
      !transactionBatch.transactions ||
      !transactionBatch.transactions.length
    ) {
      log.debug(`Moving past empty block ${transactionBatch.blockNumber}.`)
      return
    }

    const timestamp: string = numberToHexString(transactionBatch.timestamp)
    const txs = JSON.stringify(
      transactionBatch.transactions.map((x) => {
        return {
          nonce: x.nonce > 0 ? numberToHexString(x.nonce) : '',
          sender: x.sender,
          calldata: x.calldata,
        }
      })
    )
    const signedTxsArray: string = await this.l2Wallet.signMessage(txs)
    await this.l2Provider.send(sendL1ToL2TransactionsMethod, [
      timestamp,
      txs,
      signedTxsArray,
    ])
  }
}
