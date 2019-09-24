/* Imports */
import { keccak256, abi, hexStrToBuf, bufToHexString } from '@pigi/core'
import { RollupTransition, TransferTransition } from '@pigi/wallet'

/* Export files */
export * from './RollupBlock'

/**********************************
 * Byte String Generation Helpers *
 *********************************/

// Create a byte string of some length in bytes. It repeats the value provided until the
// string hits that length
export function makeRepeatedBytes(value: string, length: number): string {
  const repeated = value.repeat((length * 2) / value.length + 1)
  const sliced = repeated.slice(0, length * 2)
  return '0x' + sliced
}

// Make padded bytes. Bytes are right padded.
export function makePaddedBytes(value: string, length: number): string {
  if (value.length > length * 2) {
    throw new Error('Value too large to fit in ' + length + ' byte string')
  }
  const targetLength = length * 2
  while (value.length < (targetLength || 2)) {
    value = value + '0'
  }
  return '0x' + value
}

// Make a padded uint. Uints are left padded.
export function makePaddedUint(value: string, length: number): string {
  if (value.length > length * 2) {
    throw new Error('Value too large to fit in ' + length + ' byte string')
  }
  const targetLength = length * 2
  while (value.length < (targetLength || 2)) {
    value = '0' + value
  }
  return '0x' + value
}

/*******************************
 * Transition Encoding Helpers *
 ******************************/
export type Transition = string

// Generates some number of dummy transitions
export function generateNTransitions(
  numTransitions: number
): RollupTransition[] {
  const transitions = []
  for (let i = 0; i < numTransitions; i++) {
    const transfer: TransferTransition = {
      stateRoot: getStateRoot('ab'),
      senderSlotIndex: 2,
      recipientSlotIndex: 2,
      tokenType: 0,
      amount: 1,
      signature: getSignature('01'),
    }
    transitions.push(transfer)
  }
  return transitions
}

/****************
 * Misc Helpers *
 ***************/

export const ZERO_BYTES32 = makeRepeatedBytes('0', 32)
export const ZERO_ADDRESS = makeRepeatedBytes('0', 20)
export const ZERO_UINT32 = makeRepeatedBytes('0', 4)
export const ZERO_SIGNATURE = makeRepeatedBytes('0', 65)

/* Extra Helpers */
export const STORAGE_TREE_HEIGHT = 5
export const AMOUNT_BYTES = 5
export const getSlot = (storageSlot: string) =>
  makePaddedUint(storageSlot, STORAGE_TREE_HEIGHT)
export const getAmount = (amount: string) =>
  makePaddedUint(amount, AMOUNT_BYTES)
export const getAddress = (address: string) => makeRepeatedBytes(address, 20)
export const getSignature = (sig: string) => makeRepeatedBytes(sig, 65)
export const getStateRoot = (bytes: string) => makeRepeatedBytes(bytes, 32)
export const getBytes32 = (bytes: string) => makeRepeatedBytes(bytes, 32)

export const UNISWAP_ADDRESS = getAddress('00')
export const UNISWAP_STORAGE_SLOT = 0
