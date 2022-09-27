/* Imports: External */
import { DeployFunction } from 'hardhat-deploy/dist/types'
import { HardhatRuntimeEnvironment } from 'hardhat/types'
import '@eth-optimism/hardhat-deploy-config'
import '@nomiclabs/hardhat-ethers'
import 'hardhat-deploy'
import fetch from 'node-fetch'

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

  const L1CrossDomainMessengerAddress =
    Deployment__L1CrossDomainMessengerProxy.address

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

  {
    // Upgrade the Proxy to the newly deployed implementation
    const tx = await L1ERC721BridgeProxy.upgradeTo(
      Deployment__L1ERC721Bridge.address
    )
    const receipt = await tx.wait()
    console.log(`L1ERC721BridgeProxy upgraded: ${receipt.transactionHash}`)
  }

  {
    if (
      hre.network.name === 'optimism' ||
      hre.network.name === 'optimism-goerli' ||
      hre.network.name === 'ops-l2'
    ) {
      let newAdmin: string
      if (hre.network.name === 'optimism') {
        newAdmin = '0x2501c477D0A35545a387Aa4A3EEe4292A9a8B3F0'
      } else if (hre.network.name === 'optimism-goerli') {
        newAdmin = '0xf80267194936da1E98dB10bcE06F3147D580a62e'
      } else {
        newAdmin = deployer
      }
      const tx = await L1ERC721BridgeProxy.changeAdmin(newAdmin)
      const receipt = await tx.wait()
      console.log(
        `L1ERC721BridgeProxy admin updated: ${receipt.transactionHash}`
      )
    }
  }
}

deployFn.tags = ['L1ERC721BridgeImplementation']
deployFn.dependencies = ['L1ERC721BridgeProxy']

export default deployFn
