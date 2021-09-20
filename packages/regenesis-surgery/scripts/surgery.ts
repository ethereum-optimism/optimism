import * as fs from 'fs'
import * as path from 'path'
import byline from 'byline'
import { ethers } from 'ethers'
import * as dotenv from 'dotenv'
import {
  computePoolAddress,
  POOL_INIT_CODE_HASH,
  POOL_INIT_CODE_HASH_OPTIMISM,
  POOL_INIT_CODE_HASH_OPTIMISM_KOVAN,
} from '@uniswap/v3-sdk'
import { Token } from '@uniswap/sdk-core'
import { abi as UNISWAP_FACTORY_ABI } from '@uniswap/v3-core/artifacts/contracts/UniswapV3Factory.sol/UniswapV3Factory.json'
import { KECCAK256_RLP_S, KECCAK256_NULL_S } from 'ethereumjs-util'

interface ChainState {
  [address: string]: {
    balance: string
    nonce: number
    root: string
    codeHash: string
    code?: string
    storage?: {
      [key: string]: string
    }
  }
}

interface StateDump {
  root: string
  accounts: ChainState
}

const toHex32 = (val: string | number | ethers.BigNumber) => {
  return ethers.utils.hexZeroPad(ethers.BigNumber.from(val).toHexString(), 32)
}

