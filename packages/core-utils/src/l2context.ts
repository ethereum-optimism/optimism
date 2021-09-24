import cloneDeep from 'lodash/cloneDeep'
import { providers, BigNumber } from 'ethers'

const parseNumber = (n: string | number): number => {
  if (typeof n === 'string' && n.startsWith('0x')) {
    return parseInt(n, 16)
  }
  if (typeof n === 'number') {
    return n
  }
  return parseInt(n, 10)
}

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
        b.transactions[i].l1BlockNumber = parseNumber(
          b.transactions[i].l1BlockNumber
        )
      }
      b.transactions[i].l1Timestamp = block.transactions[i].l1Timestamp
      if (b.transactions[i].l1Timestamp != null) {
        b.transactions[i].l1Timestamp = parseNumber(
          b.transactions[i].l1Timestamp
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

  const formatReceiptResponse = provider.formatter.receipt.bind(
    provider.formatter
  )
  provider.formatter.receipt = (receipt) => {
    const r = formatReceiptResponse(receipt)
    r.l1GasPrice = BigNumber.from(receipt.l1GasPrice)
    r.l1GasUsed = BigNumber.from(receipt.l1GasUsed)
    r.l1Fee = BigNumber.from(receipt.l1Fee)
    r.l1FeeScalar = parseFloat(receipt.l1FeeScalar)
    return r
  }

  return provider
}
