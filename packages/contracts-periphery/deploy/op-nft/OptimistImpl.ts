/* Imports: External */
import { DeployFunction } from 'hardhat-deploy/dist/types'
import { HardhatRuntimeEnvironment } from 'hardhat/types'
import '@nomiclabs/hardhat-ethers'
import '@eth-optimism/hardhat-deploy-config'
import 'hardhat-deploy'

const deployFn: DeployFunction = async (hre: HardhatRuntimeEnvironment) => {
  const { deployer } = await hre.getNamedAccounts()

  console.log(`Deploying Optimist implementation with ${deployer}`)

  const Deployment__AttestationStation = await hre.deployments.get(
    'AttestationStationProxy'
  )
  const attestationStationAddress = Deployment__AttestationStation.address

  console.log(`Using ${attestationStationAddress} as the AttestationStation`)
  console.log(`Using ${hre.deployConfig.attestorAddress} as ATTESTOR`)

  const { deploy } = await hre.deployments.deterministic('Optimist', {
    salt: hre.ethers.utils.solidityKeccak256(['string'], ['Optimist']),
    from: deployer,
    args: [
      hre.deployConfig.optimistName,
      hre.deployConfig.optimistSymbol,
      hre.deployConfig.attestorAddress,
      attestationStationAddress,
    ],
    log: true,
  })

  await deploy()
}

deployFn.tags = ['Optimist', 'OptimistEnvironment']

export default deployFn