const getMappingKey = (keys: any[], slot: number) => {
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

const requireEnv = (name: string): any => {
  const value = process.env[name]
  if (value === undefined) {
    throw new Error(`missing env var ${name}`)
  }
  return value
}

const readDumpFile = async (dumppath: string): Promise<StateDump> => {
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

const main = async () => {
  // Load required enviorment variables
  dotenv.config()
  const STATE_DUMP_FILE = requireEnv('REGEN__STATE_DUMP_FILE')
  const L2_PROVIDER_URL = requireEnv('REGEN__L2_PROVIDER_URL')
  const TESTNET_PROVIDER_URL = requireEnv('REGEN__TESTNET_PROVIDER_URL')
  const TESTNET_PRIVATE_KEY = requireEnv('REGEN__TESTNET_PRIVATE_KEY')
  const UNISWAP_FACTORY_ADDRESS = requireEnv('REGEN__UNISWAP_FACTORY_ADDRESS')
  const UNISWAP_NFPM_ADDRESS = requireEnv('REGEN__UNISWAP_NFPM_ADDRESS')

  // Load the state dump from the JSON file
  const dump: StateDump = await readDumpFile(
    path.resolve(__dirname, `../dumps/${STATE_DUMP_FILE}`)
  )

  // Set up the L2 provider.
  const l2Provider = new ethers.providers.JsonRpcProvider(L2_PROVIDER_URL)

  // Create an empty object that represents the new genesis state
  // We're going to move items from the dump into this genesis state
  const genesis: ChainState = {}

  // Sanity check to guarantee that all addresses in dump.accounts are lower case.
  console.log(`verifying that all contract addresses are lower case`)
  for (const address of Object.keys(dump.accounts)) {
    if (address !== address.toLowerCase()) {
      throw new Error(`unexpected upper case character in state dump address`)
    }
  }

  // Step 1. Transfer the state of each precompiled contract.
  console.log(`moving all precompile contract states to new genesis`)
  for (const [address, account] of Object.entries(dump.accounts)) {
    if (address.startsWith('0x00000000000000000000000000000000000000')) {
      genesis[address] = account
      delete dump.accounts[address]
    }
  }

  // Step 2. Transfer over each EOA address and turn it into a normal EOA.
  // TODO: Verify these are the correct and only EOA code hashes.
  console.log(`removing code from all EOA addresses`)
  const EOA_CODE_HASHES = [
    'a73df79c90ba2496f3440188807022bed5c7e2e826b596d22bcb4e127378835a',
    'ef2ab076db773ffc554c9f287134123439a5228e92f5b3194a28fec0a0afafe3',
  ]
  for (const [address, account] of Object.entries(dump.accounts)) {
    if (EOA_CODE_HASHES.includes(account.codeHash)) {
      genesis[address] = {
        balance: account.balance,
        nonce: account.nonce,
        root: KECCAK256_RLP_S,
        codeHash: KECCAK256_NULL_S,
      }
      delete dump.accounts[address]
    }
  }

  /* --- BEGIN UNISWAP SURGERY SECTION --- */

  // Set up the uniswap factory contract reference
  const UniswapV3Factory = new ethers.Contract(
    UNISWAP_FACTORY_ADDRESS,
    UNISWAP_FACTORY_ABI,
    l2Provider
  )

  // Step 3. (UNISWAP) Fix the UniswapV3Factory `owner` address.
  console.log(`fixing UniswapV3Factory owner address`)
  const oldOwnerSlot = toHex32(0)
  const newOwnerSlot = toHex32(3)
  dump.accounts[UNISWAP_FACTORY_ADDRESS].storage[newOwnerSlot] =
    dump.accounts[UNISWAP_FACTORY_ADDRESS].storage[oldOwnerSlot]
  delete dump.accounts[UNISWAP_FACTORY_ADDRESS].storage[oldOwnerSlot]

  // Step 4. (UNISWAP) Fix the UniswapV3Factory `feeAmountTickSpacing` mapping.
  console.log(`fixing UniswapV3Factory feeAmountTickSpacing mapping`)
  const feeEvents = await UniswapV3Factory.queryFilter(
    UniswapV3Factory.filters.FeeAmountEnabled()
  )
  for (const event of feeEvents) {
    const oldSlotKey = getMappingKey([event.args.fee], 1)
    const newSlotKey = getMappingKey([event.args.fee], 4)
    dump.accounts[UNISWAP_FACTORY_ADDRESS].storage[newSlotKey] =
      dump.accounts[UNISWAP_FACTORY_ADDRESS].storage[oldSlotKey]
    delete dump.accounts[UNISWAP_FACTORY_ADDRESS].storage[oldSlotKey]
  }

  // Step 5. (UNISWAP) Figure out the old and new pool addresses.
  console.log(`finding all UniswapV3Factory pool addresses`)
  const pools: {
    [oldAddress: string]: {
      newAddress: string
      token0: string
      token1: string
      fee: ethers.BigNumber
    }
  } = {}
  // TODO: Get these events in a better way
  const poolEvents = await UniswapV3Factory.queryFilter(
    UniswapV3Factory.filters.PoolCreated()
  )
  for (const event of poolEvents) {
    const oldPoolAddress = computePoolAddress({
      factoryAddress: UNISWAP_FACTORY_ADDRESS,
      tokenA: new Token(0, event.args.token0, 18),
      tokenB: new Token(0, event.args.token1, 18),
      fee: event.args.fee,
      initCodeHashManualOverride: POOL_INIT_CODE_HASH_OPTIMISM,
    }).toLowerCase()
    const newPoolAddress = computePoolAddress({
      factoryAddress: UNISWAP_FACTORY_ADDRESS,
      tokenA: new Token(0, event.args.token0, 18),
      tokenB: new Token(0, event.args.token1, 18),
      fee: event.args.fee,
      initCodeHashManualOverride: POOL_INIT_CODE_HASH,
    }).toLowerCase()

    if (oldPoolAddress in dump.accounts) {
      pools[oldPoolAddress] = {
        newAddress: newPoolAddress,
        token0: event.args.token0,
        token1: event.args.token1,
        fee: event.args.fee,
      }
    } else {
      console.log(event)
      throw new Error(
        `found pool event but contract not in state: ${oldPoolAddress}`
      )
    }
  }

  // Step 6. (UNISWAP) Fix the UniswapV3Factory `getPool` mapping.
  for (const newPoolData of Object.values(pools)) {
    // Fix the token0 => token1 => fee mapping
    const oldSlotKey1 = getMappingKey(
      [newPoolData.token0, newPoolData.token1, newPoolData.fee],
      2
    )
    const newSlotKey1 = getMappingKey(
      [newPoolData.token0, newPoolData.token1, newPoolData.fee],
      5
    )
    dump.accounts[UNISWAP_FACTORY_ADDRESS].storage[newSlotKey1] =
      dump.accounts[UNISWAP_FACTORY_ADDRESS].storage[oldSlotKey1]
    delete dump.accounts[UNISWAP_FACTORY_ADDRESS].storage[oldSlotKey1]

    // Fix the token1 => token0 => fee mapping
    const oldSlotKey2 = getMappingKey(
      [newPoolData.token1, newPoolData.token0, newPoolData.fee],
      2
    )
    const newSlotKey2 = getMappingKey(
      [newPoolData.token1, newPoolData.token0, newPoolData.fee],
      5
    )
    dump.accounts[UNISWAP_FACTORY_ADDRESS].storage[newSlotKey2] =
      dump.accounts[UNISWAP_FACTORY_ADDRESS].storage[oldSlotKey2]
    delete dump.accounts[UNISWAP_FACTORY_ADDRESS].storage[oldSlotKey2]
  }

  // Step 7. (UNISWAP) Fix the NonfungiblePositionManager `poolId` mapping.
  for (const [oldPoolAddress, newPoolData] of Object.entries(pools)) {
    const oldSlotKey = getMappingKey([oldPoolAddress], 10)
    const newSlotKey = getMappingKey([newPoolData.newAddress], 10)
    dump.accounts[UNISWAP_NFPM_ADDRESS].storage[newSlotKey] =
      dump.accounts[UNISWAP_NFPM_ADDRESS].storage[oldSlotKey]
    delete dump.accounts[UNISWAP_NFPM_ADDRESS].storage[oldSlotKey]
  }

  // Step 8. (UNISWAP) Perform a final bruteforce step to find any remaining references to old addresses.
  for (const [oldPoolAddress, newPoolData] of Object.entries(pools)) {
    for (const [address, account] of Object.entries(dump.accounts)) {
      if (account.storage === undefined) {
        continue
      }

      // Check for any references to the pool address in storage values.
      for (const [slotKey, slotValue] of Object.entries(account.storage)) {
        if (slotValue.includes(oldPoolAddress.slice(2))) {
          // TODO: Figure out what to do here.
          throw new Error(`found unexpected reference to pool address`)
        }
      }

      // TODO: Choose an appropriate ceiling for the storage slots here
      // Check for single-level nested keys (i.e., address => xxxx).
      for (let i = 0; i < 1000; i++) {
        const oldSlotKey = getMappingKey([oldPoolAddress], i)
        if (account.storage[oldSlotKey] !== undefined) {
          const newSlotKey = getMappingKey([newPoolData.newAddress], i)
          account.storage[newSlotKey] = account.storage[oldSlotKey]
          delete account.storage[oldSlotKey]
        }
      }

      // Check for double-level nested keys (i.e., address => address => xxxx).
      for (let i = 0; i < 1000; i++) {
        for (const otherAddress of Object.keys(dump.accounts)) {
          // otherAddress => poolAddress => xxxx
          const oldSlotKey1 = getMappingKey([otherAddress, oldPoolAddress], i)
          if (account.storage[oldSlotKey1] !== undefined) {
            const newSlotKey = getMappingKey(
              [otherAddress, newPoolData.newAddress],
              i
            )
            account.storage[newSlotKey] = account.storage[oldSlotKey1]
            delete account.storage[oldSlotKey1]
          }

          // poolAddress => otherAddress => xxxx
          const oldSlotKey2 = getMappingKey([oldPoolAddress, otherAddress], i)
          if (account.storage[oldSlotKey2] !== undefined) {
            const newSlotKey = getMappingKey(
              [newPoolData.newAddress, otherAddress],
              i
            )
            account.storage[newSlotKey] = account.storage[oldSlotKey2]
            delete account.storage[oldSlotKey2]
          }
        }
      }
    }
  }

  // Step 9. (UNISWAP) Compute the new code for each pool.
  // Set up a testnet wallet so we can deploy Uniswap pools.
  console.log('deploying pool code')
  const testnetWallet = new ethers.Wallet(
    TESTNET_PRIVATE_KEY,
    new ethers.providers.JsonRpcProvider(TESTNET_PROVIDER_URL)
  )
  for (const [oldPoolAddress, newPoolData] of Object.entries(pools)) {
    let poolCode = await testnetWallet.provider.getCode(newPoolData.newAddress)

    if (poolCode === '0x') {
      console.log(`address ${newPoolData.newAddress} has no code, deploying`)
      const poolCreationTx = await UniswapV3Factory.connect(
        testnetWallet
      ).createPool(newPoolData.token0, newPoolData.token1, newPoolData.fee)
      await poolCreationTx.wait()

      poolCode = await testnetWallet.provider.getCode(newPoolData.newAddress)
      if (poolCode === '0x') {
        throw new Error(`failed to deploy pool`)
      }
    }

    dump.accounts[newPoolData.newAddress] = dump.accounts[oldPoolAddress]
    dump.accounts[newPoolData.newAddress].code = poolCode
    delete dump.accounts[oldPoolAddress]
  }

  /* --- END UNISWAP SURGERY SECTION --- */

  // Step 10. Remove any remaining unverified contracts from the state dump.

  // Step 11. Recompile every remaining contract with the standard compiler.
}

main()
