/* External Imports */
import { Signer, ContractFactory, Contract } from 'ethers'
import { Overrides } from '@ethersproject/contracts'

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
  ethConfig: {
    initialAmount: number
  }
  whitelistConfig: {
    owner: string | Signer
    allowArbitraryContractDeployment: boolean
  }
  addressManager?: string
  deployOverrides?: Overrides
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
    OVM_L2CrossDomainMessenger: {
      factory: getContractFactory('OVM_L2CrossDomainMessenger'),
      params: [AddressManager.address],
    },
    OVM_L1CrossDomainMessenger: {
      factory: getContractFactory('OVM_L1CrossDomainMessenger'),
      params: [],
      afterDeploy: async (contracts): Promise<void> => {
        if (config.l1CrossDomainMessengerConfig.relayerAddress) {
          const relayer = config.l1CrossDomainMessengerConfig.relayerAddress
          const address =
            typeof relayer === 'string' ? relayer : await relayer.getAddress()
          await AddressManager.setAddress('OVM_L2MessageRelayer', address)
        }
      },
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
        await AddressManager.setAddress(
          'OVM_L2CrossDomainMessenger',
          config.ovmGlobalContext.L2CrossDomainMessengerAddress
        )
      },
    },
    OVM_CanonicalTransactionChain: {
      factory: getContractFactory('OVM_CanonicalTransactionChain'),
      params: [
        AddressManager.address,
        config.transactionChainConfig.forceInclusionPeriodSeconds,
        config.transactionChainConfig.forceInclusionPeriodBlocks,
        config.ovmGasMeteringConfig.maxTransactionGasLimit,
      ],
      afterDeploy: async (): Promise<void> => {
        const sequencer = config.transactionChainConfig.sequencer
        const sequencerAddress =
          typeof sequencer === 'string'
            ? sequencer
            : await sequencer.getAddress()
        await AddressManager.setAddress(
          'OVM_DecompressionPrecompileAddress',
          '0x4200000000000000000000000000000000000005'
        )
        await AddressManager.setAddress('OVM_Sequencer', sequencerAddress)
        await AddressManager.setAddress('Sequencer', sequencerAddress)
      },
    },
    OVM_StateCommitmentChain: {
      factory: getContractFactory('OVM_StateCommitmentChain'),
      params: [
        AddressManager.address,
        config.stateChainConfig.fraudProofWindowSeconds,
        config.stateChainConfig.sequencerPublishWindowSeconds,
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
      params: [],
    },
    OVM_FraudVerifier: {
      factory: getContractFactory('OVM_FraudVerifier'),
      params: [AddressManager.address],
    },
    OVM_StateTransitionerFactory: {
      factory: getContractFactory('OVM_StateTransitionerFactory'),
      params: [AddressManager.address],
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
    OVM_BondManager: {
      factory: getContractFactory('mockOVM_BondManager'),
      params: [AddressManager.address],
    },
    OVM_ETH: {
      factory: getContractFactory('OVM_ETH'),
      params: [config.ethConfig.initialAmount, 'Ether', 18, 'ETH'],
    },
    'OVM_ChainStorageContainer:CTC:batches': {
      factory: getContractFactory('OVM_ChainStorageContainer'),
      params: [AddressManager.address, 'OVM_CanonicalTransactionChain'],
    },
    'OVM_ChainStorageContainer:CTC:queue': {
      factory: getContractFactory('OVM_ChainStorageContainer'),
      params: [AddressManager.address, 'OVM_CanonicalTransactionChain'],
    },
    'OVM_ChainStorageContainer:SCC:batches': {
      factory: getContractFactory('OVM_ChainStorageContainer'),
      params: [AddressManager.address, 'OVM_StateCommitmentChain'],
    },
  }
}
