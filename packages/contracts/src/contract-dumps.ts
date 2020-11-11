/* External Imports */
import * as path from 'path'
import { ethers } from 'ethers'
import * as Ganache from 'ganache-core'
import { keccak256 } from 'ethers/lib/utils'

/* Internal Imports */
import { deploy, RollupDeployConfig } from './contract-deployment'
import { fromHexString, toHexString, remove0x } from './utils'
import { getContractDefinition } from './contract-defs'

interface StorageDump {
  [key: string]: string
}

export interface StateDump {
  accounts: {
    [name: string]: {
      address: string
      code: string
      codeHash: string
      storage: StorageDump
      abi: any
    }
  }
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
        storage['0x' + val.key.toString('hex')] =
          '0x' + val.value.toString('hex').slice(2)
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
  accounts: Array<{
    originalAddress: string
    deadAddress: string
  }>
): StorageDump => {
  for (const account of accounts) {
    account.originalAddress = remove0x(account.originalAddress).toLowerCase()
    account.deadAddress = remove0x(account.deadAddress).toLowerCase()
  }

  for (const [key, value] of Object.entries(storageDump)) {
    let parsedKey = key
    let parsedValue = value
    for (const account of accounts) {
      const re = new RegExp(`${account.originalAddress}`, 'g')
      parsedValue = parsedValue.replace(re, account.deadAddress)
      parsedKey = parsedKey.replace(re, account.deadAddress)
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
      secondsPerEpoch: 0,
    },
    ovmGlobalContext: {
      ovmCHAINID: 420,
    },
    transactionChainConfig: {
      sequencer: signer,
      forceInclusionPeriodSeconds: 600,
    },
    whitelistConfig: {
      owner: signer,
      allowArbitraryContractDeployment: true,
    },
    dependencies: [
      'Lib_AddressManager',
      'OVM_DeployerWhitelist',
      'OVM_L1MessageSender',
      'OVM_L2ToL1MessagePasser',
      'OVM_ProxyEOA',
      'OVM_ECDSAContractAccount',
      'OVM_ProxySequencerEntrypoint',
      'OVM_SequencerEntrypoint',
      'OVM_L2CrossDomainMessenger',
      'OVM_SafetyChecker',
      'OVM_ExecutionManager',
      'OVM_StateManager',
      'mockOVM_ECDSAContractAccount',
    ],
  }

  const precompiles = {
    OVM_L2ToL1MessagePasser: '0x4200000000000000000000000000000000000000',
    OVM_L1MessageSender: '0x4200000000000000000000000000000000000001',
    OVM_DeployerWhitelist: '0x4200000000000000000000000000000000000002',
    OVM_ECDSAContractAccount: '0x4200000000000000000000000000000000000003',
    OVM_ProxySequencerEntrypoint: '0x4200000000000000000000000000000000000004',
    OVM_SequencerEntrypoint: '0x4200000000000000000000000000000000000005',
  }

  const deploymentResult = await deploy(config)
  deploymentResult.contracts['Lib_AddressManager'] =
    deploymentResult.AddressManager

  if (deploymentResult.failedDeployments.length > 0) {
    throw new Error(
      `Could not generate state dump, deploy failed for: ${deploymentResult.failedDeployments}`
    )
  }

  const pStateManager = ganache.engine.manager.state.blockchain.vm.pStateManager
  const cStateManager = pStateManager._wrapped

  const dump: StateDump = {
    accounts: {},
  }

  for (let i = 0; i < Object.keys(deploymentResult.contracts).length; i++) {
    const name = Object.keys(deploymentResult.contracts)[i]
    const contract = deploymentResult.contracts[name]

    const codeBuf = await pStateManager.getContractCode(
      fromHexString(contract.address)
    )
    const code = toHexString(codeBuf)

    const deadAddress =
      precompiles[name] ||
      `0xdeaddeaddeaddeaddeaddeaddeaddeaddead${i.toString(16).padStart(4, '0')}`

    dump.accounts[name] = {
      address: deadAddress,
      code,
      codeHash: keccak256(code),
      storage: await getStorageDump(cStateManager, contract.address),
      abi: getContractDefinition(name.replace('Proxy__', '')).abi,
    }
  }

  const addressMap = Object.keys(dump.accounts).map((name) => {
    return {
      originalAddress: deploymentResult.contracts[name].address,
      deadAddress: dump.accounts[name].address,
    }
  })

  for (const name of Object.keys(dump.accounts)) {
    dump.accounts[name].storage = sanitizeStorageDump(
      dump.accounts[name].storage,
      addressMap
    )
  }

  return dump
}

export const getLatestStateDump = (): StateDump => {
  return require(path.join(__dirname, '../dumps', `state-dump.latest.json`))
}
