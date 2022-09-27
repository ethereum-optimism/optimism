/* Imports: External */
import { DeployFunction } from 'hardhat-deploy/dist/types'
import '@eth-optimism/hardhat-deploy-config'
import '@nomiclabs/hardhat-ethers'
import 'hardhat-deploy'

const deployFn: DeployFunction = async (hre) => {
  const { deployer } = await hre.getNamedAccounts()
  const { deploy } = hre.deployments

  console.log(`Deploying L1ERC721BridgeProxy to ${hre.network.name}`)
  console.log(`Using deployer ${deployer}`)

  await deploy('L1ERC721BridgeProxy', {
    contract: 'Proxy',
    from: deployer,
    args: [deployer],
    log: true,
    waitConfirmations: 1,
  })

  const Deployment__L1ERC721BridgeProxy = await hre.deployments.get(
    'L1ERC721BridgeProxy'
  )
  console.log(
    `L1ERC721BridgeProxy deployed to ${Deployment__L1ERC721BridgeProxy.address}`
  )
}

deployFn.tags = ['L1ERC721BridgeProxy']

export default deployFn
