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
  //here is the deployed value of ropsten net, remove for new one
  //config.addressManager="0xFa51A89716C6991Df44116654Fc90d4b246F2ff5"
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
    console.log("deployed address manager:"+AddressManager.address);
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

    var addr=await AddressManager.getAddress(name);
    console.log(name+":"+addr);
    var fac=contractDeployParameters.factory
                .connect(config.deploymentSigner)
    if(addr == "0x0000000000000000000000000000000000000000"){
      try {
        contracts[name] = await fac
          .connect(config.deploymentSigner)
          .deploy(
            ...(contractDeployParameters.params || [])
          )
        if (config.waitForReceipts) {
          await contracts[name].deployTransaction.wait()
        }
        const res = await AddressManager.setAddress(name, contracts[name].address)
        if (config.waitForReceipts) {
          await res.wait()
        }
        console.log("deployed "+name+" contract.");
      } catch (err) {
        console.error(`Error deploying ${name}: ${err}`)
        failedDeployments.push(name)
      }
    }else{
        const d=fac.attach(addr);
        contracts[name] = d;
    }
    console.log(name+":"+contracts[name].address);
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
