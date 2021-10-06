import { ethers } from 'ethers'
import { abi as UNISWAP_FACTORY_ABI } from '@uniswap/v3-core/artifacts/contracts/UniswapV3Factory.sol/UniswapV3Factory.json'
import * as fs from 'fs'
import byline from 'byline'
import * as dotenv from 'dotenv'
import * as assert from 'assert'

import { UNISWAP_V3_FACTORY_ADDRESS } from './constants'
import { Account, StateDump, SurgeryConfigs } from './types'

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

export const loadConfigs = (): SurgeryConfigs => {
  dotenv.config()
  const stateDumpFilePath = reqenv('REGEN__STATE_DUMP_FILE')
  const etherscanFilePath = reqenv('REGEN__ETHERSCAN_FILE')
  const l2ProviderUrl = reqenv('REGEN__L2_PROVIDER_URL')
  const l2NetworkName = reqenv('REGEN__L2_NETWORK_NAME')
  const l1MainnetProviderUrl = reqenv('REGEN__L1_PROVIDER_URL')
  const l1TestnetProviderUrl = reqenv('REGEN__L1_TESTNET_PROVIDER_URL')
  const l1TestnetPrivateKey = reqenv('REGEN__L1_TESTNET_PRIVATE_KEY')

  // Input assertions
  assert.ok(
    ['mainnet', 'kovan'].includes(l2NetworkName),
    `L2_NETWORK_NAME must be one of "mainnet" or "kovan"`
  )

  return {
    stateDumpFilePath,
    etherscanFilePath,
    l2ProviderUrl,
    l2NetworkName,
    l1MainnetProviderUrl,
    l1TestnetProviderUrl,
    l1TestnetPrivateKey,
  }
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
    const dump: StateDump = []

    const stream = byline(fs.createReadStream(dumppath, { encoding: 'utf8' }))

    let isFirstRow = true
    stream.on('data', (line: any) => {
      const account = JSON.parse(line)
      if (isFirstRow) {
        isFirstRow = false
      } else {
        delete account.key
        dump.push(account)
      }
    })

    stream.on('end', () => {
      resolve(dump)
    })
  })
}

export const checkStateDump = (dump: StateDump) => {
  // Sanity check to guarantee that all addresses in dump.accounts are lower case.
  console.log(`verifying that all contract addresses are lower case`)
  for (const account of dump) {
    assert.equal(
      account.address.toLowerCase(),
      account.address,
      `unexpected upper case character in state dump address: ${account.address}`
    )

    assert.ok(
      typeof account.nonce === 'number',
      `nonce is not a number: ${account.nonce}`
    )

    assert.equal(
      account.codeHash.toLowerCase(),
      account.codeHash,
      `unexpected upper case character in state dump codeHash: ${account.codeHash}`
    )

    assert.equal(
      account.root.toLowerCase(),
      account.root,
      `unexpected upper case character in state dump root: ${account.root}`
    )

    if (account.code !== undefined) {
      assert.equal(
        account.code.toLowerCase(),
        account.code,
        `unexpected upper case character in state dump code: ${account.code}`
      )
    }

    // All accounts other than precompiles should have a balance of zero.
    if (
      !account.address.startsWith('0x00000000000000000000000000000000000000')
    ) {
      assert.equal(
        account.balance,
        '0',
        `unexpected non-zero balance in state dump address: ${account.address}`
      )
    }

    if (account.storage !== undefined) {
      for (const [storageKey, storageVal] of Object.entries(account.storage)) {
        assert.equal(
          storageKey.toLowerCase(),
          storageKey,
          `unexpected upper case character in state dump storage key: ${storageKey}`
        )
        assert.equal(
          storageVal.toLowerCase(),
          storageVal,
          `unexpected upper case character in state dump storage value: ${storageVal}`
        )
      }
    }
  }
}
