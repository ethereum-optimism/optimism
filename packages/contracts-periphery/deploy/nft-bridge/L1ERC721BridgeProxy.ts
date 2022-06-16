/* Imports: External */
import { DeployFunction } from 'hardhat-deploy/dist/types'

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
    }
  )

  await deploy()
}

deployFn.tags = ['L1ERC721BridgeProxy']

export default deployFn
