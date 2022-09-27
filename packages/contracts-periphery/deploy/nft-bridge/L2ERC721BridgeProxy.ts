/* Imports: External */
import { DeployFunction } from 'hardhat-deploy/dist/types'
import '@eth-optimism/hardhat-deploy-config'
import '@nomiclabs/hardhat-ethers'
import 'hardhat-deploy'

const predeploy = '0x4200000000000000000000000000000000000014'

const deployFn: DeployFunction = async (hre) => {
  const { deployer } = await hre.getNamedAccounts()
  const { getAddress } = hre.ethers.utils

  console.log(`Deploying L2ERC721BridgeProxy to ${hre.network.name}`)
  console.log(`Using deployer ${deployer}`)

  // Check to make sure that the Proxy has not been deployed yet
  const pre = await hre.ethers.provider.getCode(predeploy, 'latest')
  if (pre !== '0x') {
    console.log(`Code already deployed to ${predeploy}`)
    return
  }

  // A special deployer account must be used
  const mainnetDeployer = getAddress(
    '0x53A6eecC2dD4795Fcc68940ddc6B4d53Bd88Bd9E'
  )
  const goerliDeployer = getAddress(
    '0x5c679a57e018f5f146838138d3e032ef4913d551'
  )
  const localDeployer = getAddress('0xdfc82d475833a50de90c642770f34a9db7deb725')

  // Deploy the L2ERC721BridgeProxy as a predeploy address
  if (hre.network.name === 'optimism') {
    if (getAddress(deployer) !== mainnetDeployer) {
      throw new Error(`Incorrect deployer: ${deployer}`)
    }
  } else if (hre.network.name === 'optimism-goerli') {
    if (getAddress(deployer) !== goerliDeployer) {
      throw new Error(`Incorrect deployer: ${deployer}`)
    }
  } else if (hre.network.name === 'ops-l2') {
    if (getAddress(deployer) !== localDeployer) {
      throw new Error(`Incorrect deployer: ${deployer}`)
    }
  } else {
    throw new Error(`Unknown network: ${hre.network.name}`)
  }

  // Set the deployer as the admin of the Proxy. This is
  // temporary, the admin will be updated when deploying
  // the implementation
  await hre.deployments.deploy('L2ERC721BridgeProxy', {
    contract: 'Proxy',
    from: deployer,
    args: [deployer],
    log: true,
    waitConfirmations: 1,
  })

  // Check that the Proxy was deployed to the correct address
  const code = await hre.ethers.provider.getCode(predeploy)
  if (code === '0x') {
    throw new Error('Code is not set at expected predeploy address')
  }
  console.log(`L2ERC721BridgeProxy deployed to ${predeploy}`)
}

deployFn.tags = ['L2ERC721BridgeProxy']

export default deployFn
