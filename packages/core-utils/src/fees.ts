/**
 * Fee related serialization and deserialization
 */

import { BigNumber } from 'ethers'
import { remove0x } from './common'

const hundredMillion = BigNumber.from(100_000_000)
const feeScalar = 1000
export const TxGasPrice = BigNumber.from(feeScalar + feeScalar / 2)
const txDataZeroGas = 4
const txDataNonZeroGasEIP2028 = 16
const overhead = 4200 + 200 * txDataNonZeroGasEIP2028

export interface EncodableL2GasLimit {
  data: Buffer | string
  l1GasPrice: BigNumber | number
  l2GasLimit: BigNumber | number
  l2GasPrice: BigNumber | number
}

function encode(input: EncodableL2GasLimit): BigNumber {
  const { data } = input
  let { l1GasPrice, l2GasLimit, l2GasPrice } = input
  if (typeof l1GasPrice === 'number') {
    l1GasPrice = BigNumber.from(l1GasPrice)
  }
  if (typeof l2GasLimit === 'number') {
    l2GasLimit = BigNumber.from(l2GasLimit)
  }
  if (typeof l2GasPrice === 'number') {
    l2GasPrice = BigNumber.from(l2GasPrice)
  }
  const l1GasLimit = calculateL1GasLimit(data)
  const l1Fee = l1GasLimit.mul(l1GasPrice)
  const l2Fee = l2GasLimit.mul(l2GasPrice)
  const sum = l1Fee.add(l2Fee)
  const scaled = sum.div(feeScalar)
  const remainder = scaled.mod(hundredMillion)
  const scaledSum = scaled.add(hundredMillion)
  const rounded = scaledSum.sub(remainder)
  return rounded.add(l2GasLimit)
}

function decode(fee: BigNumber | number): BigNumber {
  if (typeof fee === 'number') {
    fee = BigNumber.from(fee)
  }
  return fee.mod(hundredMillion)
}

export const TxGasLimit = {
  encode,
  decode,
}

export function calculateL1GasLimit(data: string | Buffer): BigNumber {
  const [zeroes, ones] = zeroesAndOnes(data)
  const zeroesCost = zeroes * txDataZeroGas
  const onesCost = ones * txDataNonZeroGasEIP2028
  const gasLimit = zeroesCost + onesCost + overhead
  return BigNumber.from(gasLimit)
}

export function zeroesAndOnes(data: Buffer | string): Array<number> {
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
