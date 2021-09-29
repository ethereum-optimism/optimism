/* External Imports */
import { Signer, ContractFactory, Contract, constants } from 'ethers'
import { TransactionResponse } from '@ethersproject/abstract-provider'
import { Overrides } from '@ethersproject/contracts'

/* Internal Imports */
import { getContractFactory } from '../contract-defs'
import { predeploys } from '../predeploys'

export interface RollupDeployConfig {
  deploymentSigner: Signer
  ovmGasMeteringConfig: {
    minTransactionGasLimit: number
    maxTransactionGasLimit: number
    maxGasPerQueuePerEpoch: number
    secondsPerEpoch: number
  }
  ovmGlobalContext: {
    ovmCHAINID: number
    L2CrossDomainMessengerAddress: string
  }
  transactionChainConfig: {
    sequencer: string | Signer
    forceInclusionPeriodSeconds: number
    forceInclusionPeriodBlocks: number
  }
  stateChainConfig: {
    fraudProofWindowSeconds: number
    sequencerPublishWindowSeconds: number
  }
  l1CrossDomainMessengerConfig: {
    relayerAddress?: string | Signer
  }
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
  const _sendTx = async (
    txPromise: Promise<TransactionResponse>
  ): Promise<TransactionResponse> => {
    const res = await txPromise
    if (config.waitForReceipts) {
      await res.wait()
    }
    return res
  }

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
      factory: getContractFactory('OVM_DeployerWhitelist', undefined, true),
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
    OVM_SafetyChecker: {
      factory: getContractFactory('OVM_SafetyChecker'),
      params: [],
    },
    OVM_ExecutionManager: {
      factory: getContractFactory('OVM_ExecutionManager'),
      params: [
        AddressManager.address,
        config.ovmGasMeteringConfig,
        config.ovmGlobalContext,
      ],
    },
    OVM_StateManager: {
      factory: getContractFactory('OVM_StateManager'),
      params: [await config.deploymentSigner.getAddress()],
      afterDeploy: async (contracts): Promise<void> => {
        await _sendTx(
          contracts.OVM_StateManager.setExecutionManager(
            contracts.OVM_ExecutionManager.address,
            config.deployOverrides
          )
        )
      },
    },
    OVM_ECDSAContractAccount: {
      factory: getContractFactory('OVM_ECDSAContractAccount', undefined, true),
    },
    OVM_SequencerEntrypoint: {
      factory: getContractFactory('OVM_SequencerEntrypoint', undefined, true),
    },
    OVM_ETH: {
      factory: getContractFactory('OVM_ETH'),
      params: [],
    },
    OVM_ProxyEOA: {
      factory: getContractFactory('OVM_ProxyEOA', undefined, true),
    },
    OVM_ExecutionManagerWrapper: {
      factory: getContractFactory(
        'OVM_ExecutionManagerWrapper',
        undefined,
        true
      ),
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