/* Imports: External */
import { DeployFunction } from 'hardhat-deploy/dist/types'
import '@eth-optimism/hardhat-deploy-config'
import '@nomiclabs/hardhat-ethers'
import 'hardhat-deploy'

const deployFn: DeployFunction = async (hre) => {
  const { deployer } = await hre.getNamedAccounts()
  const { getAddress } = hre.ethers.utils

  const mainnetDeployer = getAddress(
    '0x53A6eecC2dD4795Fcc68940ddc6B4d53Bd88Bd9E'
  )
  const goerliDeployer = getAddress(
    '0x5c679a57e018f5f146838138d3e032ef4913d551'
  )

  // Deploy the L2ERC721BridgeProxy as a predeploy address.
  // A special deployer account must be used.
  if (hre.network.name === 'optimism') {
    if (getAddress(deployer) !== mainnetDeployer) {
      throw new Error(`Incorrect deployer: ${deployer}`)
    }
  } else if (hre.network.name === 'optimism-goerli') {
    if (getAddress(deployer) !== goerliDeployer) {
      throw new Error(`Incorrect deployer: ${deployer}`)
    }
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
  if (
    hre.network.name === 'optimism' ||
    hre.network.name === 'optimism-goerli'
  ) {
    const code = await hre.ethers.provider.getCode(
      '0x4200000000000000000000000000000000000014'
    )
    if (code === '0x') {
      throw new Error('Code is not set at expected predeploy address')
    }
    console.log(
      'L2ERC721BridgeProxy deployed to 0x4200000000000000000000000000000000000014'
    )
  } else {
    const Deployment__L2ERC721BridgeProxy = await hre.deployments.get(
      'L2ERC721BridgeProxy'
    )
    console.log(
      `L2ERC721BridgeProxy deployed to ${Deployment__L2ERC721BridgeProxy.address}`
    )
  }
}

deployFn.tags = ['L2ERC721BridgeProxy']

export default deployFn
