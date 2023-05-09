import { hashWithdrawal } from '@eth-optimism/core-utils'
import { BigNumber, utils, ethers } from 'ethers'

import { LowLevelMessage } from '../interfaces'

const { hexDataLength } = utils

// Constants used by `CrossDomainMessenger.baseGas`
const RELAY_CONSTANT_OVERHEAD = 200_000
const RELAY_PER_BYTE_DATA_COST = 16
const MIN_GAS_DYNAMIC_OVERHEAD_NUMERATOR = 64
const MIN_GAS_DYNAMIC_OVERHEAD_DENOMINATOR = 63
const RELAY_CALL_OVERHEAD = 40_000
const RELAY_RESERVED_GAS = 40_000
const RELAY_GAS_CHECK_BUFFER = 5_000

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
 * Utility for hashing a message hash. This computes the storage slot
 * where the message hash will be stored in state. HashZero is used
 * because the first mapping in the contract is used.
 *
 * @param messageHash Message hash to hash.
 * @returns Hash of the given message hash.
 */
export const hashMessageHash = (messageHash: string): string => {
  const data = ethers.utils.defaultAbiCoder.encode(
    ['bytes32', 'uint256'],
    [messageHash, ethers.constants.HashZero]
  )
  return ethers.utils.keccak256(data)
}

/**
 * Compute the min gas limit for a migrated withdrawal.
 */
export const migratedWithdrawalGasLimit = (
  data: string,
  chainID: number
): BigNumber => {
  // Compute the gas limit and cap at 25 million
  const dataCost = BigNumber.from(hexDataLength(data)).mul(
    RELAY_PER_BYTE_DATA_COST
  )
  let overhead: number
  if (chainID === 420) {
    overhead = 200_000
  } else {
    // Constant overhead
    overhead =
      RELAY_CONSTANT_OVERHEAD +
      // Dynamic overhead (EIP-150)
      (MIN_GAS_DYNAMIC_OVERHEAD_NUMERATOR * 1_000_000) /
        MIN_GAS_DYNAMIC_OVERHEAD_DENOMINATOR +
      // Gas reserved for the worst-case cost of 3/5 of the `CALL` opcode's dynamic gas
      // factors. (Conservative)
      RELAY_CALL_OVERHEAD +
      // Relay reserved gas (to ensure execution of `relayMessage` completes after the
      // subcontext finishes executing) (Conservative)
      RELAY_RESERVED_GAS +
      // Gas reserved for the execution between the `hasMinGas` check and the `CALL`
      // opcode. (Conservative)
      RELAY_GAS_CHECK_BUFFER
  }

  let minGasLimit = dataCost.add(overhead)
  if (minGasLimit.gt(25_000_000)) {
    minGasLimit = BigNumber.from(25_000_000)
  }
  return minGasLimit
}
