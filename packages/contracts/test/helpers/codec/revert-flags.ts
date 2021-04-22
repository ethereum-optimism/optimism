/* External Imports */
import { ethers } from 'hardhat'

export const encodeRevertData = (
  flag: number,
  data: string = '0x',
  nuisanceGasLeft: number = 0,
  ovmGasRefund: number = 0
): string => {
  const abiEncoded: string = ethers.utils.defaultAbiCoder.encode(
    ['uint256', 'uint256', 'uint256', 'bytes'],
    [flag, nuisanceGasLeft, ovmGasRefund, data]
  )
  return abiEncoded
}

export const decodeRevertData = (revertData: string): any => {
  const decoded = ethers.utils.defaultAbiCoder.decode(
    ['uint256', 'uint256', 'uint256', 'bytes'],
    revertData
  )

  return (
    '[revertFlag:' +
    Object.keys(REVERT_FLAGS)[decoded[0]] +
    ', nuisanceGasLeft:' +
    decoded[1] +
    ', ovmGasRefund: ' +
    decoded[2] +
    ', data: ' +
    decoded[3] +
    ']'
  )
}

export const REVERT_FLAGS = {
  OUT_OF_GAS: 0,
  INTENTIONAL_REVERT: 1,
  EXCEEDS_NUISANCE_GAS: 2,
  INVALID_STATE_ACCESS: 3,
  UNSAFE_BYTECODE: 4,
  CREATE_COLLISION: 5,
  STATIC_VIOLATION: 6,
  CREATOR_NOT_ALLOWED: 7,
  CALLER_NOT_ALLOWED: 8,
}
