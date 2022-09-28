/* Imports: External */
import { DeployFunction } from 'hardhat-deploy/dist/types'
import '@eth-optimism/hardhat-deploy-config'
import '@nomiclabs/hardhat-ethers'
import 'hardhat-deploy'
import { predeploys } from '@eth-optimism/contracts'

import {
  isTargetL2Network,
  predeploy,
  validateERC721Bridge,
  getProxyAdmin,
} from '../../src/nft-bridge-deploy-helpers'

const deployFn: DeployFunction = async (hre) => {
  const { deployer } = await hre.getNamedAccounts()
  const { getAddress } = hre.ethers.utils

  if (!isTargetL2Network(hre.network.name)) {
    console.log(`Deploying to unsupported network ${hre.network.name}`)
    return
  }

  console.log(`Deploying L2ERC721Bridge to ${hre.network.name}`)
  console.log(`Using deployer ${deployer}`)

  const L2ERC721BridgeProxy = await hre.ethers.getContractAt('Proxy', predeploy)

  // Check to make sure that the admin of the proxy is the deployer.
  // The deployer of the L2ERC721Bridge should be the same as the
  // admin of the L2ERC721BridgeProxy so that it is easy to upgrade
  // the implementation. The admin is then changed depending on the
  // network after the L2ERC721BridgeProxy is upgraded to the implementation
  const admin = await L2ERC721BridgeProxy.callStatic.admin()

  if (getAddress(admin) !== getAddress(deployer)) {
    throw new Error(`Unexpected admin ${admin}`)
  }

  const Deployment__L1ERC721Bridge = await hre.deployments.get(
    'L1ERC721BridgeProxy'
  )

  const L1ERC721BridgeAddress = Deployment__L1ERC721Bridge.address

  // Deploy the L2ERC721Bridge implementation
  await hre.deployments.deploy('L2ERC721Bridge', {
    from: deployer,
    args: [predeploys.L2CrossDomainMessenger, L1ERC721BridgeAddress],
    log: true,
    waitConfirmations: 1,
  })

  const Deployment__L2ERC721Bridge = await hre.deployments.get('L2ERC721Bridge')
  console.log(
    `L2ERC721Bridge deployed to ${Deployment__L2ERC721Bridge.address}`
  )

  await validateERC721Bridge(hre, Deployment__L2ERC721Bridge.address, {
    messenger: predeploys.L2CrossDomainMessenger,
    otherBridge: L1ERC721BridgeAddress,
  })

  {
    // Upgrade the implementation of the proxy to the newly deployed
    // L2ERC721Bridge
    const tx = await L2ERC721BridgeProxy.upgradeTo(
      Deployment__L2ERC721Bridge.address
    )
    const receipt = await tx.wait()
    console.log(
      `Upgraded the implementation of the L2ERC721BridgeProxy: ${receipt.transactionhash}`
    )
  }

  await validateERC721Bridge(hre, L2ERC721BridgeProxy.address, {
    messenger: predeploys.L2CrossDomainMessenger,
    otherBridge: L1ERC721BridgeAddress,
  })

  {
    const newAdmin = getProxyAdmin(hre.network.name)
    console.log(`Changing admin to ${newAdmin}`)
    const tx = await L2ERC721BridgeProxy.changeAdmin(newAdmin)
    const receipt = await tx.wait()
    console.log(
      `Changed admin of the L2ERC721BridgeProxy: ${receipt.transactionHash}`
    )
  }
}

deployFn.tags = ['L2ERC721BridgeImplementation']
deployFn.dependencies = ['L2ERC721BridgeProxy', 'L1ERC721BridgeProxy']

export default deployFn
