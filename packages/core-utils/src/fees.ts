/**
 * Fee related serialization and deserialization
 */

import { BigNumber } from 'ethers'
import { remove0x } from './common'

const hundredMillion = BigNumber.from(100_000_000)
const hundredBillion = BigNumber.from(100_000_000_000)
const feeScalar = BigNumber.from(1000)
const txDataZeroGas = 4
const txDataNonZeroGasEIP2028 = 16
const overhead = 4200

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

  if (!verifyGasPrice(l2GasPrice)) {
    throw new Error(`Invalid L2 Gas Price: ${l2GasPrice.toString()}`)
  }
  if (!verifyGasPrice(l1GasPrice)) {
    throw new Error(`Invalid L1 Gas Price: ${l1GasPrice.toString()}`)
  }
  const l1GasLimit = calculateL1GasLimit(data)
  const l1Fee = l1GasLimit.mul(l1GasPrice)
  const l2Fee = l2GasLimit.mul(l2GasPrice)
  const sum = l1Fee.add(l2Fee)
  const scaled = sum.div(feeScalar)
  return scaled.add(l2GasLimit)
}

function verifyGasPrice(gasPrice: BigNumber | number): boolean {
  if (typeof gasPrice === 'number') {
    gasPrice = BigNumber.from(gasPrice)
  }
  if (gasPrice.eq(0)) {
    return true
  }
  if (gasPrice.lt(hundredBillion)) {
    return false
  }
  return gasPrice.mod(hundredMillion).eq(0)
}

function decode(fee: BigNumber | number): BigNumber {
  if (typeof fee === 'number') {
    fee = BigNumber.from(fee)
  }
  return fee.mod(hundredMillion)
}

export const L2GasLimit = {
  encode,
  decode,
}

export function verifyL2GasPrice(gasPrice: BigNumber | number): boolean {
  if (typeof gasPrice === 'number') {
    gasPrice = BigNumber.from(gasPrice)
  }
  return gasPrice.mod(hundredMillion).eq(0)
}

export function verifyL1GasPrice(gasPrice: BigNumber | number): boolean {
  if (typeof gasPrice === 'number') {
    gasPrice = BigNumber.from(gasPrice)
  }
  return gasPrice.mod(hundredMillion).eq(0)
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

export function roundGasPrice(gasPrice: BigNumber | number): BigNumber {
  if (typeof gasPrice === 'number') {
    gasPrice = BigNumber.from(gasPrice)
  }
  if (gasPrice.eq(0)) {
    return gasPrice
  }
  if (gasPrice.mod(hundredBillion).eq(0)) {
    return gasPrice
  }
  const sum = gasPrice.add(hundredBillion)
  const mod = gasPrice.mod(hundredBillion)
  return sum.sub(mod)
}
