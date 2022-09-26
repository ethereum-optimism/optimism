/* Imports: External */
import { DeployFunction } from 'hardhat-deploy/dist/types'
import '@eth-optimism/hardhat-deploy-config'
import '@nomiclabs/hardhat-ethers'
import 'hardhat-deploy'
import { predeploys } from '@eth-optimism/contracts-bedrock'

const deployFn: DeployFunction = async (hre) => {
  const { deployer } = await hre.getNamedAccounts()
  const remoteChainId = hre.deployConfig.remoteChainId

  await hre.deployments.deploy('OptimismMintableERC721Factory', {
    from: deployer,
    args: [predeploys.L2ERC721Bridge, remoteChainId],
    log: true,
  })
}

deployFn.tags = ['OptimismMintableERC721FactoryImplementation']
deployFn.dependencies = ['OptimismMintableERC721FactoryProxy']

export default deployFn
