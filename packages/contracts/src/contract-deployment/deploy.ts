/* External Imports */
import { Contract } from 'ethers'

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
  let AddressManager: Contract

  if (config.addressManager) {
    // console.log(`Connecting to existing address manager.`) //console.logs currently break our deployer
    AddressManager = getContractFactory(
      'Lib_AddressManager',
      config.deploymentSigner
    ).attach(config.addressManager)
  } else {
    // console.log(
    //   `Address manager wasn't provided, so we're deploying a new one.`
    // ) //console.logs currently break our deployer
    AddressManager = await getContractFactory(
      'Lib_AddressManager',
      config.deploymentSigner
    ).deploy()
    if (config.waitForReceipts) {
      await AddressManager.deployTransaction.wait()
    }
  }

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
        .deploy(
          ...(contractDeployParameters.params || []),
          config.deployOverrides
        )
      if (config.waitForReceipts) {
        await contracts[name].deployTransaction.wait()
      }
      const res = await AddressManager.setAddress(name, contracts[name].address)
      if (config.waitForReceipts) {
        await res.wait()
      }
    } catch (err) {
      console.error(`Error deploying ${name}: ${err}`)
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
