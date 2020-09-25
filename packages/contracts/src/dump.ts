/* External Imports */
import * as path from 'path'
import { ethers } from 'ethers'
import * as Ganache from 'ganache-core'
import { deployAllContracts, RollupDeployConfig } from './deployment'

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

export const makeStateDump = async (): Promise<any> => {
  const ganache = Ganache.provider({
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
    signer,
    rollupOptions: {
      forceInclusionPeriodSeconds: 600,
      ownerAddress: await signer.getAddress(),
      sequencerAddress: await signer.getAddress(),
      gasMeterConfig: {
        ovmTxFlatGasFee: 0,
        ovmTxMaxGas: 1_000_000_000,
        gasRateLimitEpochLength: 600,
        maxSequencedGasPerEpoch: 1_000_000_000_000,
        maxQueuedGasPerEpoch: 1_000_000_000_000,
      },
      deployerWhitelistOwnerAddress: await signer.getAddress(),
      allowArbitraryContractDeployment: true,
    },
  }

  const resolver = await deployAllContracts(config)

  const dump: StateDump = {
    contracts: {
      ovmExecutionManager: resolver.contracts.executionManager.address,
      ovmStateManager: resolver.contracts.stateManager.address,
    },
    accounts: {},
  }

  const pStateManager = ganache.engine.manager.state.blockchain.vm.pStateManager
  const cStateManager = pStateManager._wrapped

  const changedAccounts = await getChangedAccounts(cStateManager)
  for (const account of changedAccounts) {
    const code = await pStateManager.getContractCode(account)

    if (code.length > 0) {
      dump.accounts[account] = {
        balance: 0,
        nonce: 0,
        code: code.toString('hex'),
        storage: await getStorageDump(cStateManager, account),
      }
    }
  }

  return dump
}

export const getLatestStateDump = (): StateDump => {
  return require(path.join(__dirname, '../dumps', `state-dump.latest.json`))
}
