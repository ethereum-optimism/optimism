/* Imports: External */
import { DeployFunction } from 'hardhat-deploy/dist/types'
import '@eth-optimism/hardhat-deploy-config'
import '@nomiclabs/hardhat-ethers'
import 'hardhat-deploy'

const deployFn: DeployFunction = async (hre) => {
  const { deployer } = await hre.getNamedAccounts()

  let remoteChainId: number
  if (hre.network.name === 'optimism') {
    remoteChainId = 1
  } else if (hre.network.name === 'optimism-goerli') {
    remoteChainId = 5
  } else {
    remoteChainId = hre.deployConfig.remoteChainId
  }

  await hre.deployments.deploy('OptimismMintableERC721Factory', {
    from: deployer,
    args: ['0x4200000000000000000000000000000000000014', remoteChainId],
    log: true,
    waitConfirmations: 1,
  })

  const Deployment__OptimismMintableERC721Factory = await hre.deployments.get(
    'OptimismMintableERC721Factory'
  )
  console.log(
    `OptimismMintableERC721Factory deployed to ${Deployment__OptimismMintableERC721Factory.address}`
  )
}

deployFn.tags = ['OptimismMintableERC721FactoryImplementation']
deployFn.dependencies = ['OptimismMintableERC721FactoryProxy']

export default deployFn
