/* External Imports */
import { ethers } from '@nomiclabs/buidler'
import { Contract, Signer } from 'ethers'

/* Internal Imports */
import {
  AddressResolverConfig,
  ContractDeployConfig,
  AddressResolverMapping,
  factoryToContractName,
} from './types'
import { getLibraryDeployConfig, makeDeployConfig } from './config'

/**
 * Deploys all necessary libraries.
 * @param addressResolver Address resolver to attach libraries to.
 * @param signer Signer to deploy libraries from.
 */
const deployLibraries = async (
  addressResolver: Contract,
  signer: Signer
): Promise<void> => {
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
  signer: Signer,
  name: string,
  deployConfig: ContractDeployConfig
): Promise<Contract> => {
  deployConfig.factory.connect(signer)
  const deployedContract = await deployConfig.factory.deploy(
    ...deployConfig.params
  )
  await addressResolver.setAddress(name, deployedContract.address)
  return deployedContract
}

/**
 * Creates an address resolver based on some user config. Defaults used for any
 * values not provided for the user.
 * @param signer Wallet to deploy all contracts from.
 * @param config Config used to deploy various contracts.
 * @returns Object containing the resolver and any deployed contracts.
 */
export const makeAddressResolver = async (
  signer: Signer,
  config: Partial<AddressResolverConfig> = {}
): Promise<AddressResolverMapping> => {
  const AddressResolver = await ethers.getContractFactory('AddressResolver')
  const addressResolver = await AddressResolver.deploy()

  await deployLibraries(addressResolver, signer)

  const deployConfig = await makeDeployConfig(addressResolver, config)

  const contracts: any = {}
  for (const name of Object.keys(deployConfig)) {
    if (
      config.dependencies === undefined ||
      config.dependencies.includes(name as any)
    ) {
      const contractName = factoryToContractName[name]
      contracts[contractName] = await deployAndRegister(
        addressResolver,
        signer,
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
