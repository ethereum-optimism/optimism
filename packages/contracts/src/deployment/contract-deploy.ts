/* External Imports */
import { Contract } from 'ethers'

/* Internal Imports */
import { getContractFactory } from '../contract-imports'
import { mergeDefaultConfig } from './default-config'
import {
  ContractDeployOptions,
  RollupDeployConfig,
  factoryToContractName,
  AddressResolverMapping,
} from './types'

/**
 * Deploys a single contract.
 * @param config Contract deployment configuration.
 * @return Deployed contract.
 */
const deployContract = async (
  config: ContractDeployOptions
): Promise<Contract> => {
  config.factory = config.factory.connect(config.signer)
  const deployedContract = await config.factory.deploy(...config.params)
  return deployedContract
}

/**
 * Deploys a contract and registers it with the address resolver.
 * @param addressResolver Address resolver to register to.
 * @param signer Wallet to deploy the contract from.
 * @param name Name of the contract within the resolver.
 * @param deployConfig Contract deployment configuration.
 * @returns Ethers Contract instance.
 */
export const deployAndRegister = async (
  addressResolver: Contract,
  name: string,
  deployConfig: ContractDeployOptions
): Promise<Contract> => {
  const deployedContract = await deployContract(deployConfig)
  await addressResolver.setAddress(name, deployedContract.address)
  return deployedContract
}

/**
 * Deploys all contracts according to a config.
 * @param config Contract deployment config.
 * @return AddressResolver and all other contracts.
 */
export const deployAllContracts = async (
  config: RollupDeployConfig
): Promise<AddressResolverMapping> => {
  if (!config.addressResolverConfig) {
    config.addressResolverConfig = {
      factory: getContractFactory('AddressResolver'),
      params: [],
      signer: config.signer,
    }
  }

  const addressResolver = await deployContract(config.addressResolverConfig)

  const deployConfig = await mergeDefaultConfig(
    config.contractDeployConfig,
    addressResolver,
    config.signer,
    config.rollupOptions
  )

  const contracts: any = {}
  for (const name of Object.keys(deployConfig)) {
    if (!config.dependencies || config.dependencies.includes(name as any)) {
      const contractName = factoryToContractName[name]
      contracts[contractName] = await deployAndRegister(
        addressResolver,
        name,
        deployConfig[name]
      )
    }
  }

  return {
    addressResolver,
    contracts,
  }
}
