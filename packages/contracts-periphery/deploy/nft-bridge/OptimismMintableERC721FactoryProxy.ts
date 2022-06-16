/* Imports: External */
import { DeployFunction } from 'hardhat-deploy/dist/types'

const deployFn: DeployFunction = async (hre) => {
  const { deployer } = await hre.getNamedAccounts()

  const { deploy } = await hre.deployments.deterministic(
    'OptimismMintableERC721FactoryProxy',
    {
      contract: 'Proxy',
      salt: hre.ethers.utils.solidityKeccak256(
        ['string'],
        ['OptimismMintableERC721FactoryProxy']
      ),
      from: deployer,
      args: [hre.deployConfig.ddd],
      log: true,
    }
  )

  await deploy()
}

deployFn.tags = ['OptimismMintableERC721FactoryProxy']

export default deployFn
