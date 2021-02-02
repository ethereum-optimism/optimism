import { NULL_BYTES32, NON_ZERO_ADDRESS } from '../constants'
import { ethers } from 'hardhat'

export const DUMMY_BATCH_HEADERS = [
  {
    batchIndex: 0,
    batchRoot: NULL_BYTES32,
    batchSize: 0,
    prevTotalElements: 0,
    extraData: ethers.utils.defaultAbiCoder.encode(
      ['uint256', 'address'],
      [NULL_BYTES32, NON_ZERO_ADDRESS]
    ),
  },
  {
    batchIndex: 1,
    batchRoot: NULL_BYTES32,
    batchSize: 0,
    prevTotalElements: 0,
    extraData: ethers.utils.defaultAbiCoder.encode(
      ['uint256', 'address'],
      [NULL_BYTES32, NON_ZERO_ADDRESS]
    ),
  },
]

export const DUMMY_BATCH_PROOFS = [
  {
    index: 0,
    siblings: [NULL_BYTES32],
  },
  {
    index: 1,
    siblings: [NULL_BYTES32],
  },
]
