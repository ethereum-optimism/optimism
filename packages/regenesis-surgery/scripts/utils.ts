/* eslint @typescript-eslint/no-var-requires: "off" */
import { ethers } from 'ethers'
import { abi as UNISWAP_FACTORY_ABI } from '@uniswap/v3-core/artifacts/contracts/UniswapV3Factory.sol/UniswapV3Factory.json'
import { Interface } from '@ethersproject/abi'
import { parseChunked } from '@discoveryjs/json-ext'
import { createReadStream } from 'fs'
import * as fs from 'fs'
import byline from 'byline'
import * as dotenv from 'dotenv'
import * as assert from 'assert'
import { reqenv, getenv, remove0x } from '@eth-optimism/core-utils'
import {
  Account,
  EtherscanContract,
  StateDump,
  SurgeryConfigs,
  GenesisFile,
} from './types'
import { UNISWAP_V3_FACTORY_ADDRESS } from './constants'

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

export const replaceWETH = (code: string): string => {
  return code.replace(
    /c02aaa39b223fe8d0a0e5c4f27ead9083c756cc2/g,
    '4200000000000000000000000000000000000006'
  )
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

// ERC20 interface
const iface = new Interface([
  'function balanceOf(address)',
  'function name()',
  'function symbol()',
  'function decimals()',
  'function totalSupply()',
  'function transfer(address,uint256)',
])

// PUSH4 should prefix any 4 byte selector
const PUSH4 = 0x63
const erc20Sighashes = new Set()
// Build the set of erc20 4 byte selectors
for (const fn of Object.keys(iface.functions)) {
  const sighash = iface.getSighash(fn)
  erc20Sighashes.add(sighash)
}

export const isBytecodeERC20 = (bytecode: string): boolean => {
  if (bytecode === '0x' || bytecode === undefined) {
    return false
  }
  const seen = new Set()
  const buf = Buffer.from(remove0x(bytecode), 'hex')
  for (const [i, byte] of buf.entries()) {
    // Track all of the observed 4 byte selectors that follow a PUSH4
    // and are also present in the set of erc20Sighashes
    if (byte === PUSH4) {
      const sighash = '0x' + buf.slice(i + 1, i + 5).toString('hex')
      if (erc20Sighashes.has(sighash)) {
        seen.add(sighash)
      }
    }
  }

  // create a set that contains those elements of set
  // erc20Sighashes that are not in set seen
  const elements = [...erc20Sighashes].filter((x) => !seen.has(x))
  return !elements.length
}

export const getUniswapV3Factory = (signerOrProvider: any): ethers.Contract => {
  return new ethers.Contract(
    UNISWAP_V3_FACTORY_ADDRESS,
    UNISWAP_FACTORY_ABI,
    signerOrProvider
  )
}

export const loadConfigs = (): SurgeryConfigs => {
  dotenv.config()
  const stateDumpFilePath = reqenv('REGEN__STATE_DUMP_FILE')
  const etherscanFilePath = reqenv('REGEN__ETHERSCAN_FILE')
  const genesisFilePath = reqenv('REGEN__GENESIS_FILE')
  const outputFilePath = reqenv('REGEN__OUTPUT_FILE')
  const l2ProviderUrl = reqenv('REGEN__L2_PROVIDER_URL')
  const ropstenProviderUrl = reqenv('REGEN__ROPSTEN_PROVIDER_URL')
  const ropstenPrivateKey = reqenv('REGEN__ROPSTEN_PRIVATE_KEY')
  const ethProviderUrl = reqenv('REGEN__ETH_PROVIDER_URL')
  const stateDumpHeight = parseInt(reqenv('REGEN__STATE_DUMP_HEIGHT'), 10)
  const startIndex = parseInt(getenv('REGEN__START_INDEX', '0'), 10)
  const endIndex = parseInt(getenv('REGEN__END_INDEX', '0'), 10) || Infinity

  return {
    stateDumpFilePath,
    etherscanFilePath,
    genesisFilePath,
    outputFilePath,
    l2ProviderUrl,
    ropstenProviderUrl,
    ropstenPrivateKey,
    ethProviderUrl,
    stateDumpHeight,
    startIndex,
    endIndex,
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

export const readEtherscanFile = async (
  etherscanpath: string
): Promise<EtherscanContract[]> => {
  return parseChunked(createReadStream(etherscanpath))
}

export const readGenesisFile = async (
  genesispath: string
): Promise<GenesisFile> => {
  return JSON.parse(fs.readFileSync(genesispath, 'utf8'))
}

export const readGenesisStateDump = async (
  genesispath: string
): Promise<StateDump> => {
  const genesis = await readGenesisFile(genesispath)
  const genesisDump: StateDump = []
  for (const [address, account] of Object.entries(genesis.alloc)) {
    genesisDump.push({
      address,
      ...account,
    })
  }
  return genesisDump
}

export const checkStateDump = (dump: StateDump) => {
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

    if (account.codeHash) {
      assert.equal(
        account.codeHash.toLowerCase(),
        account.codeHash,
        `unexpected upper case character in state dump codeHash: ${account.codeHash}`
      )
    }

    if (account.root) {
      assert.equal(
        account.root.toLowerCase(),
        account.root,
        `unexpected upper case character in state dump root: ${account.root}`
      )
    }

    if (account.code) {
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
