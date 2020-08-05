/* External Imports */
import { getLogger, remove0x, ZERO_ADDRESS } from '@eth-optimism/core-utils'

import { Contract, ethers } from 'ethers'
/* Internal Imports */
import {
  getContractFactory,
  getContractInterface,
} from '../contract-imports'
import { mergeDefaultConfig } from './default-config'
import {
  AddressResolverMapping,
  ContractDeployOptions,
  factoryToContractName,
  RollupDeployConfig,
} from './types'

const log = getLogger('contract-deploy')

/**
 * Deploys a single contract.
 * @param config Contract deployment configuration.
 * @return Deployed contract.
 */
const deployContract = async (
  config: ContractDeployOptions
): Promise<Contract> => {
  config.factory = config.factory.connect(config.signer)
  const rawTx = config.factory.getDeployTransaction(...config.params)

  // Can't use this because it fails on ExecutionManager & FraudVerifier
  // return config.factory.deploy(...config.params)

  const res = await config.signer.sendTransaction({
    data: rawTx.data,
    gasLimit: 9_500_000,
    gasPrice: 2_000_000_000,
    value: 0,
    nonce: await config.signer.getTransactionCount('pending'),
  })

  const receipt: ethers.providers.TransactionReceipt = await config.signer.provider.waitForTransaction(
    res.hash
  )

  return new Contract(
    receipt.contractAddress,
    config.factory.interface,
    config.signer
  )
}

/**
 * Deploys a contract and registers it with the address resolver.
 * @param addressResolver Address resolver to register to.
 * @param name Name of the contract within the resolver.
 * @param deployConfig Contract deployment configuration.
 * @returns Ethers Contract instance.
 */
export const deployAndRegister = async (
  addressResolver: Contract,
  name: string,
  deployConfig: ContractDeployOptions
): Promise<Contract> => {
  log.debug(`Deploying ${name}...`)
  const deployedContract = await deployContract(deployConfig)
  log.info(`Deployed ${name} at address ${deployedContract.address}.`)

  log.debug(`Registering ${name} with AddressResolver`)
  const res: ethers.providers.TransactionResponse = await addressResolver.setAddress(
    name,
    deployedContract.address
  )
  await addressResolver.provider.waitForTransaction(res.hash)
  log.debug(
    `Registered ${name} with AddressResolver (${addressResolver.address})`
  )
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
  let addressResolver: Contract
  if (!config.addressResolverContractAddress) {
    if (!config.addressResolverConfig) {
      config.addressResolverConfig = {
        factory: getContractFactory('AddressResolver'),
        params: [],
        signer: config.signer,
      }
    }
    log.debug(`No deployed AddressResolver found. Deploying...`)
    addressResolver = await deployContract(config.addressResolverConfig)
    log.info(`Deployed AddressResolver to ${addressResolver.address}`)
  } else {
    log.info(
      `Using deployed AddressResolver at address ${config.addressResolverContractAddress}`
    )
    addressResolver = new Contract(
      config.addressResolverContractAddress,
      getContractInterface('AddressResolver'),
      config.signer
    )
  }

  const deployConfig = await mergeDefaultConfig(
    addressResolver.address,
    config.contractDeployConfig,
    config.signer,
    config.rollupOptions
  )

  const contracts: any = {}
  for (const name of Object.keys(deployConfig)) {
    if (!config.dependencies || config.dependencies.includes(name as any)) {
      const contractName = factoryToContractName[name]
      const deployedAddress = await addressResolver.getAddress(name)

      if (!!deployedAddress && deployedAddress !== ZERO_ADDRESS) {
        log.info(
          `Using existing deployed and registered contract for ${name} at address ${deployedAddress}`
        )
        contracts[contractName] = new Contract(
          deployedAddress,
          deployConfig[name].factory.interface,
          config.signer
        )
        continue
      }

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
