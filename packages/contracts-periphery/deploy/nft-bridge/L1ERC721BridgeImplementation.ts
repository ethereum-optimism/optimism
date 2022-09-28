/* Imports: External */
import { DeployFunction } from 'hardhat-deploy/dist/types'
import { HardhatRuntimeEnvironment } from 'hardhat/types'
import '@eth-optimism/hardhat-deploy-config'
import '@nomiclabs/hardhat-ethers'
import 'hardhat-deploy'
import fetch from 'node-fetch'

import {
  isTargetL1Network,
  predeploy,
  getProxyAdmin,
  validateERC721Bridge,
} from '../../src/nft-bridge-deploy-helpers'

// Handle the `ops` deployment
const getL1CrossDomainMessengerProxyDeployment = async (
  hre: HardhatRuntimeEnvironment
) => {
  const network = hre.network.name
  if (network === 'ops-l1') {
    const res = await fetch(
      'http://localhost:8080/deployments/local/Proxy__OVM_L1CrossDomainMessenger.json'
    )
    return res.json()
  } else {
    return hre.deployments.get('Proxy__OVM_L1CrossDomainMessenger')
  }
}

const deployFn: DeployFunction = async (hre) => {
  const { deployer } = await hre.getNamedAccounts()
  const { deploy } = hre.deployments
  const { getAddress } = hre.ethers.utils

  if (!isTargetL1Network(hre.network.name)) {
    console.log(`Deploying to unsupported network ${hre.network.name}`)
    return
  }

  console.log(`Deploying L1ERC721Bridge to ${hre.network.name}`)
  console.log(`Using deployer ${deployer}`)

  const Deployment__L1ERC721BridgeProxy = await hre.deployments.get(
    'L1ERC721BridgeProxy'
  )

  const L1ERC721BridgeProxy = await hre.ethers.getContractAt(
    'Proxy',
    Deployment__L1ERC721BridgeProxy.address
  )

  const admin = await L1ERC721BridgeProxy.callStatic.admin()
  if (getAddress(admin) !== getAddress(deployer)) {
    throw new Error('deployer is not proxy admin')
  }

  // Get the address of the currently deployed L1CrossDomainMessenger.
  // This should be the address of the proxy
  const Deployment__L1CrossDomainMessengerProxy =
    await getL1CrossDomainMessengerProxyDeployment(hre)

  const L1CrossDomainMessengerProxyAddress =
    Deployment__L1CrossDomainMessengerProxy.address

  // Deploy the L1ERC721Bridge. The arguments are
  // - messenger
  // - otherBridge
  // Since this is the L1ERC721Bridge, the otherBridge is the
  // predeploy address
  await deploy('L1ERC721Bridge', {
    from: deployer,
    args: [L1CrossDomainMessengerProxyAddress, predeploy],
    log: true,
    waitConfirmations: 1,
  })

  const Deployment__L1ERC721Bridge = await hre.deployments.get('L1ERC721Bridge')
  console.log(
    `L1ERC721Bridge deployed to ${Deployment__L1ERC721Bridge.address}`
  )

  await validateERC721Bridge(hre, Deployment__L1ERC721Bridge.address, {
    messenger: L1CrossDomainMessengerProxyAddress,
    otherBridge: predeploy,
  })

  {
    // Upgrade the Proxy to the newly deployed implementation
    const tx = await L1ERC721BridgeProxy.upgradeTo(
      Deployment__L1ERC721Bridge.address
    )
    const receipt = await tx.wait()
    console.log(`L1ERC721BridgeProxy upgraded: ${receipt.transactionHash}`)
  }

  {
    // Set the admin correctly
    const newAdmin = getProxyAdmin(hre.network.name)
    const tx = await L1ERC721BridgeProxy.changeAdmin(newAdmin)
    const receipt = await tx.wait()
    console.log(`L1ERC721BridgeProxy admin updated: ${receipt.transactionHash}`)
  }

  await validateERC721Bridge(hre, L1ERC721BridgeProxy.address, {
    messenger: L1CrossDomainMessengerProxyAddress,
    otherBridge: predeploy,
  })
}

deployFn.tags = ['L1ERC721BridgeImplementation']
deployFn.dependencies = ['L1ERC721BridgeProxy']

export default deployFn
