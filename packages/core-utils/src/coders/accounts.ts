/* External Imports */
import { BigNumber, constants } from 'ethers'
import { parse, Transaction } from '@ethersproject/transactions'

export const LibEIP155TxStruct = (tx: Transaction | string): Array<any> => {
  if (typeof tx === 'string') {
    tx = parse(tx)
  }
  if (tx.chainId === 0) {
    throw new Error('Not an EIP155 compatible transaction')
  }
  const values = [
    tx.nonce,
    tx.gasPrice,
    tx.gasLimit,
    tx.to ? tx.to : constants.AddressZero,
    tx.value,
    tx.data,
    tx.v % 256,
    tx.r,
    tx.s,
    tx.chainId,
    parseV(tx.v, tx.chainId),
    tx.to === null,
  ]
  return values
}

const parseV = (orig: number, chainId: number): number => {
  if ([0, 1].includes(orig)) {
    return orig
  }
  return orig - 2 * chainId - 35
}
