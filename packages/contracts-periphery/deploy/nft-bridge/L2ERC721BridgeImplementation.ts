/* Imports: External */
import { DeployFunction } from 'hardhat-deploy/dist/types'
import '@eth-optimism/hardhat-deploy-config'
import '@nomiclabs/hardhat-ethers'
import 'hardhat-deploy'
import { predeploys } from '@eth-optimism/contracts'

const deployFn: DeployFunction = async (hre) => {
  const { deployer } = await hre.getNamedAccounts()

  const Deployment__L1ERC721Bridge = await hre.deployments.get(
    'L1ERC721BridgeProxy'
  )

  await hre.deployments.deploy('L2ERC721Bridge', {
    from: deployer,
    args: [
      predeploys.L2CrossDomainMessenger,
      Deployment__L1ERC721Bridge.address,
    ],
    log: true,
  })
}

deployFn.tags = ['L2ERC721BridgeImplementation']
deployFn.dependencies = ['L2ERC721BridgeProxy', 'L1ERC721BridgeProxy']

export default deployFn
