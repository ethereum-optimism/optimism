/* Imports: External */
import { DeployFunction } from 'hardhat-deploy/dist/types'
import '@eth-optimism/hardhat-deploy-config'
import '@nomiclabs/hardhat-ethers'
import 'hardhat-deploy'

const deployFn: DeployFunction = async (hre) => {
  const { deployer } = await hre.getNamedAccounts()
  const { getAddress } = hre.ethers.utils

  const Deployment__OptimismMintableERC721FactoryProxy =
    await hre.deployments.get('OptimismMintableERC721FactoryProxy')

  const OptimismMintableERC721FactoryProxy = await hre.ethers.getContractAt(
    'Proxy',
    Deployment__OptimismMintableERC721FactoryProxy.address
  )

  // Check that the admin of the OptimismMintableERC721FactoryProxy is the
  // deployer. This makes it easy to upgrade the implementation of the proxy
  // and then transfer the admin privilege after deploying the implementation
  const admin = await OptimismMintableERC721FactoryProxy.admin()
  if (getAddress(admin) !== getAddress(deployer)) {
    throw new Error('deployer is not proxy admin')
  }

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

  {
    // Upgrade the Proxy to the newly deployed implementation
    const tx = await OptimismMintableERC721FactoryProxy.upgradeTo(
      Deployment__OptimismMintableERC721Factory.address
    )
    const receipt = await tx.wait()
    console.log(
      `OptimismMintableERC721FactoryProxy upgraded: ${receipt.transactionHash}`
    )
  }

  {
    if (
      hre.network.name === 'optimism' ||
      hre.network.name === 'optimism-goerli'
    ) {
      let newAdmin: string
      if (hre.network.name === 'optimism') {
        newAdmin = '0x2501c477D0A35545a387Aa4A3EEe4292A9a8B3F0'
      } else if (hre.network.name === 'optimism-goerli') {
        newAdmin = '0xf80267194936da1E98dB10bcE06F3147D580a62e'
      }
      const tx = await OptimismMintableERC721FactoryProxy.changeAdmin(newAdmin)
      const receipt = await tx.wait()
      console.log(
        `OptimismMintableERC721FactoryProxy admin updated: ${receipt.transactionHash}`
      )
    }
  }
}

deployFn.tags = ['OptimismMintableERC721FactoryImplementation']
deployFn.dependencies = ['OptimismMintableERC721FactoryProxy']

export default deployFn
