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

  const Deployment__AttestationStationProxy = await hre.deployments.get(
    'AttestationStationProxy'
  )
  const attestationStationAddress = Deployment__AttestationStationProxy.address
  console.log(`Using ${attestationStationAddress} as the ATTESTATION_STATION`)
  console.log(
    `Using ${deployConfig.optimistBaseUriAttestorAddress} as BASE_URI_ATTESTOR`
  )

  const Deployment__OptimistAllowlistProxy = await hre.deployments.get(
    'OptimistAllowlistProxy'
  )
  const optimistAllowlistAddress = Deployment__OptimistAllowlistProxy.address

  const { deploy } = await hre.deployments.deterministic('Optimist', {
    salt: hre.ethers.utils.solidityKeccak256(['string'], ['Optimist']),
    from: deployer,
    args: [
      deployConfig.optimistName,
      deployConfig.optimistSymbol,
      deployConfig.optimistBaseUriAttestorAddress,
      attestationStationAddress,
      optimistAllowlistAddress,
    ],
    log: true,
  })

  await deploy()
}

deployFn.tags = ['OptimistImpl', 'OptimistEnvironment']
deployFn.dependencies = ['AttestationStationProxy', 'OptimistAllowlistProxy']

export default deployFn
