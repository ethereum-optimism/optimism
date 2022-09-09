import { BigNumber } from '@ethersproject/bignumber'
import { getAddress } from '@ethersproject/address'

import { remove0x, add0x } from './hex-strings'

/**
 * Converts an ethers BigNumber into an equivalent Ethereum address representation.
 *
 * @param bn BigNumber to convert to an address.
 * @return BigNumber converted to an address, represented as a hex string.
 */
export const bnToAddress = (bn: BigNumber | number): string => {
  // Coerce numbers into a BigNumber.
  bn = BigNumber.from(bn)

  // Negative numbers are converted to addresses by adding MAX_ADDRESS + 1.
  // TODO: Explain this in more detail, it's basically just matching the behavior of doing
  // addr(uint256(addr) - some_number) in Solidity where some_number > uint256(addr).
  if (bn.isNegative()) {
    bn = BigNumber.from('0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF')
      .add(bn)
      .add(1)
  }

  // Convert to a hex string
  let addr = bn.toHexString()
  // Remove leading 0x so we can mutate the address a bit
  addr = remove0x(addr)
  // Make sure it's 40 characters (= 20 bytes)
  addr = addr.padStart(40, '0')
  // Only take the last 40 characters (= 20 bytes)
  addr = addr.slice(addr.length - 40, addr.length)
  // Add 0x again
  addr = add0x(addr)
  // Convert into a checksummed address
  addr = getAddress(addr)

  return addr
}
