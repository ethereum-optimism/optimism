import { ethers } from 'ethers'
import { abi as UNISWAP_FACTORY_ABI } from '@uniswap/v3-core/artifacts/contracts/UniswapV3Factory.sol/UniswapV3Factory.json'
import { UNISWAP_V3_FACTORY_ADDRESS } from './constants'
import { Account, StateDump } from './types'

export const findAccount = (dump: StateDump, address: string): Account => {
  return dump.find((acc) => {
    return hexStringEqual(acc.address, address)
  })
}

export const hexStringIncludes = (a: string, b: string): boolean => {
  if (!ethers.utils.isHexString(a)) {
    throw new Error(`not a hex string: ${a}`)
  }
  if (!ethers.utils.isHexString(b)) {
    throw new Error(`not a hex string: ${b}`)
  }

  return a.slice(2).toLowerCase().includes(b.slice(2).toLowerCase())
}

export const hexStringEqual = (a: string, b: string): boolean => {
  if (!ethers.utils.isHexString(a)) {
    throw new Error(`not a hex string: ${a}`)
  }
  if (!ethers.utils.isHexString(b)) {
    throw new Error(`not a hex string: ${b}`)
  }

  return a.toLowerCase() === b.toLowerCase()
}

/**
 * Left-pads a hex string with zeroes to 32 bytes.
 *
 * @param val Value to hex pad to 32 bytes.
 * @returns Value padded to 32 bytes.
 */
export const toHex32 = (val: string | number | ethers.BigNumber) => {
  return ethers.utils.hexZeroPad(ethers.BigNumber.from(val).toHexString(), 32)
}

export const transferStorageSlot = (opts: {
  account: Account
  oldSlot: string | number
  newSlot: string | number
  newValue?: string
}): void => {
  if (opts.account.storage === undefined) {
    throw new Error(`account has no storage: ${opts.account.address}`)
  }

  if (typeof opts.oldSlot !== 'string') {
    opts.oldSlot = toHex32(opts.oldSlot)
  }

  if (typeof opts.newSlot !== 'string') {
    opts.newSlot = toHex32(opts.newSlot)
  }

  const oldSlotVal = opts.account.storage[opts.oldSlot]
  if (oldSlotVal === undefined) {
    throw new Error(
      `old slot not found in state dump, address=${opts.account.address}, slot=${opts.oldSlot}`
    )
  }

  if (opts.newValue === undefined) {
    opts.account.storage[opts.newSlot] = oldSlotVal
  } else {
    if (opts.newValue.startsWith('0x')) {
      opts.newValue = opts.newValue.slice(2)
    }
    opts.account.storage[opts.newSlot] = opts.newValue
  }

  delete opts.account.storage[opts.oldSlot]
}

export const getMappingKey = (keys: any[], slot: number) => {
  // TODO: assert keys.length > 0
  let key = ethers.utils.keccak256(
    ethers.utils.hexConcat([toHex32(keys[0]), toHex32(slot)])
  )
  if (keys.length > 1) {
    for (let i = 1; i < keys.length; i++) {
      key = ethers.utils.keccak256(
        ethers.utils.hexConcat([toHex32(keys[i]), key])
      )
    }
  }
  return key
}

export const getUniswapV3Factory = (signerOrProvider: any): ethers.Contract => {
  return new ethers.Contract(
    UNISWAP_V3_FACTORY_ADDRESS,
    UNISWAP_FACTORY_ABI,
    signerOrProvider
  )
}

export const clone = (obj: any): any => {
  return JSON.parse(JSON.stringify(obj))
}
