/* External Imports */
import { keccak256 } from 'ethers/lib/utils'

export const DUMMY_BYTECODE = '0x123412341234'
export const DUMMY_BYTECODE_BYTELEN = 6
export const UNSAFE_BYTECODE = '0x6069606955'
export const DUMMY_BYTECODE_HASH = keccak256(DUMMY_BYTECODE)
