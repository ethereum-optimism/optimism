import { Provider, TransactionRequest } from '@ethersproject/abstract-provider'
import { serialize } from '@ethersproject/transactions'
import { Contract, BigNumber } from 'ethers'
import { predeploys, getContractInterface } from '@eth-optimism/contracts'
import cloneDeep from 'lodash/cloneDeep'

import { assert } from './utils/assert'
import { L2Provider, ProviderLike, NumberLike } from './interfaces'
import { toProvider, toNumber, toBigNumber } from './utils'

type ProviderTypeIsWrong = any

/**
 * Gets a reasonable nonce for the transaction.
 *
 * @param provider Provider to get the nonce from.
 * @param tx Requested transaction.
 * @returns A reasonable nonce for the transaction.
 */
const getNonceForTx = async (
  provider: ProviderLike,
  tx: TransactionRequest
): Promise<number> => {
  if (tx.nonce !== undefined) {
    return toNumber(tx.nonce as NumberLike)
  } else if (tx.from !== undefined) {
    return toProvider(provider).getTransactionCount(tx.from)
  } else {
    // Large nonce with lots of non-zero bytes
    return 0xffffffff
  }
}

/**
 * Returns a Contract object for the GasPriceOracle.
 *
 * @param provider Provider to attach the contract to.
 * @returns Contract object for the GasPriceOracle.
 */
const connectGasPriceOracle = (provider: ProviderLike): Contract => {
  return new Contract(
    predeploys.OVM_GasPriceOracle,
    getContractInterface('OVM_GasPriceOracle'),
    toProvider(provider)
  )
}

/**
 * Gets the current L1 gas price as seen on L2.
 *
 * @param l2Provider L2 provider to query the L1 gas price from.
 * @returns Current L1 gas price as seen on L2.
 */
export const getL1GasPrice = async (
  l2Provider: ProviderLike
): Promise<BigNumber> => {
  const gpo = connectGasPriceOracle(l2Provider)
  return gpo.l1BaseFee()
}

/**
 * Estimates the amount of L1 gas required for a given L2 transaction.
 *
 * @param l2Provider L2 provider to query the gas usage from.
 * @param tx Transaction to estimate L1 gas for.
 * @returns Estimated L1 gas.
 */
export const estimateL1Gas = async (
  l2Provider: ProviderLike,
  tx: TransactionRequest
): Promise<BigNumber> => {
  const gpo = connectGasPriceOracle(l2Provider)
  return gpo.getL1GasUsed(
    serialize({
      data: tx.data,
      to: tx.to,
      gasPrice: tx.gasPrice,
      type: tx.type,
      gasLimit: tx.gasLimit,
      nonce: await getNonceForTx(l2Provider, tx),
    })
  )
}

/**
 * Estimates the amount of L1 gas cost for a given L2 transaction in wei.
 *
 * @param l2Provider L2 provider to query the gas usage from.
 * @param tx Transaction to estimate L1 gas cost for.
 * @returns Estimated L1 gas cost.
 */
export const estimateL1GasCost = async (
  l2Provider: ProviderLike,
  tx: TransactionRequest
): Promise<BigNumber> => {
  const gpo = connectGasPriceOracle(l2Provider)
  return gpo.getL1Fee(
    serialize({
      data: tx.data,
      to: tx.to,
      gasPrice: tx.gasPrice,
      type: tx.type,
      gasLimit: tx.gasLimit,
      nonce: await getNonceForTx(l2Provider, tx),
    })
  )
}

/**
 * Estimates the L2 gas cost for a given L2 transaction in wei.
 *
 * @param l2Provider L2 provider to query the gas usage from.
 * @param tx Transaction to estimate L2 gas cost for.
 * @returns Estimated L2 gas cost.
 */
export const estimateL2GasCost = async (
  l2Provider: ProviderLike,
  tx: TransactionRequest
): Promise<BigNumber> => {
  const parsed = toProvider(l2Provider)
  const l2GasPrice = await parsed.getGasPrice()
  const l2GasCost = await parsed.estimateGas(tx)
  return l2GasPrice.mul(l2GasCost)
}

/**
 * Estimates the total gas cost for a given L2 transaction in wei.
 *
 * @param l2Provider L2 provider to query the gas usage from.
 * @param tx Transaction to estimate total gas cost for.
 * @returns Estimated total gas cost.
 */
export const estimateTotalGasCost = async (
  l2Provider: ProviderLike,
  tx: TransactionRequest
): Promise<BigNumber> => {
  const l1GasCost = await estimateL1GasCost(l2Provider, tx)
  const l2GasCost = await estimateL2GasCost(l2Provider, tx)
  return l1GasCost.add(l2GasCost)
}

/**
 * Determines if a given Provider is an L2Provider.  Will coerce type
 * if true
 *
 * @param provider The provider to check
 * @returns Boolean
 * @example
 * if (isL2Provider(provider)) {
 *   // typescript now knows it is of type L2Provider
 *   const gasPrice = await provider.estimateL2GasPrice(tx)
 * }
 */
