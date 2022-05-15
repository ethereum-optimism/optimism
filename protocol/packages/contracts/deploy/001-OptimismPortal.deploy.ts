/* Imports: Internal */
import { DeployFunction } from 'hardhat-deploy/dist/types'
import 'hardhat-deploy'

const deployFn: DeployFunction = async (hre) => {
  const { deploy, get } = hre.deployments
  const { deployer } = await hre.getNamedAccounts()
  const oracle = await get('L2OutputOracle')

  await deploy('OptimismPortal', {
    from: deployer,
    args: [oracle.address, 2],
    log: true,
    waitConfirmations: 1,
  })
}

deployFn.tags = ['OptimismPortal']

export default deployFn
