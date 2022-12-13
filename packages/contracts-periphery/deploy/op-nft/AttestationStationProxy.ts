/* Imports: External */
import { DeployFunction } from 'hardhat-deploy/dist/types'
import '@eth-optimism/hardhat-deploy-config'
import '@nomiclabs/hardhat-ethers'
import 'hardhat-deploy'
import {
  assertContractVariable,
  deploy,
  getDeploymentAddress,
} from '@eth-optimism/contracts-bedrock/src/deploy-utils'

const deployFn: DeployFunction = async (hre) => {
  const proxyAdmin = await getDeploymentAddress(hre, 'ProxyAdmin')

  await deploy({
    hre,
    name: 'AttestationStationProxy',
    contract: 'Proxy',
    args: [proxyAdmin],
    postDeployAction: async (contract) => {
      await assertContractVariable(contract, 'admin', proxyAdmin)
    },
  })

  const Deployment__AttestationStationProxy = await hre.deployments.get(
    'AttestationStationProxy'
  )
  console.log(
    `AttestationStationProxy deployed to ${Deployment__AttestationStationProxy.address}`
  )
}

deployFn.tags = ['AttestationStationProxy']

export default deployFn
