/* External Imports */
import { Signer, ContractFactory, Contract, constants } from 'ethers'
import { TransactionResponse } from '@ethersproject/abstract-provider'
import { Overrides } from '@ethersproject/contracts'

/* Internal Imports */
import { getContractFactory } from '../contract-defs'
import { predeploys } from '../predeploys'

export interface RollupDeployConfig {
  deploymentSigner: Signer
  whitelistConfig: {
    owner: string | Signer
    allowArbitraryContractDeployment: boolean
  }
  gasPriceOracleConfig: {
    owner: string | Signer
    initialGasPrice: number
  }
  addressManager?: string
  dependencies?: string[]
  deployOverrides: Overrides
  waitForReceipts: boolean
}

export interface ContractDeployParameters {
  factory: ContractFactory
  params?: any[]
  afterDeploy?: (contracts?: { [name: string]: Contract }) => Promise<void>
}

export interface ContractDeployConfig {
  [name: string]: ContractDeployParameters
}

export const makeContractDeployConfig = async (
  config: RollupDeployConfig,
  AddressManager: Contract
): Promise<ContractDeployConfig> => {
  return {
    OVM_L2CrossDomainMessenger: {
      factory: getContractFactory('OVM_L2CrossDomainMessenger'),
      params: [AddressManager.address],
    },
    OVM_L2StandardBridge: {
      factory: getContractFactory('OVM_L2StandardBridge'),
      params: [
        predeploys.OVM_L2CrossDomainMessenger,
        constants.AddressZero, // we'll set this to the L1 Bridge address in genesis.go
      ],
    },
    OVM_DeployerWhitelist: {
      factory: getContractFactory('OVM_DeployerWhitelist'),
      params: [],
    },
    OVM_L1MessageSender: {
      factory: getContractFactory('OVM_L1MessageSender'),
      params: [],
    },
    OVM_L2ToL1MessagePasser: {
      factory: getContractFactory('OVM_L2ToL1MessagePasser'),
      params: [],
    },
    OVM_ETH: {
      factory: getContractFactory('OVM_ETH'),
      params: [],
    },
    OVM_GasPriceOracle: {
      factory: getContractFactory('OVM_GasPriceOracle'),
      params: [
        (() => {
          if (typeof config.gasPriceOracleConfig.owner !== 'string') {
            return config.gasPriceOracleConfig.owner.getAddress()
          }
          return config.gasPriceOracleConfig.owner
        })(),
        config.gasPriceOracleConfig.initialGasPrice,
      ],
    },
    OVM_SequencerFeeVault: {
      factory: getContractFactory('OVM_SequencerFeeVault'),
      params: [`0x${'11'.repeat(20)}`],
    },
  }
}
