/* Imports: External */
import { DeployFunction } from 'hardhat-deploy/dist/types'
import '@eth-optimism/hardhat-deploy-config'
import '@nomiclabs/hardhat-ethers'
import 'hardhat-deploy'
import { predeploys } from '@eth-optimism/contracts-bedrock'

const deployFn: DeployFunction = async (hre) => {
  const { deployer } = await hre.getNamedAccounts()

  const Deployment__L1CrossDomainMessenger = await hre.deployments.get(
    'Proxy__OVM_L1CrossDomainMessenger'
  )

  await hre.deployments.deploy('L1ERC721Bridge', {
    from: deployer,
    args: [
      Deployment__L1CrossDomainMessenger.address,
      predeploys.L2ERC721Bridge,
    ],
    log: true,
  })
}

deployFn.tags = ['L1ERC721BridgeImplementation']
deployFn.dependencies = ['L1ERC721BridgeProxy']

export default deployFn
