/* Imports: External */
import { DeployFunction } from 'hardhat-deploy/dist/types'

const deployFn: DeployFunction = async (hre) => {
  const Artifact__L1CrossDomainMessenger = await hre.deployments.get(
    'Proxy__OVM_L1CrossDomainMessenger'
  )

  const { deployer } = await hre.getNamedAccounts()
  const L2ERC721Bridge = await hre.companionNetworks['l2'].deployments.get(
    'L2ERC721Bridge'
  )

  await hre.deployments.deploy('L1ERC721Bridge', {
    from: deployer,
    args: [Artifact__L1CrossDomainMessenger.address, L2ERC721Bridge.address],
    log: true,
  })
}

deployFn.tags = ['nft-bridge', 'L1ERC721Bridge']

export default deployFn
