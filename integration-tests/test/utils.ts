import { remove0x } from '@eth-optimism/core-utils'
import { JsonRpcProvider } from '@ethersproject/providers'
import cloneDeep from 'lodash/cloneDeep'

import { utils, providers, Transaction } from 'ethers'

/**
 * Helper for adding additional L2 context to transactions
 */
export const injectL2Context = (l1Provider: providers.JsonRpcProvider) => {
  const provider = cloneDeep(l1Provider)
  const format = provider.formatter.transaction.bind(provider.formatter)
  provider.formatter.transaction = (transaction) => {
    const tx = format(transaction)
    const sig = utils.joinSignature(tx)
    const hash = sighashEthSign(tx)
    tx.from = utils.verifyMessage(hash, sig)
    return tx
  }

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
      b.transactions[i].txType = block.transactions[i].txType
      b.transactions[i].queueOrigin = block.transactions[i].queueOrigin
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

function serializeEthSignTransaction(transaction: Transaction): any {
  const encoded = utils.defaultAbiCoder.encode(
    ['uint256', 'uint256', 'uint256', 'uint256', 'address', 'bytes'],
    [
      transaction.nonce,
      transaction.gasLimit,
      transaction.gasPrice,
      transaction.chainId,
      transaction.to,
      transaction.data,
    ]
  )

  return Buffer.from(encoded.slice(2), 'hex')
}

// Use this function as input to `eth_sign`. It does not
// add the prefix because `eth_sign` does that. It does
// serialize the transaction and hash the serialized
// transaction.
function sighashEthSign(transaction: any): Buffer {
  const serialized = serializeEthSignTransaction(transaction)
  const hash = remove0x(utils.keccak256(serialized))
  return Buffer.from(hash, 'hex')
}
