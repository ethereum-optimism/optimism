/* Imports: External */
import * as fs from 'fs'
import byline from 'byline'
import { ethers } from 'ethers'

/* Imports: Internal */
import { StateDump } from './types'

/**
 * Left-pads a hex string with zeroes to 32 bytes.
 *
 * @param val Value to hex pad to 32 bytes.
 * @returns Value padded to 32 bytes.
 */
export const toHex32 = (val: string | number | ethers.BigNumber) => {
  return ethers.utils.hexZeroPad(ethers.BigNumber.from(val).toHexString(), 32)
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

/**
 * Loads a variable from the environment and throws if the variable is not defined.
 *
 * @param name Name of the variable to load.
 * @returns Value of the variable as a string.
 */
export const reqenv = (name: string): any => {
  const value = process.env[name]
  if (value === undefined) {
    throw new Error(`missing env var ${name}`)
  }
  return value
}

/**
 * Reads the state dump file into an object. Required because the dumps get quite large.
 * JavaScript throws an error when trying to load large JSON files (>512mb) directly via
 * fs.readFileSync. Need a streaming approach instead.
 *
 * @param dumppath Path to the state dump file.
 * @returns Parsed state dump object.
 */
export const readDumpFile = async (dumppath: string): Promise<StateDump> => {
  return new Promise<StateDump>((resolve) => {
    const dump: StateDump = {
      root: '',
      accounts: {},
    }

    const stream = byline(fs.createReadStream(dumppath, { encoding: 'utf8' }))

    let isFirstRow = true
    stream.on('data', (line: any) => {
      const data = JSON.parse(line)
      if (isFirstRow) {
        dump.root = data.root
        isFirstRow = false
      } else {
        const address = data.address
        delete data.address
        delete data.key
        dump.accounts[address] = data
      }
    })

    stream.on('end', () => {
      resolve(dump)
    })
  })
}

export const transferStorageSlot = (opts: {
  dump: StateDump
  address: string
  oldSlot: string
  newSlot: string
}): void => {
  const account = opts.dump.accounts[opts.address]
  if (account === undefined) {
    throw new Error(`account not found in state dump: ${opts.address}`)
  }

  if (account.storage === undefined) {
    throw new Error(`account has no storage: ${opts.address}`)
  }

  const oldSlotVal = account.storage[opts.oldSlot]
  if (oldSlotVal === undefined) {
    throw new Error(`old slot not found in state dump: ${opts.address}`)
  }

  account.storage[opts.newSlot] = oldSlotVal
  delete account.storage[opts.oldSlot]
}
