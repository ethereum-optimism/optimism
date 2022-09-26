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

  if (hre.network.name === 'optimism') {
    if (getAddress(deployer) !== mainnetDeployer) {
      throw new Error(`Incorrect deployer: ${deployer}`)
    }
    //
  } else if (hre.network.name === 'optimism-goerli') {
    if (getAddress(deployer) !== goerliDeployer) {
      throw new Error(`Incorrect deployer: ${deployer}`)
    }
  }

  await hre.deployments.deploy('L2ERC721BridgeProxy', {
    contract: 'Proxy',
    from: deployer,
    args: [],
    log: true,
  })
}

deployFn.tags = ['L2ERC721BridgeProxy']

export default deployFn
