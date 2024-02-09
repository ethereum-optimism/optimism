import { DeployFunction } from 'hardhat-deploy/dist/types'
import '@eth-optimism/hardhat-deploy-config'

import { deploy } from '../scripts/deploy-utils'

const deployFn: DeployFunction = async (hre) => {
  const { deployer } = await hre.getNamedAccounts()
  const isLiveDeployer =
    deployer.toLowerCase() === hre.deployConfig.controller.toLowerCase()

  const portalGuardian = hre.deployConfig.portalGuardian
  const portalGuardianCode = await hre.ethers.provider.getCode(portalGuardian)
  if (portalGuardianCode === '0x') {
    console.log(
      `WARNING: setting OptimismPortal.GUARDIAN to ${portalGuardian} and it has no code`
    )
    if (!isLiveDeployer) {
      throw new Error(
        'Do not deploy to production networks without the GUARDIAN being a contract'
      )
    }
  }

  // Deploy the OptimismPortal implementation as paused to
  // ensure that users do not interact with it and instead
  // interact with the proxied contract.
  // The `portalGuardian` is set at the GUARDIAN.
  await deploy({
    hre,
    name: 'OptimismPortal',
    args: [],
  })
}

deployFn.tags = ['OptimismPortalImpl', 'setup', 'l1']

export default deployFn
