/* External Imports */
import { ethers } from 'ethers'

export const DUMMY_BYTES32: string[] = Array.from(
  {
    length: 10,
  },
  (_, i) => {
    return ethers.utils.keccak256(`0x0${i}`)
  }
)
