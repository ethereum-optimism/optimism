/* External Imports */
import { Signer, ContractFactory, Contract } from 'ethers'

/* Internal Imports */
import { getContractFactory } from '../contract-defs'

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
  }
  transactionChainConfig: {
    sequencer: string | Signer
    forceInclusionPeriodSeconds: number
  }
  whitelistConfig: {
    owner: string | Signer
    allowArbitraryContractDeployment: boolean
  }
  dependencies?: string[]
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
    OVM_L1CrossDomainMessenger: {
      factory: getContractFactory('OVM_L1CrossDomainMessenger'),
      params: [],
    },
    Proxy__OVM_L1CrossDomainMessenger: {
      factory: getContractFactory('Lib_ResolvedDelegateProxy'),
      params: [AddressManager.address, 'OVM_L1CrossDomainMessenger'],
      afterDeploy: async (contracts): Promise<void> => {
        const xDomainMessenger = getContractFactory(
          'OVM_L1CrossDomainMessenger'
        )
          .connect(config.deploymentSigner)
          .attach(contracts.Proxy__OVM_L1CrossDomainMessenger.address)
        await xDomainMessenger.initialize(AddressManager.address)
      },
    },
    OVM_L2CrossDomainMessenger: {
      factory: getContractFactory('OVM_L2CrossDomainMessenger'),
      params: [AddressManager.address],
    },
    OVM_CanonicalTransactionChain: {
      factory: getContractFactory('OVM_CanonicalTransactionChain'),
      params: [
        AddressManager.address,
        config.transactionChainConfig.forceInclusionPeriodSeconds,
      ],
      afterDeploy: async (contracts): Promise<void> => {
        const sequencer = config.transactionChainConfig.sequencer
        const sequencerAddress =
          typeof sequencer === 'string'
            ? sequencer
            : await sequencer.getAddress()
        await AddressManager.setAddress('OVM_Sequencer', sequencerAddress)
        await AddressManager.setAddress('Sequencer', sequencerAddress)
        await contracts.OVM_CanonicalTransactionChain.init()
      },
    },
    OVM_StateCommitmentChain: {
      factory: getContractFactory('OVM_StateCommitmentChain'),
      params: [AddressManager.address],
      afterDeploy: async (contracts): Promise<void> => {
        await contracts.OVM_StateCommitmentChain.init()
      },
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
        await contracts.OVM_StateManager.setExecutionManager(
          contracts.OVM_ExecutionManager.address
        )
      },
    },
    OVM_StateManagerFactory: {
      factory: getContractFactory('OVM_StateManagerFactory'),
    },
    OVM_FraudVerifier: {
      factory: getContractFactory('OVM_FraudVerifier'),
      params: [AddressManager.address],
    },
    OVM_StateTransitionerFactory: {
      factory: getContractFactory('OVM_StateTransitionerFactory'),
    },
    OVM_ECDSAContractAccount: {
      factory: getContractFactory('OVM_ECDSAContractAccount'),
    },
    OVM_SequencerEntrypoint: {
      factory: getContractFactory('OVM_SequencerEntrypoint'),
    },
    OVM_ProxySequencerEntrypoint: {
      factory: getContractFactory('OVM_ProxySequencerEntrypoint'),
    },
    mockOVM_ECDSAContractAccount: {
      factory: getContractFactory('mockOVM_ECDSAContractAccount'),
    },
  }
}
