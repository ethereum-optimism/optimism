/**
 * Fee related serialization and deserialization
 */

import { BigNumber } from 'ethers'
import { remove0x } from './common'

const hundredMillion = BigNumber.from(100_000_000)
const feeScalar = 10_000_000
export const TxGasPrice = BigNumber.from(feeScalar + feeScalar / 2)
const txDataZeroGas = 4
const txDataNonZeroGasEIP2028 = 16
const overhead = 4200 + 200 * txDataNonZeroGasEIP2028
const tenThousand = BigNumber.from(10_000)

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
  const roundedL2GasLimit = ceilmod(l2GasLimit, tenThousand)
  const l1Fee = l1GasLimit.mul(l1GasPrice)
  const l2Fee = roundedL2GasLimit.mul(l2GasPrice)
  const sum = l1Fee.add(l2Fee)
  const scaled = sum.div(feeScalar)
  const rounded = ceilmod(scaled, tenThousand)
  const roundedScaledL2GasLimit = roundedL2GasLimit.div(tenThousand)
  return rounded.add(roundedScaledL2GasLimit)
}

function decode(fee: BigNumber | number): BigNumber {
  if (typeof fee === 'number') {
    fee = BigNumber.from(fee)
  }
  const scaled = fee.mod(tenThousand)
  return scaled.mul(tenThousand)
}

export const TxGasLimit = {
  encode,
  decode,
}

export function ceilmod(a: BigNumber | number, b: BigNumber | number) {
  if (typeof a === 'number') {
    a = BigNumber.from(a)
  }
  if (typeof b === 'number') {
    b = BigNumber.from(b)
  }
  const remainder = a.mod(b)
  if (remainder.eq(0)) {
    return a
  }
  const sum = a.add(b)
  const rounded = sum.sub(remainder)
  return rounded
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
