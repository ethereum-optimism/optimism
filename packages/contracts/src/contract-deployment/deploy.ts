/* External Imports */
import { Signer, Contract, ContractFactory } from 'ethers'

/* Internal Imports */
import { RollupDeployConfig, makeContractDeployConfig } from './config'
import { getContractFactory } from '../contract-defs'

export interface DeployResult {
  AddressManager: Contract
  failedDeployments: string[]
  contracts: {
    [name: string]: Contract
  }
}

export const deploy = async (
  config: RollupDeployConfig
): Promise<DeployResult> => {
  const AddressManager: Contract = await getContractFactory(
    'Lib_AddressManager',
    config.deploymentSigner
  ).deploy()

  const contractDeployConfig = await makeContractDeployConfig(
    config,
    AddressManager
  )

  const failedDeployments: string[] = []
  const contracts: {
    [name: string]: Contract
  } = {}

  for (const [name, contractDeployParameters] of Object.entries(
    contractDeployConfig
  )) {
    if (config.dependencies && !config.dependencies.includes(name)) {
      continue
    }

    try {
      contracts[name] = await contractDeployParameters.factory
        .connect(config.deploymentSigner)
        .deploy(...(contractDeployParameters.params || []))
      await AddressManager.setAddress(name, contracts[name].address)
    } catch (err) {
      failedDeployments.push(name)
    }
  }

  for (const [name, contractDeployParameters] of Object.entries(
    contractDeployConfig
  )) {
    if (config.dependencies && !config.dependencies.includes(name)) {
      continue
    }

    if (contractDeployParameters.afterDeploy) {
      await contractDeployParameters.afterDeploy(contracts)
    }
  }

  return {
    AddressManager,
    failedDeployments,
    contracts,
  }
}
