/* Imports: External */
import { DeployFunction } from 'hardhat-deploy/dist/types'
import { predeploys } from '@eth-optimism/contracts'

const deployFn: DeployFunction = async (hre) => {
  const { deployer } = await hre.getNamedAccounts()

  const L1ERC721Bridge = await hre.companionNetworks['l1'].deployments.get(
    'L1ERC721Bridge'
  )

  await hre.deployments.deploy('L2ERC721Bridge', {
    from: deployer,
    args: [predeploys.L2CrossDomainMessenger, L1ERC721Bridge.address],
    log: true,
  })
}

deployFn.tags = ['l2-nft-bridge', 'L2ERC721Bridge']

export default deployFn
