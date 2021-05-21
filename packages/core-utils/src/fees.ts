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

export const fees = {
  encode,
  decode,
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
  if (gasPrice.eq(0)) {
    return gasPrice
  }
  if (gasPrice.eq(1)) {
    return hundredMillion
  }
  if (gasPrice.mod(hundredMillion).lt(2)) {
    gasPrice = gasPrice.add(hundredMillion).sub(2)
  } else {
    gasPrice = gasPrice.add(hundredMillion)
  }
  return gasPrice.sub(gasPrice.mod(hundredMillion))
}

export function roundL2GasPrice(gasPrice: BigNumber): BigNumber {
  if (typeof gasPrice === 'number') {
    gasPrice = BigNumber.from(gasPrice)
  }
  if (gasPrice.eq(0)) {
    return BigNumber.from(1)
  }
  if (gasPrice.eq(1)) {
    return hundredMillion.add(1)
  }
  if (gasPrice.mod(hundredMillion).lt(2)) {
    gasPrice = gasPrice.add(hundredMillion).sub(2)
  } else {
    gasPrice = gasPrice.add(hundredMillion)
  }
  return gasPrice.sub(gasPrice.mod(hundredMillion).add(1))
}
