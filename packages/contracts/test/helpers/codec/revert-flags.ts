import { ethers } from '@nomiclabs/buidler'

export const encodeRevertData = (
  flag: number,
  data: string = '0x',
  nuisanceGasLeft: number = 0,
  ovmGasRefund: number = 0
): string => {
  return ethers.utils.defaultAbiCoder.encode(
    ['uint256','uint256','uint256','bytes'],
    [flag, nuisanceGasLeft, ovmGasRefund, data]
  )
}

export const REVERT_FLAGS = {
  DID_NOT_REVERT: 0,
  OUT_OF_GAS: 1,
  INTENTIONAL_REVERT: 2,
  EXCEEDS_NUISANCE_GAS: 3,
  INVALID_STATE_ACCESS: 4,
  UNSAFE_BYTECODE: 5
}
