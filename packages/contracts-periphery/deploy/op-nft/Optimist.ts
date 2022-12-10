/* Imports: External */
import { DeployFunction } from 'hardhat-deploy/dist/types'

const deployFn: DeployFunction = async (hre) => {
  const { deployer } = await hre.getNamedAccounts()

  const Deployment__AttestationStation = await hre.companionNetworks[
    'l1'
  ].deployments.get('AttestationStation')

  const attestationStationAddress = Deployment__AttestationStation.address

  const { deploy } = await hre.deployments.deterministic('Optimist', {
    salt: hre.ethers.utils.solidityKeccak256(['string'], ['Optimist']),
    from: deployer,
    // make these more configurable
    args: [
      'OptimistNFT',
      'OPSBT',
      '0x8F0EBDaA1cF7106bE861753B0f9F5c0250fE0819',
      attestationStationAddress,
    ],
    log: true,
  })

  await deploy()
}

deployFn.tags = ['Optimist', 'OptimistEnvironment']

export default deployFn