export const isL2Provider = <TProvider extends Provider>(
  provider: TProvider
): provider is L2Provider<TProvider> => {
  return Boolean((provider as L2Provider<TProvider>)._isL2Provider)
}

/**
 * Returns an provider wrapped as an Optimism L2 provider. Adds a few extra helper functions to
 * simplify the process of estimating the gas usage for a transaction on Optimism. Returns a COPY
 * of the original provider.
 *
 * @param provider Provider to wrap into an L2 provider.
 * @returns Provider wrapped as an L2 provider.
 */
export const asL2Provider = <TProvider extends Provider>(
  provider: TProvider
): L2Provider<TProvider> => {
  // Skip if we've already wrapped this provider.
  if (isL2Provider(provider)) {
    return provider
  }

  // Make a copy of the provider since we'll be modifying some internals and don't want to mess
  // with the original object.
  const l2Provider = cloneDeep(provider) as L2Provider<TProvider>

  // Not exactly sure when the provider wouldn't have a formatter function, but throw an error if
  // it doesn't have one. The Provider type doesn't define it but every provider I've dealt with
  // seems to have it.
  // TODO this may be fixed if library has gotten updated since
  const formatter = (l2Provider as ProviderTypeIsWrong).formatter
  assert(formatter, `provider.formatter must be defined`)

  // Modify the block formatter to return the state root. Not strictly related to Optimism, just a
  // generally useful thing that really should've been on the Ethers block object to begin with.
  // TODO: Maybe we should make a PR to add this to the Ethers library?
  const ogBlockFormatter = formatter.block.bind(formatter)
  formatter.block = (block: any) => {
    const parsed = ogBlockFormatter(block)
    parsed.stateRoot = block.stateRoot
    return parsed
  }

  // Modify the block formatter to include all the L2 fields for transactions.
  const ogBlockWithTxFormatter = formatter.blockWithTransactions.bind(formatter)
  formatter.blockWithTransactions = (block: any) => {
    const parsed = ogBlockWithTxFormatter(block)
    parsed.stateRoot = block.stateRoot
    parsed.transactions = parsed.transactions.map((tx: any, idx: number) => {
      const ogTx = block.transactions[idx]
      tx.l1BlockNumber = ogTx.l1BlockNumber
        ? toNumber(ogTx.l1BlockNumber)
        : ogTx.l1BlockNumber
      tx.l1Timestamp = ogTx.l1Timestamp
        ? toNumber(ogTx.l1Timestamp)
        : ogTx.l1Timestamp
      tx.l1TxOrigin = ogTx.l1TxOrigin
      tx.queueOrigin = ogTx.queueOrigin
      tx.rawTransaction = ogTx.rawTransaction
      return tx
    })
    return parsed
  }

  // Modify the transaction formatter to include all the L2 fields for transactions.
  const ogTxResponseFormatter = formatter.transactionResponse.bind(formatter)
  formatter.transactionResponse = (tx: any) => {
    const parsed = ogTxResponseFormatter(tx)
    parsed.txType = tx.txType
    parsed.queueOrigin = tx.queueOrigin
    parsed.rawTransaction = tx.rawTransaction
    parsed.l1TxOrigin = tx.l1TxOrigin
    parsed.l1BlockNumber = tx.l1BlockNumber
      ? parseInt(tx.l1BlockNumber, 16)
      : tx.l1BlockNumbers
    return parsed
  }

  // Modify the receipt formatter to include all the L2 fields.
  const ogReceiptFormatter = formatter.receipt.bind(formatter)
  formatter.receipt = (receipt: any) => {
    const parsed = ogReceiptFormatter(receipt)
    parsed.l1GasPrice = toBigNumber(receipt.l1GasPrice)
    parsed.l1GasUsed = toBigNumber(receipt.l1GasUsed)
    parsed.l1Fee = toBigNumber(receipt.l1Fee)
    parsed.l1FeeScalar = parseFloat(receipt.l1FeeScalar)
    return parsed
  }

  // Connect extra functions.
  l2Provider.getL1GasPrice = async () => {
    return getL1GasPrice(l2Provider)
  }
  l2Provider.estimateL1Gas = async (tx: TransactionRequest) => {
    return estimateL1Gas(l2Provider, tx)
  }
  l2Provider.estimateL1GasCost = async (tx: TransactionRequest) => {
    return estimateL1GasCost(l2Provider, tx)
  }
  l2Provider.estimateL2GasCost = async (tx: TransactionRequest) => {
    return estimateL2GasCost(l2Provider, tx)
  }
  l2Provider.estimateTotalGasCost = async (tx: TransactionRequest) => {
    return estimateTotalGasCost(l2Provider, tx)
  }

  l2Provider._isL2Provider = true

  return l2Provider
}
