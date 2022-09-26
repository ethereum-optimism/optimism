/* Imports: External */
import { DeployFunction } from 'hardhat-deploy/dist/types'
import '@eth-optimism/hardhat-deploy-config'
import '@nomiclabs/hardhat-ethers'
import 'hardhat-deploy'
import { predeploys } from '@eth-optimism/contracts'

const deployFn: DeployFunction = async (hre) => {
  const { deployer } = await hre.getNamedAccounts()
  const { getAddress } = hre.ethers.utils

  const Deployment__L1ERC721Bridge = await hre.deployments.get(
    'L1ERC721BridgeProxy'
  )

  const L2ERC721BridgeProxy = await hre.ethers.getContractAt(
    'Proxy',
    '0x4200000000000000000000000000000000000014'
  )

  // Check to make sure that the admin of the proxy is the deployer.
  // The deployer of the L2ERC721Bridge should be the same as the
  // admin of the L2ERC721BridgeProxy so that it is easy to upgrade
  // the implementation. The admin is then changed depending on the
  // network after the L2ERC721BridgeProxy is upgraded to the implementation
  const admin = L2ERC721BridgeProxy.admin()
  if (getAddress(admin) !== getAddress(deployer)) {
    throw new Error(`Unexpected admin ${admin}`)
  }

  // Deploy the L2ERC721Bridge implementation
  await hre.deployments.deploy('L2ERC721Bridge', {
    from: deployer,
    args: [
      predeploys.L2CrossDomainMessenger,
      Deployment__L1ERC721Bridge.address,
    ],
    log: true,
    waitConfirmations: 1,
  })

  const Deployment__L2ERC721Bridge = await hre.deployments.get('L2ERC721Bridge')
  console.log(
    `L2ERC721Bridge deployed to ${Deployment__L2ERC721Bridge.address}`
  )

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

  {
    // On production networks transfer the admin privilege to the
    // appropriate address
    if (
      hre.network.name === 'optimism' ||
      hre.network.name === 'optimism-goerli'
    ) {
      let owner: string
      if (hre.network.name === 'optimism') {
        // L2 Foundation Multisig
        owner = '0x2501c477D0A35545a387Aa4A3EEe4292A9a8B3F0'
      } else if (hre.network.name === 'optimism-goerli') {
        // Goerli admin key
        owner = '0xf80267194936da1E98dB10bcE06F3147D580a62e'
      }

      const tx = await L2ERC721BridgeProxy.changeAdmin(owner)
      const receipt = await tx.wait()
      console.log(
        `Changed admin of the L2ERC721BridgeProxy: ${receipt.transactionhash}`
      )
    }
  }
}

deployFn.tags = ['L2ERC721BridgeImplementation']
deployFn.dependencies = ['L2ERC721BridgeProxy', 'L1ERC721BridgeProxy']

export default deployFn
