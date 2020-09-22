/* External Imports */
import { getLogger } from '@eth-optimism/core-utils'
import { Signer } from 'ethers'

/* Internal Imports */
import {
  AddressResolverMapping,
  DeployResult,
  factoryToContractName,
  RollupOptions,
} from './types'
import {
  GAS_METER_PARAMS,
  getL1ContractOwnerAddress,
  getL1DeploymentSigner,
  getL1SequencerAddress,
} from './config'
import { Environment } from './environment'
import { deployAllContracts } from './contract-deploy'

const log = getLogger('deploy-l1-rollup-contracts')

/**
 * Deploys all L1 contracts according to the environment variable configuration.
 * Please see README for more info.
 */
export const deployContracts = async (): Promise<DeployResult> => {
  let res: DeployResult
  try {
    const signer: Signer = getL1DeploymentSigner()
    log.info(`Read deployer wallet info. Address: ${await signer.getAddress()}`)

    const ownerAddress: string = await getL1ContractOwnerAddress()
    const sequencerAddress: string = getL1SequencerAddress()
    const rollupOptions: RollupOptions = {
      forceInclusionPeriodSeconds: Environment.forceInclusionPeriodSeconds(),
      ownerAddress,
      sequencerAddress,
      gasMeterConfig: GAS_METER_PARAMS,
      deployerWhitelistOwnerAddress:  ownerAddress,
      allowArbitraryContractDeployment: true,
    }

    res = await deployAllContracts({
      signer,
      rollupOptions,
      addressResolverContractAddress: Environment.addressResolverContractAddress(),
    })
  } catch (e) {
    log.error(`Error deploying contracts: ${e.message}`)
    return undefined
  }

  log.info(`\n\nSuccessfully deployed the following contracts:`)
  log.info(`\taddressResolver: ${res.addressResolver.address}`)
  Object.keys(res.contracts).forEach((key) => {
    if (res.contracts[key]) {
      log.info(`\t${key}: ${res.contracts[key].address}`)
    }
  })

  log.info(`\nThe following contracts failed deployment:`)
  res.failedDeployments.forEach((contractName) => {
    log.info(`\t${factoryToContractName[contractName]}`)
  })

  return res
}
