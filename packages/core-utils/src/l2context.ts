import cloneDeep from 'lodash/cloneDeep'
import { providers } from 'ethers'

/**
 * Helper for adding additional L2 context to transactions
 */
export const injectL2Context = (l1Provider: providers.JsonRpcProvider) => {
  const provider = cloneDeep(l1Provider)

  // Pass through the state root
  const blockFormat = provider.formatter.block.bind(provider.formatter)
  provider.formatter.block = (block) => {
    const b = blockFormat(block)
    b.stateRoot = block.stateRoot
    return b
  }

  // Pass through the state root and additional tx data
  const blockWithTransactions = provider.formatter.blockWithTransactions.bind(
    provider.formatter
  )
  provider.formatter.blockWithTransactions = (block) => {
    const b = blockWithTransactions(block)
    b.stateRoot = block.stateRoot
    for (let i = 0; i < b.transactions.length; i++) {
      b.transactions[i].l1BlockNumber = block.transactions[i].l1BlockNumber
      if (b.transactions[i].l1BlockNumber != null) {
        b.transactions[i].l1BlockNumber = parseInt(
          b.transactions[i].l1BlockNumber,
          16
        )
      }
      b.transactions[i].l1TxOrigin = block.transactions[i].l1TxOrigin
      b.transactions[i].queueOrigin = block.transactions[i].queueOrigin
      b.transactions[i].rawTransaction = block.transactions[i].rawTransaction
    }
    return b
  }

  // Handle additional tx data
  const formatTxResponse = provider.formatter.transactionResponse.bind(
    provider.formatter
  )
  provider.formatter.transactionResponse = (transaction) => {
    const tx = formatTxResponse(transaction) as any
    tx.txType = transaction.txType
    tx.queueOrigin = transaction.queueOrigin
    tx.rawTransaction = transaction.rawTransaction
    tx.l1BlockNumber = transaction.l1BlockNumber
    if (tx.l1BlockNumber != null) {
      tx.l1BlockNumber = parseInt(tx.l1BlockNumber, 16)
    }
    tx.l1TxOrigin = transaction.l1TxOrigin
    return tx
  }

  return provider
}
