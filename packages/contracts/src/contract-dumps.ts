/* External Imports */
import * as path from 'path'
import { ethers } from 'ethers'
import * as Ganache from 'ganache-core'

/* Internal Imports */
import { deploy, RollupDeployConfig } from './contract-deployment'
import { getContractDefinition } from './contract-defs'
import { keccak256 } from 'ethers/lib/utils'

type Accounts = Array<{
  originalAddress: string
  address: string
  code: string
}>

interface StorageDump {
  [key: string]: string
}

export interface StateDump {
  contracts: {
    ovmExecutionManager: string
    ovmStateManager: string
  }
  accounts: {
    [address: string]: {
      balance: number
      nonce: number
      code: string
      storage: StorageDump
    }
  }
}

/**
 * Finds the addresses of all accounts changed in the state.
 * @param cStateManager Instance of the callback-based internal vm StateManager.
 * @returns Array of changed addresses.
 */
const getChangedAccounts = async (cStateManager: any): Promise<string[]> => {
  return new Promise<string[]>((resolve, reject) => {
    const accounts: string[] = []
    const stream = cStateManager._trie.createReadStream()

    stream.on('data', (val: any) => {
      accounts.push(val.key.toString('hex'))
    })

    stream.on('end', () => {
      resolve(accounts)
    })
  })
}

/**
 * Generates a storage dump for a given address.
 * @param cStateManager Instance of the callback-based internal vm StateManager.
 * @param address Address to generate a state dump for.
 */
const getStorageDump = async (
  cStateManager: any,
  address: string
): Promise<StorageDump> => {
  return new Promise<StorageDump>((resolve, reject) => {
    cStateManager._getStorageTrie(address, (err: any, trie: any) => {
      if (err) {
        reject(err)
      }

      const storage: StorageDump = {}
      const stream = trie.createReadStream()

      stream.on('data', (val: any) => {
        storage[val.key.toString('hex')] = val.value.toString('hex')
      })

      stream.on('end', () => {
        resolve(storage)
      })
    })
  })
}

/**
 * Replaces old addresses found in a storage dump with new ones.
 * @param storageDump Storage dump to sanitize.
 * @param accounts Set of accounts to sanitize with.
 * @returns Sanitized storage dump.
 */
const sanitizeStorageDump = (
  storageDump: StorageDump,
  accounts: Accounts
): StorageDump => {
  for (const [key, value] of Object.entries(storageDump)) {
    let parsedKey = key
    let parsedValue = value
    for (const account of accounts) {
      const re = new RegExp(`${account.originalAddress}`, 'g')
      parsedValue = parsedValue.replace(re, account.address)
      parsedKey = parsedKey.replace(re, account.address)
    }

    if (parsedKey !== key) {
      delete storageDump[key]
    }

    storageDump[parsedKey] = parsedValue
  }

  return storageDump
}

export const makeStateDump = async (): Promise<any> => {
  const ganache = (Ganache as any).provider({
    gasLimit: 100_000_000,
    allowUnlimitedContractSize: true,
    accounts: [
      {
        secretKey:
          '0x29f3edee0ad3abf8e2699402e0e28cd6492c9be7eaab00d732a791c33552f797',
        balance: 10000000000000000000000000000000000,
      },
    ],
  })

  const provider = new ethers.providers.Web3Provider(ganache)
  const signer = provider.getSigner(0)

  const config: RollupDeployConfig = {
    deploymentSigner: signer,
    ovmGasMeteringConfig: {
      minTransactionGasLimit: 0,
      maxTransactionGasLimit: 1_000_000_000,
      maxGasPerQueuePerEpoch: 1_000_000_000_000,
      secondsPerEpoch: 600,
    },
    transactionChainConfig: {
      sequencer: signer,
      forceInclusionPeriodSeconds: 600,
    },
    whitelistConfig: {
      owner: signer,
      allowArbitraryContractDeployment: true,
    },
  }

  const deploymentResult = await deploy(config)

  const pStateManager = ganache.engine.manager.state.blockchain.vm.pStateManager
  const cStateManager = pStateManager._wrapped

  const ovmExecutionManagerOriginalAddress = deploymentResult.contracts.OVM_ExecutionManager.address
    .slice(2)
    .toLowerCase()
  const ovmExecutionManagerAddress = 'c0dec0dec0dec0dec0dec0dec0dec0dec0de0000'

  const ovmStateManagerOriginalAddress = deploymentResult.contracts.OVM_StateManager.address
    .slice(2)
    .toLowerCase()
  const ovmStateManagerAddress = 'c0dec0dec0dec0dec0dec0dec0dec0dec0de0001'

  const l2ToL1MessagePasserDef = getContractDefinition(
    'OVM_L2ToL1MessagePasser'
  )
  const l2ToL1MessagePasserHash = keccak256(
    l2ToL1MessagePasserDef.deployedBytecode
  )
  const l2ToL1MessagePasserAddress = '4200000000000000000000000000000000000000'

  const l1MessageSenderDef = getContractDefinition('OVM_L1MessageSender')
  const l1MessageSenderHash = keccak256(l1MessageSenderDef.deployedBytecode)
  const l1MessageSenderAddress = '4200000000000000000000000000000000000001'

  const changedAccounts = await getChangedAccounts(cStateManager)

  let deadAddressIndex = 0
  let accounts: Accounts = []

  for (const originalAddress of changedAccounts) {
    const code = (
      await pStateManager.getContractCode(originalAddress)
    ).toString('hex')
    const codeHash = keccak256('0x' + code)

    if (code.length === 0) {
      continue
    }

    // Sorry for this one!
    let address = originalAddress
    if (codeHash === l2ToL1MessagePasserHash) {
      address = l2ToL1MessagePasserAddress
    } else if (codeHash === l1MessageSenderHash) {
      address = l1MessageSenderAddress
    } else if (originalAddress === ovmExecutionManagerOriginalAddress) {
      address = ovmExecutionManagerAddress
    } else if (originalAddress === ovmStateManagerOriginalAddress) {
      address = ovmStateManagerAddress
    } else {
      address = `deaddeaddeaddeaddeaddeaddeaddeaddead${deadAddressIndex
        .toString(16)
        .padStart(4, '0')}`
      deadAddressIndex++
    }

    accounts.push({
      originalAddress,
      address,
      code: code,
    })
  }

  const dump: StateDump = {
    contracts: {
      ovmExecutionManager: '0x' + ovmExecutionManagerAddress,
      ovmStateManager: '0x' + ovmStateManagerAddress,
    },
    accounts: {},
  }

  for (const account of accounts) {
    const storageDump = sanitizeStorageDump(
      await getStorageDump(cStateManager, account.originalAddress),
      accounts
    )

    dump.accounts[account.address] = {
      balance: 0,
      nonce: 0,
      code: account.code,
      storage: storageDump,
    }
  }

  return dump
}

export const getLatestStateDump = (): StateDump => {
  return require(path.join(__dirname, '../dumps', `state-dump.latest.json`))
}
