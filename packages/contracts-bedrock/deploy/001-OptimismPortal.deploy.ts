/* Imports: Internal */
import { DeployFunction } from 'hardhat-deploy/dist/types'
import 'hardhat-deploy'
import '@nomiclabs/hardhat-ethers'
import '@eth-optimism/hardhat-deploy-config'

const deployFn: DeployFunction = async (hre) => {
  const { deploy, get } = hre.deployments
  const { deployer } = await hre.getNamedAccounts()
  const { deployConfig } = hre

  await deploy('OptimismPortalProxy', {
    contract: 'Proxy',
    from: deployer,
    args: [deployer],
    log: true,
    waitConfirmations: deployConfig.deploymentWaitConfirmations,
  })

  const oracle = await get('L2OutputOracle')

  await deploy('OptimismPortal', {
    from: deployer,
    args: [oracle.address, 2],
    log: true,
    waitConfirmations: deployConfig.deploymentWaitConfirmations,
  })

  const proxy = await hre.deployments.get('OptimismPortalProxy')
  const Proxy = await hre.ethers.getContractAt('Proxy', proxy.address)

  const OptimismPortal = await hre.ethers.getContractAt(
    'OptimismPortal',
    proxy.address
  )

  const portal = await hre.deployments.get('OptimismPortal')
  const tx = await Proxy.upgradeToAndCall(
    portal.address,
    OptimismPortal.interface.encodeFunctionData('initialize()')
  )
  await tx.wait()

  const l2Oracle = await OptimismPortal.L2_ORACLE()
  if (l2Oracle !== oracle.address) {
    throw new Error('L2 Oracle mismatch')
  }
}

deployFn.tags = ['OptimismPortal']

export default deployFn
