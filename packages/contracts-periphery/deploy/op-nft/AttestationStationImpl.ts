/* Imports: External */
import { DeployFunction } from 'hardhat-deploy/dist/types'
import { HardhatRuntimeEnvironment } from 'hardhat/types'
import '@nomiclabs/hardhat-ethers'
import '@eth-optimism/hardhat-deploy-config'
import 'hardhat-deploy'

const deployFn: DeployFunction = async (hre: HardhatRuntimeEnvironment) => {
  const { deployer } = await hre.getNamedAccounts()

  console.log(`Deploying AttestationStation with ${deployer}`)

  const { deploy } = await hre.deployments.deterministic('AttestationStation', {
    salt: hre.ethers.utils.solidityKeccak256(
      ['string'],
      ['AttestationStation']
    ),
    from: deployer,
    args: [],
    log: true,
  })

  await deploy()

  const Deployment__AttestationStation = await hre.deployments.get(
    'AttestationStation'
  )
  const addr = Deployment__AttestationStation.address
  console.log(`AttestationStation deployed to ${addr}`)
}

deployFn.tags = ['AttestationStation', 'OptimistEnvironment']

export default deployFn
