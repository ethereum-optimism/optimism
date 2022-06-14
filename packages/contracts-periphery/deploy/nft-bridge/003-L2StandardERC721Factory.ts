/* Imports: External */
import { DeployFunction } from 'hardhat-deploy/dist/types'

const deployFn: DeployFunction = async (hre) => {
  const { deployer } = await hre.getNamedAccounts()
  const L2ERC721Bridge = await hre.deployments.get('L2ERC721Bridge')

  await hre.deployments.deploy('L2StandardERC721Factory', {
    from: deployer,
    args: [L2ERC721Bridge.address],
    log: true,
  })
}

deployFn.tags = ['l2-nft-bridge', 'L2StandardERC721Factory']

export default deployFn
