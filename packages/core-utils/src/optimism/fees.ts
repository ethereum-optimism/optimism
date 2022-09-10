/**
 * Fee related serialization and deserialization
 */

import { BigNumber } from '@ethersproject/bignumber'

import { remove0x } from '../common'

export const txDataZeroGas = 4
export const txDataNonZeroGasEIP2028 = 16
const big10 = BigNumber.from(10)

export const scaleDecimals = (
  value: number | BigNumber,
  decimals: number | BigNumber
): BigNumber => {
  value = BigNumber.from(value)
  decimals = BigNumber.from(decimals)
  // 10**decimals
  const divisor = big10.pow(decimals)
  return value.div(divisor)
}

// data is the RLP encoded unsigned transaction
export const calculateL1GasUsed = (
  data: string | Buffer,
  overhead: number | BigNumber
): BigNumber => {
  const [zeroes, ones] = zeroesAndOnes(data)
  const zeroesCost = zeroes * txDataZeroGas
  // Add a buffer to account for the signature
  const onesCost = (ones + 68) * txDataNonZeroGasEIP2028
  return BigNumber.from(onesCost).add(zeroesCost).add(overhead)
}

export const calculateL1Fee = (
  data: string | Buffer,
  overhead: number | BigNumber,
  l1GasPrice: number | BigNumber,
  scalar: number | BigNumber,
  decimals: number | BigNumber
): BigNumber => {
  const l1GasUsed = calculateL1GasUsed(data, overhead)
  const l1Fee = l1GasUsed.mul(l1GasPrice)
  const scaled = l1Fee.mul(scalar)
  const result = scaleDecimals(scaled, decimals)
  return result
}

// Count the number of zero bytes and non zero bytes in a buffer
export const zeroesAndOnes = (data: Buffer | string): Array<number> => {
  if (typeof data === 'string') {
    data = Buffer.from(remove0x(data), 'hex')
  }
  let zeros = 0
  let ones = 0
  for (const byte of data) {
    if (byte === 0) {
      zeros++
    } else {
      ones++
    }
  }
  return [zeros, ones]
}

/**
 * Computes the L1 calldata cost of bytes based
 * on the London hardfork.
 *
 * @param data {Buffer|string} Bytes
 * @returns {BigNumber} Gas consumed by the bytes
 */
export const calldataCost = (data: Buffer | string): BigNumber => {
  const [zeros, ones] = zeroesAndOnes(data)
  const zeroCost = BigNumber.from(zeros).mul(txDataZeroGas)
  const nonZeroCost = BigNumber.from(ones).mul(txDataNonZeroGasEIP2028)
  return zeroCost.add(nonZeroCost)
}
