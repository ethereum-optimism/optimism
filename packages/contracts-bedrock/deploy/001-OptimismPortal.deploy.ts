/* Imports: Internal */
import { DeployFunction } from 'hardhat-deploy/dist/types'
import 'hardhat-deploy'
import '@nomiclabs/hardhat-ethers'
import '@eth-optimism/hardhat-deploy-config'

const deployFn: DeployFunction = async (hre) => {
  const { deploy, get } = hre.deployments
  const { deployer } = await hre.getNamedAccounts()
  const { deployConfig } = hre

  const oracle = await get('L2OutputOracle')

  await deploy('OptimismPortal', {
    from: deployer,
    args: [oracle.address, 2],
    log: true,
    waitConfirmations: deployConfig.deploymentWaitConfirmations,
  })

  const portal = await hre.deployments.get('OptimismPortal')

  const OptimismPortal = await hre.ethers.getContractAt(
    'OptimismPortal',
    portal.address
  )

  const l2Oracle = await OptimismPortal.L2_ORACLE()
  if (l2Oracle !== oracle.address) {
    throw new Error('L2 Oracle mismatch')
  }
}

deployFn.tags = ['OptimismPortal']

export default deployFn
