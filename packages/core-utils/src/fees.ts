/**
 * Fee related serialization and deserialization
 */

import { BigNumber } from 'ethers'
import { remove0x } from './common'

const hundredMillion = BigNumber.from(100_000_000)
const txDataZeroGas = 4
const txDataNonZeroGasEIP2028 = 16
const overhead = 4200

function encode(
  data: Buffer | string,
  l1GasPrice: BigNumber,
  l2GasLimit: BigNumber,
  l2GasPrice: BigNumber
): BigNumber {
  if (!verifyL2GasPrice(l2GasPrice)) {
    throw new Error(`Invalid L2 Gas Price: ${l2GasPrice.toString()}`)
  }
  if (!verifyL1GasPrice(l1GasPrice)) {
    throw new Error(`Invalid L1 Gas Price: ${l1GasPrice.toString()}`)
  }
  const l1GasLimit = calculateL1GasLimit(data)
  const l1Fee = l1GasPrice.mul(l1GasLimit)
  const l2Fee = l2GasLimit.mul(l2GasPrice)
  return l1Fee.add(l2Fee)
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
  // If the gas price is not equal to 0 and the gas price mod
  // one hundred million is not one
  if (!gasPrice.eq(0) && !gasPrice.mod(hundredMillion).eq(1)) {
    return false
  }
  if (gasPrice.eq(0)) {
    return false
  }
  return true
}

export function verifyL1GasPrice(gasPrice: BigNumber | number): boolean {
  if (typeof gasPrice === 'number') {
    gasPrice = BigNumber.from(gasPrice)
  }
  return gasPrice.mod(hundredMillion).eq(0)
}

export function calculateL1GasLimit(data: string | Buffer): number {
  const [zeroes, ones] = zeroesAndOnes(data)
  const zeroesCost = zeroes * txDataZeroGas
  const onesCost = ones * txDataNonZeroGasEIP2028
  const gasLimit = zeroesCost + onesCost + overhead
  return gasLimit
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

export function roundL1GasPrice(gasPrice: BigNumber | number): BigNumber {
  if (typeof gasPrice === 'number') {
    gasPrice = BigNumber.from(gasPrice)
  }
  return ceilModOneHundredMillion(gasPrice)
}

function ceilModOneHundredMillion(num: BigNumber): BigNumber {
  if (num.mod(hundredMillion).eq(0)) {
    return num
  }
  const sum = num.add(hundredMillion)
  const mod = num.mod(hundredMillion)
  return sum.sub(mod)
}

export function roundL2GasPrice(gasPrice: BigNumber | number): BigNumber {
  if (typeof gasPrice === 'number') {
    gasPrice = BigNumber.from(gasPrice)
  }
  if (gasPrice.eq(0)) {
    return BigNumber.from(1)
  }
  if (gasPrice.eq(1)) {
    return hundredMillion.add(1)
  }
  const gp = gasPrice.sub(1)
  const mod = ceilModOneHundredMillion(gp)
  return mod.add(1)
}
