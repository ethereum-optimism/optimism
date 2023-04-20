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

  console.log(`Deploying OptimistAllowlist implementation with ${deployer}`)

  const Deployment__AttestationStation = await hre.deployments.get(
    'AttestationStationProxy'
  )

  const Deployment__OptimistInviter = await hre.deployments.get(
    'OptimistInviterProxy'
  )

  const attestationStationAddress = Deployment__AttestationStation.address
  const optimistInviterAddress = Deployment__OptimistInviter.address

  console.log(`Using ${attestationStationAddress} as the ATTESTATION_STATION`)
  console.log(
    `Using ${deployConfig.optimistAllowlistAllowlistAttestor} as ALLOWLIST_ATTESTOR`
  )
  console.log(
    `Using ${deployConfig.optimistAllowlistCoinbaseQuestAttestor} as COINBASE_QUEST_ATTESTOR`
  )
  console.log(`Using ${optimistInviterAddress} as OPTIMIST_INVITER`)

  const { deploy } = await hre.deployments.deterministic('OptimistAllowlist', {
    salt: hre.ethers.utils.solidityKeccak256(['string'], ['OptimistAllowlist']),
    from: deployer,
    args: [
      attestationStationAddress,
      deployConfig.optimistAllowlistAllowlistAttestor,
      deployConfig.optimistAllowlistCoinbaseQuestAttestor,
      optimistInviterAddress,
    ],
    log: true,
  })

  await deploy()
}

deployFn.tags = ['OptimistAllowlistImpl', 'OptimistEnvironment']
deployFn.dependencies = ['AttestationStationProxy', 'OptimistInviterProxy']

export default deployFn
