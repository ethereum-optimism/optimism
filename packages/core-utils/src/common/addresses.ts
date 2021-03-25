/* Imports: External */
import { ethers } from 'ethers'

/* Imports: Internal */
import { getRandomHexString } from './hex-strings'

/* @returns a random Ethereum address as a string of 40 hex characters, normalized as a checksum address. */
export const getRandomAddress = (): string => {
  return ethers.utils.getAddress(getRandomHexString(20))
}
