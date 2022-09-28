/* Imports: External */
import { DeployFunction } from 'hardhat-deploy/dist/types'
import '@eth-optimism/hardhat-deploy-config'
import '@nomiclabs/hardhat-ethers'
import 'hardhat-deploy'

import { getProxyAdmin, predeploy } from '../../src/nft-bridge-deploy-helpers'

const deployFn: DeployFunction = async (hre) => {
  const { deployer } = await hre.getNamedAccounts()
  const { getAddress } = hre.ethers.utils

  console.log(`Deploying OptimismMintableERC721Factory to ${hre.network.name}`)
  console.log(`Using deployer ${deployer}`)

  const Deployment__OptimismMintableERC721FactoryProxy =
    await hre.deployments.get('OptimismMintableERC721FactoryProxy')

  const OptimismMintableERC721FactoryProxy = await hre.ethers.getContractAt(
    'Proxy',
    Deployment__OptimismMintableERC721FactoryProxy.address
  )

  // Check that the admin of the OptimismMintableERC721FactoryProxy is the
  // deployer. This makes it easy to upgrade the implementation of the proxy
  // and then transfer the admin privilege after deploying the implementation
  const admin = await OptimismMintableERC721FactoryProxy.callStatic.admin()
  if (getAddress(admin) !== getAddress(deployer)) {
    throw new Error('deployer is not proxy admin')
  }

  let remoteChainId: number
  if (hre.network.name === 'optimism') {
    remoteChainId = 1
  } else if (hre.network.name === 'optimism-goerli') {
    remoteChainId = 5
  } else if (hre.network.name === 'ops-l2') {
    remoteChainId = 31337
  } else {
    remoteChainId = hre.deployConfig.remoteChainId
  }

  if (typeof remoteChainId !== 'number') {
    throw new Error('remoteChainId not defined')
  }

  await hre.deployments.deploy('OptimismMintableERC721Factory', {
    from: deployer,
    args: [predeploy, remoteChainId],
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
    const newAdmin = getProxyAdmin(hre.network.name)
    const tx = await OptimismMintableERC721FactoryProxy.changeAdmin(newAdmin)
    const receipt = await tx.wait()
    console.log(
      `OptimismMintableERC721FactoryProxy admin updated: ${receipt.transactionHash}`
    )
  }
}

deployFn.tags = ['OptimismMintableERC721FactoryImplementation']
deployFn.dependencies = ['OptimismMintableERC721FactoryProxy']

export default deployFn
