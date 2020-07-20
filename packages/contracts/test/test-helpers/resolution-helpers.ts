/* External Imports */
import { ethers } from '@nomiclabs/buidler'
import { Contract, ContractFactory, Signer } from 'ethers'

/* Internal Imports */
import {
  GAS_LIMIT,
  DEFAULT_FORCE_INCLUSION_PERIOD,
  DEFAULT_OPCODE_WHITELIST_MASK
} from './constants'

interface ContractDeployConfig {
  factory: ContractFactory
  params: any[]
}

type ContractFactoryName =
  "L1ToL2TransactionQueue" |
  "SafetyTransactionQueue" |
  "CanonicalTransactionChain" |
  "StateCommitmentChain" |
  "ExecutionManager" |
  "SafetyChecker" |
  "FraudVerifier"

interface AddressResolverDeployConfig {
  L1ToL2TransactionQueue: ContractDeployConfig
  SafetyTransactionQueue: ContractDeployConfig
  CanonicalTransactionChain: ContractDeployConfig
  StateCommitmentChain: ContractDeployConfig
  ExecutionManager: ContractDeployConfig
  SafetyChecker: ContractDeployConfig
  FraudVerifier: ContractDeployConfig
}

interface AddressResolverConfig {
  deployConfig: AddressResolverDeployConfig
  dependencies: ContractFactoryName[]
}

interface ContractMapping {
  l1ToL2TransactionQueue: Contract
  safetyTransactionQueue: Contract
  canonicalTransactionChain: Contract
  stateCommitmentChain: Contract
  executionManager: Contract
  safetyChecker: Contract
  fraudVerifier: Contract
}

export interface AddressResolverMapping {
  addressResolver: Contract
  contracts: ContractMapping
}

const factoryToContractName = {
  L1ToL2TransactionQueue: 'l1ToL2TransactionQueue',
  SafetyTransactionQueue: 'safetyTransactionQueue',
  CanonicalTransactionChain: 'canonicalTransactionChain',
  StateCommitmentChain: 'stateCommitmentChain',
  ExecutionManager: 'executionManager',
  SafetyChecker: 'safetyChecker',
  FraudVerifier: 'fraudVerifier',
}

const getDefaultDeployConfig = async (addressResolver: Contract): Promise<AddressResolverDeployConfig> => {
  const [
    owner,
    sequencer,
    l1ToL2TransactionPasser
  ] = await ethers.getSigners()

  return {
    L1ToL2TransactionQueue: {
      factory: await ethers.getContractFactory('L1ToL2TransactionQueue'),
      params: [addressResolver.address, await l1ToL2TransactionPasser.getAddress()]
    },
    SafetyTransactionQueue: {
      factory: await ethers.getContractFactory('SafetyTransactionQueue'),
      params: [addressResolver.address]
    },
    CanonicalTransactionChain: {
      factory: await ethers.getContractFactory('CanonicalTransactionChain'),
      params: [
        addressResolver.address,
        await sequencer.getAddress(),
        await l1ToL2TransactionPasser.getAddress(),
        DEFAULT_FORCE_INCLUSION_PERIOD
      ]
    },
    StateCommitmentChain: {
      factory: await ethers.getContractFactory('StateCommitmentChain'),
      params: [addressResolver.address]
    },
    ExecutionManager: {
      factory: await ethers.getContractFactory('ExecutionManager'),
      params: [
        addressResolver.address,
        await owner.getAddress(),
        GAS_LIMIT,
      ]
    },
    SafetyChecker: {
      factory: await ethers.getContractFactory('StubSafetyChecker'),
      params: []
    },
    FraudVerifier: {
      factory: await ethers.getContractFactory('FraudVerifier'),
      params: []
    },
  }
}

const makeDeployConfig = async (
  addressResolver: Contract,
  config: Partial<AddressResolverConfig>
): Promise<AddressResolverDeployConfig> => {
  const defaultDeployConfig = await getDefaultDeployConfig(addressResolver)

  return {
    ...defaultDeployConfig,
    ...config.deployConfig,
  }
}

const getLibraryDeployConfig = async (): Promise<any> => {
  return {
    ContractAddressGenerator: {
      factory: await ethers.getContractFactory('ContractAddressGenerator'),
      params: []
    },
    EthMerkleTrie: {
      factory: await ethers.getContractFactory('EthMerkleTrie'),
      params: []
    },
    RLPEncode: {
      factory: await ethers.getContractFactory('RLPEncode'),
      params: []
    },
    RollupMerkleUtils: {
      factory: await ethers.getContractFactory('RollupMerkleUtils'),
      params: []
    },
  }
}

const makeLibraries = async (addressResolver: Contract, signer: Signer): Promise<void> => {
  const libraryDeployConfig = await getLibraryDeployConfig()

  for (const name of Object.keys(libraryDeployConfig)) {
    await deployAndRegister(
      addressResolver,
      signer,
      name,
      libraryDeployConfig[name]
    )
  }
}

export const deployAndRegister = async (
  addressResolver: Contract,
  signer: Signer,
  name: string,
  deployConfig: ContractDeployConfig
): Promise<Contract> => {
  deployConfig.factory.connect(signer)
  const deployedContract = await deployConfig.factory.deploy(...deployConfig.params)
  await addressResolver.setAddress(name, deployedContract.address)
  return deployedContract
}

export const makeAddressResolver = async (
  signer: Signer,
  config: Partial<AddressResolverConfig> = {}
): Promise<AddressResolverMapping> => {
  const AddressResolver = await ethers.getContractFactory('AddressResolver')
  const addressResolver = await AddressResolver.deploy()

  await makeLibraries(addressResolver, signer)

  const deployConfig = await makeDeployConfig(addressResolver, config)

  const contracts: any = {}
  for (const name of Object.keys(deployConfig)) {
    if (config.dependencies === undefined || config.dependencies.includes(name as any)) {
      const contractName = factoryToContractName[name]
      contracts[contractName] = await deployAndRegister(
        addressResolver,
        signer,
        name,
        deployConfig[name],
      )
    }
  }

  return {
    addressResolver,
    contracts
  }
}