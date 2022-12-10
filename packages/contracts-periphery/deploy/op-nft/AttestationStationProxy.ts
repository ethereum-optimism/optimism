/* Imports: External */
import { DeployFunction } from 'hardhat-deploy/dist/types'
import '@eth-optimism/hardhat-deploy-config'
import '@nomiclabs/hardhat-ethers'
import 'hardhat-deploy'

const deployFn: DeployFunction = async (hre) => {
  const { deployer } = await hre.getNamedAccounts()
  const { deploy } = hre.deployments

  console.log(`Deploying AttestationStationProxy to ${hre.network.name}`)
  console.log(`Using deployer ${deployer}`)

  await deploy('AttestationStationProxy', {
    contract: 'Proxy',
    from: deployer,
    args: [deployer],
    log: true,
    waitConfirmations: 1,
  })

  const Deployment__AttestationStationProxy = await hre.deployments.get(
    'AttestationStationProxy'
  )
  console.log(
    `AttestationStationProxy deployed to ${Deployment__AttestationStationProxy.address}`
  )
}

deployFn.tags = ['AttestationStationProxy']

export default deployFn
