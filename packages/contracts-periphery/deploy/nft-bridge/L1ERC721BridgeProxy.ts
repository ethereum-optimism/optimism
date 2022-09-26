/* Imports: External */
import { DeployFunction } from 'hardhat-deploy/dist/types'
import '@eth-optimism/hardhat-deploy-config'
import '@nomiclabs/hardhat-ethers'
import 'hardhat-deploy'

const deployFn: DeployFunction = async (hre) => {
  const { deployer } = await hre.getNamedAccounts()

  const { deploy } = await hre.deployments.deterministic(
    'L1ERC721BridgeProxy',
    {
      contract: 'Proxy',
      salt: hre.ethers.utils.solidityKeccak256(
        ['string'],
        ['L1ERC721BridgeProxy']
      ),
      from: deployer,
      args: [hre.deployConfig.ddd],
      log: true,
      waitConfirmations: 1,
    }
  )

  await deploy()

  const Deployment__L1ERC721BridgeProxy = await hre.deployments.get(
    'L1ERC721BridgeProxy'
  )
  console.log(
    `L1ERC721BridgeProxy deployed to ${Deployment__L1ERC721BridgeProxy.address}`
  )
}

deployFn.tags = ['L1ERC721BridgeProxy']

export default deployFn
