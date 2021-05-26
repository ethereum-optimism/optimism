import { ethers } from 'ethers'

import { OvmTransaction } from '../types'
import { toUint256, toUint8 } from './hex-utils'
import { toHexString, fromHexString } from '@eth-optimism/core-utils'

export const encodeOvmTransaction = (transaction: OvmTransaction): string => {
  return toHexString(
    Buffer.concat([
      fromHexString(toUint256(transaction.timestamp)),
      fromHexString(toUint256(transaction.blockNumber)),
      fromHexString(toUint8(transaction.l1QueueOrigin)),
      fromHexString(transaction.l1TxOrigin),
      fromHexString(transaction.entrypoint),
      fromHexString(toUint256(transaction.gasLimit)),
      fromHexString(transaction.data),
    ])
  )
}

export const hashOvmTransaction = (transaction: OvmTransaction): string => {
  return ethers.utils.keccak256(encodeOvmTransaction(transaction))
}
