/* Imports: External */
import { DeployFunction } from 'hardhat-deploy/dist/types'
import '@eth-optimism/hardhat-deploy-config'
import '@nomiclabs/hardhat-ethers'
import 'hardhat-deploy'

const deployFn: DeployFunction = async (hre) => {
  const { deployer } = await hre.getNamedAccounts()
  const { deploy } = hre.deployments
  const { getAddress } = hre.ethers.utils

  // Get the address of the currently deployed L1CrossDomainMessenger.
  // This should be the address of the proxy
  const Deployment__L1CrossDomainMessenger = await hre.deployments.get(
    'Proxy__OVM_L1CrossDomainMessenger'
  )
  const L1CrossDomainMessengerAddress =
    Deployment__L1CrossDomainMessenger.address

  const predeploy = '0x4200000000000000000000000000000000000014'
  // Deploy the L1ERC721Bridge. The arguments are
  // - messenger
  // - otherBridge
  // Since this is the L1ERC721Bridge, the otherBridge is the
  // predeploy address
  await deploy('L1ERC721Bridge', {
    from: deployer,
    args: [L1CrossDomainMessengerAddress, predeploy],
    log: true,
    waitConfirmations: 1,
  })

  const Deployment__L1ERC721Bridge = await hre.deployments.get('L1ERC721Bridge')
  console.log(
    `L1ERC721Bridge deployed to ${Deployment__L1ERC721Bridge.address}`
  )

  const L1ERC721Bridge = await hre.ethers.getContractAt(
    'L1ERC721Bridge',
    Deployment__L1ERC721Bridge.address
  )

  // Check to make sure that it was initialized correctly
  const messenger = await L1ERC721Bridge.messenger()
  if (getAddress(messenger) !== getAddress(L1CrossDomainMessengerAddress)) {
    throw new Error(`L1ERC721Bridge.messenger misconfigured`)
  }

  const otherBridge = await L1ERC721Bridge.otherBridge()
  if (getAddress(otherBridge) !== getAddress(predeploy)) {
    throw new Error('L1ERC721Bridge.otherBridge misconfigured')
  }
}

deployFn.tags = ['L1ERC721BridgeImplementation']
deployFn.dependencies = ['L1ERC721BridgeProxy']

export default deployFn
