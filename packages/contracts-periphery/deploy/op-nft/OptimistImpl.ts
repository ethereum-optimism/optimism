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

  console.log(`Deploying Optimist implementation with ${deployer}`)

  const Deployment__AttestationStation = await hre.deployments.get(
    'AttestationStationProxy'
  )
  const attestationStationAddress = Deployment__AttestationStation.address

  console.log(`Using ${attestationStationAddress} as the AttestationStation`)
  console.log(`Using ${deployConfig.attestorAddress} as ATTESTOR`)

  const { deploy } = await hre.deployments.deterministic('Optimist', {
    salt: hre.ethers.utils.solidityKeccak256(['string'], ['Optimist']),
    from: deployer,
    args: [
      deployConfig.optimistName,
      deployConfig.optimistSymbol,
      deployConfig.attestorAddress,
      attestationStationAddress,
    ],
    log: true,
  })

  await deploy()
}

deployFn.tags = ['Optimist', 'OptimistEnvironment']

export default deployFn
