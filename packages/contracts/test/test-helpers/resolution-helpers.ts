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

interface AddressResolverConfig {
  L1ToL2TransactionQueue: ContractDeployConfig
  SafetyTransactionQueue: ContractDeployConfig
  CanonicalTransactionChain: ContractDeployConfig
  StateCommitmentChain: ContractDeployConfig
  ExecutionManager: ContractDeployConfig
  SafetyChecker: ContractDeployConfig
  FraudVerifier: ContractDeployConfig
}

const getDefaultConfig = async (addressResolver: Contract): Promise<AddressResolverConfig> => {
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
      factory: await ethers.getContractFactory('SafetyChecker'),
      params: [
        addressResolver.address,
        DEFAULT_OPCODE_WHITELIST_MASK
      ]
    },
    FraudVerifier: {
      factory: await ethers.getContractFactory('FraudVerifier'),
      params: []
    },
  }
}

const makeConfig = async (
  addressResolver: Contract,
  config: Partial<AddressResolverConfig>
): Promise<AddressResolverConfig> => {
  const defaultConfig = await getDefaultConfig(addressResolver)

  return {
    ...defaultConfig,
    ...config,
  }
}

const makeLibraries = async (addressResolver: Contract): Promise<void> => {
  const LibraryGenerator = await ethers.getContractFactory('LibraryGenerator')
  const libraryGenerator = await LibraryGenerator.deploy(addressResolver)
  await libraryGenerator.makeAll({
    gasLimit: GAS_LIMIT
  })
}

const deployAndRegister = async (
  addressResolver: Contract,
  name: string,
  deployConfig: ContractDeployConfig
): Promise<Contract> => {
  const deployedContract = await deployConfig.factory.deploy(...deployConfig.params)
  await addressResolver.setAddress(name, deployedContract.address)
  return deployedContract
}

export const makeAddressResolver = async (
  signer: Signer,
  config: Partial<AddressResolverConfig>
): Promise<{
  [name: string]: Contract
}> => {
  const AddressResolver = await ethers.getContractFactory('AddressResolver')
  const addressResolver = await AddressResolver.deploy()
  await makeLibraries(addressResolver)

  config = await makeConfig(addressResolver, config)

  const contracts: { [name: string]: Contract } = {}
  for (const name of Object.keys(config)) {
    contracts[name] = await deployAndRegister(
      addressResolver,
      name,
      config[name],
    )
  }

  return contracts
}