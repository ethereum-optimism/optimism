import { extendConfig, extendEnvironment } from 'hardhat/config'
import {
  HardhatRuntimeEnvironment,
  Artifact,
  HardhatConfig,
  HardhatUserConfig,
} from 'hardhat/types'
import { lazyObject } from 'hardhat/plugins'
import { Contract, Wallet, providers, Transaction, Signer } from 'ethers'
import { LogDescription, TransactionDescription } from '@ethersproject/abi'
import { Log } from '@ethersproject/abstract-provider'
import 'hardhat-deploy'
import { Deployment } from 'hardhat-deploy/types'
import '@nomiclabs/hardhat-ethers'
import { predeploys } from '@eth-optimism/contracts-bedrock'
import { OpNodeProvider } from '@eth-optimism/core-utils'

import { OptimismNetworkConfig } from './type-extensions'

enum Domain {
  L1,
  L2,
}

export interface ParsedLog {
  name: string
  log: LogDescription
}

export interface ParsedTransaction {
  name: string
  tx: TransactionDescription
}

export interface BuildOptimismOptions {
  l1Signer?: Signer
  l2Signer?: Signer
  l2Url?: string
  l1Url?: string
  opNodeUrl?: string
}

// Builds up contract objects for each of the contracts
const buildOptimism = async (
  hre: HardhatRuntimeEnvironment,
  opts?: BuildOptimismOptions
) => {
  const contracts = [
    {
      name: 'L2OutputOracle',
      proxy: 'L2OutputOracleProxy',
      domain: Domain.L1,
    },
    {
      name: 'L1StandardBridge',
      proxy: 'L1StandardBridgeProxy',
      domain: Domain.L1,
    },
    {
      name: 'L1CrossDomainMessenger',
      proxy: 'L1CrossDomainMessengerProxy',
      domain: Domain.L1,
    },
    {
      name: 'OptimismPortal',
      proxy: 'OptimismPortalProxy',
      domain: Domain.L1,
    },
    {
      name: 'L2ToL1MessagePasser',
      domain: Domain.L2,
    },
    {
      name: 'L2StandardBridge',
      domain: Domain.L2,
    },
    {
      name: 'L2CrossDomainMessenger',
      domain: Domain.L2,
    },
    {
      name: 'SequencerFeeVault',
      domain: Domain.L2,
    },
    {
      name: 'L1Block',
      domain: Domain.L2,
    },
    {
      name: 'GasPriceOracle',
      domain: Domain.L2,
    },
    {
      name: 'AddressManager',
      domain: Domain.L1,
    },
    {
      name: 'OptimismMintableERC20Factory',
      domain: Domain.L2,
    },
    {
      name: 'ProxyAdmin',
      domain: Domain.L1,
    },
  ]

  const out = {}

  let l1Signer = opts?.l1Signer
  if (!l1Signer) {
    const l1Signers = await hre.ethers.getSigners()
    l1Signer = l1Signers[0]
  }

  let networkConfig: OptimismNetworkConfig = {}
  if (hre.config.optimism) {
    networkConfig = hre.config.optimism[hre.network.name] || {}
  }

  let l2Url = opts?.l2Url || 'http://127.0.0.1:9545'
  if (networkConfig.l2Url) {
    l2Url = networkConfig.l2Url
  }

  let opNodeUrl = opts?.opNodeUrl || 'http://127.0.0.1:7545'
  if (networkConfig?.opNodeUrl) {
    opNodeUrl = networkConfig.opNodeUrl
  }

  const l2Provider = new providers.StaticJsonRpcProvider(l2Url)

  let l2Signer = opts?.l2Signer
  if (!l2Signer) {
    l2Signer = new Wallet(hre.network.config.accounts[0], l2Provider)
  }

  const opNodeProvider = new OpNodeProvider(opNodeUrl)

  for (const contract of contracts) {
    let artifact: Artifact
    try {
      artifact = await hre.deployments.getArtifact(contract.name)
    } catch (e) {
      // no op
      if (!artifact) {
        try {
          artifact = await hre.artifacts.readArtifact(contract.name)
        } catch (er) {
          // no op
        }
      }
    }

    if (!artifact) {
      continue
    }

    const deployment = await hre.deployments.getOrNull(contract.name)

    let proxy: Deployment | null
    if (contract.proxy) {
      proxy = await hre.deployments.getOrNull(contract.proxy)
      if (!proxy) {
        continue
      }
    }

    let address: string
    if (proxy?.address) {
      address = proxy.address
    } else if (contract.name in predeploys) {
      address = predeploys[contract.name]
    } else {
      if (!deployment) {
        continue
      }
      address = deployment.address
    }

    out[contract.name] = new Contract(
      address,
      artifact.abi,
      contract.domain === Domain.L1 ? l1Signer : l2Signer
    )
  }

  const findContract = (
    address: string
  ): {
    name: string
    contract: Contract
  } => {
    for (const contract of contracts) {
      if (contract.name in out) {
        if (out[contract.name].address === address) {
          return {
            name: contract.name,
            contract: out[contract.name],
          }
        }
      }
    }
    throw new Error(`Cannot find contract at address ${address}`)
  }

  const parseLog = (log: Log): ParsedLog => {
    const contract = findContract(log.address)
    return {
      name: contract.name,
      log: contract.contract.interface.parseLog(log),
    }
  }

  const parseTransaction = (tx: Transaction): ParsedTransaction => {
    const contract = findContract(tx.to)
    return {
      name: contract.name,
      tx: contract.contract.parseTransaction(tx),
    }
  }

  hre.optimism.contracts = out
  hre.optimism.parseLog = parseLog
  hre.optimism.parseTransaction = parseTransaction
  hre.optimism.opNodeProvider = opNodeProvider
  hre.optimism.l1Signer = l1Signer
  hre.optimism.l2Signer = l2Signer
  hre.optimism.l2Provider = l2Provider
}

extendConfig(
  (config: HardhatConfig, userConfig: Readonly<HardhatUserConfig>) => {
    config.optimism = lazyObject(() => {
      return {
        ...userConfig.optimism,
      }
    })
  }
)

extendEnvironment((hre) => {
  // TODO(tynes): some of these properties don't need to wait
  // until the call to init to be created
  hre.optimism = lazyObject(() => ({
    init: async (opts: BuildOptimismOptions) => buildOptimism(hre, opts),
    contracts: null,
    parseLog: null,
    parseTransaction: null,
    opNodeProvider: null,
    l1Signer: null,
    l2Signer: null,
    l1Provider: null,
    l2Provider: null,
  }))
})
