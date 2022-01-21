import { ethers } from 'hardhat'

import { NON_ZERO_ADDRESS } from '../constants'

export const DUMMY_BATCH_HEADERS = [
  {
    batchIndex: 0,
    batchRoot: ethers.constants.HashZero,
    batchSize: 0,
    prevTotalElements: 0,
    extraData: ethers.utils.defaultAbiCoder.encode(
      ['uint256', 'address'],
      [ethers.constants.HashZero, NON_ZERO_ADDRESS]
    ),
  },
  {
    batchIndex: 1,
    batchRoot: ethers.constants.HashZero,
    batchSize: 0,
    prevTotalElements: 0,
    extraData: ethers.utils.defaultAbiCoder.encode(
      ['uint256', 'address'],
      [ethers.constants.HashZero, NON_ZERO_ADDRESS]
    ),
  },
]

export const DUMMY_BATCH_PROOFS = [
  {
    index: 0,
    siblings: [ethers.constants.HashZero],
  },
  {
    index: 1,
    siblings: [ethers.constants.HashZero],
  },
]
