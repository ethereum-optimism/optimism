/* Imports: External */
import { DeployFunction } from 'hardhat-deploy/dist/types'
import { HardhatRuntimeEnvironment } from 'hardhat/types'

import '@nomiclabs/hardhat-ethers'
import '@eth-optimism/hardhat-deploy-config'
import 'hardhat-deploy'
import type { DeployConfig } from '../../src'

const deployFn: DeployFunction = async (hre: HardhatRuntimeEnvironment) => {
  const deployConfig = hre.deployConfig as DeployConfig

  const { deployer } = await hre.getNamedAccounts()

  console.log(`Deploying OptimistInviter implementation with ${deployer}`)

  const Deployment__AttestationStation = await hre.deployments.get(
    'AttestationStationProxy'
  )
  const attestationStationAddress = Deployment__AttestationStation.address

  console.log(`Using ${attestationStationAddress} as the ATTESTATION_STATION`)
  console.log(
    `Using ${deployConfig.optimistInviterInviteGranter} as INVITE_GRANTER`
  )

  const { deploy } = await hre.deployments.deterministic('OptimistInviter', {
    salt: hre.ethers.utils.solidityKeccak256(['string'], ['OptimistInviter']),
    from: deployer,
    args: [
      deployConfig.optimistInviterInviteGranter,
      attestationStationAddress,
    ],
    log: true,
  })

  await deploy()
}

deployFn.tags = ['OptimistInviterImpl', 'OptimistEnvironment']
deployFn.dependencies = ['AttestationStationProxy']

export default deployFn
