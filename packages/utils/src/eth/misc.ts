/* External Imports */
import { ethers } from 'ethers'

/* Internal Imports */
import { add0x } from '../misc'

export const abi = new ethers.utils.AbiCoder()

/**
 * Computes the keccak256 hash of a value.
 * @param value Value to hash
 * @returns the hash of the value.
 */
export const keccak256 = (value: string): string => {
  const preimage = add0x(value.replace(/0x/g, ''))
  return ethers.utils.keccak256(preimage)
}
