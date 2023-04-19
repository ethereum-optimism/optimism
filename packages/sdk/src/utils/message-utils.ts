import { hashWithdrawal } from '@eth-optimism/core-utils'
import { BigNumber, utils } from 'ethers'

import { LowLevelMessage } from '../interfaces'

const { hexDataLength } = utils

/**
 * Utility for hashing a LowLevelMessage object.
 *
 * @param message LowLevelMessage object to hash.
 * @returns Hash of the given LowLevelMessage.
 */
export const hashLowLevelMessage = (message: LowLevelMessage): string => {
  return hashWithdrawal(
    message.messageNonce,
    message.sender,
    message.target,
    message.value,
    message.minGasLimit,
    message.message
  )
}

/**
 * Compute the min gas limit for a migrated withdrawal.
 */
export const migratedWithdrawalGasLimit = (data: string): BigNumber => {
  // Compute the gas limit and cap at 25 million
  const dataCost = BigNumber.from(hexDataLength(data)).mul(16)
  let minGasLimit = dataCost.add(200_000)
  if (minGasLimit.gt(25_000_000)) {
    minGasLimit = BigNumber.from(25_000_000)
  }
  return minGasLimit
}
