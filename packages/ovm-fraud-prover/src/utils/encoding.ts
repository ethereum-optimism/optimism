/* External Imports */
import * as rlp from 'rlp'
import { BigNumber } from '@ethersproject/bignumber'

/* Internal Imports */
import { AccountState } from '../interfaces'
import { NULL_BYTES32 } from './constants'

/**
 * Utility; converts a buffer or string into a '0x'-prefixed string.
 * @param buf Element to convert.
 * @returns Converted element.
 */
export const toHexString = (buf: Buffer | string | null): string => {
  return '0x' + toHexBuffer(buf).toString('hex')
}

/**
 * Utility; converts a buffer or a string to a non '0x'-prefixed buffer.
 * @param buf Element to convert.
 * @returns Converted element.
 */
export const toHexBuffer = (buf: Buffer | string): Buffer => {
  if (typeof buf === 'string' && buf.startsWith('0x')) {
    return Buffer.from(buf.slice(2), 'hex')
  }

  return Buffer.from(buf)
}

/**
 * Utility; RLP-encodes an account state.
 * @param state State to encode.
 * @returns Encoded account state as a buffer.
 */
export const encodeAccountState = (state: Partial<AccountState>): Buffer => {
  return rlp.encode([
    state.nonce || 0,
    state.balance.toHexString() || 0,
    state.storageRoot || NULL_BYTES32,
    state.codeHash || NULL_BYTES32,
  ])
}

/**
 * Utility; RLP-decodes an account state.
 * @param state RLP-encoded account state.
 * @returns Decoded account state.
 */
export const decodeAccountState = (state: Buffer): AccountState => {
  const decoded = rlp.decode(state) as any
  return {
    nonce: decoded[0].length ? parseInt(toHexString(decoded[0]), 16) : 0,
    balance: decoded[1].length ? BigNumber.from(decoded[1]) : BigNumber.from(0),
    storageRoot: decoded[2].length ? toHexString(decoded[2]) : null,
    codeHash: decoded[3].length ? toHexString(decoded[3]) : null,
  }
}