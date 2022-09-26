/* Imports: External */
import { DeployFunction } from 'hardhat-deploy/dist/types'
import '@eth-optimism/hardhat-deploy-config'
import '@nomiclabs/hardhat-ethers'
import 'hardhat-deploy'

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
      waitConfirmations: 1,
    }
  )

  await deploy()

  const Deployment__OptimismMintableERC721FactoryProxy =
    await hre.deployments.get('OptimismMintableERC721FactoryProxy')
  console.log(
    `OptimismMintableERC721FactoryProxy deployed to ${Deployment__OptimismMintableERC721FactoryProxy.address}`
  )
}

deployFn.tags = ['OptimismMintableERC721FactoryProxy']

export default deployFn
