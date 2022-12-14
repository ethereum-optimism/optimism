import { DeployFunction } from 'hardhat-deploy/dist/types'
import { HardhatRuntimeEnvironment } from 'hardhat/types'
import '@eth-optimism/hardhat-deploy-config'
import '@nomiclabs/hardhat-ethers'
import 'hardhat-deploy'
import {
  assertContractVariable,
  deploy,
} from '@eth-optimism/contracts-bedrock/src/deploy-utils'
import { utils } from 'ethers'

const { getAddress } = utils

/**
 * Deploys the AttestationStationProxy
 */
const deployFn: DeployFunction = async (hre: HardhatRuntimeEnvironment) => {
  const { deployer } = await hre.getNamedAccounts()

  console.log(`Deploying AttestationStationProxy with ${deployer}`)

  const Deployment__AttestationStation = await hre.deployments.get(
    'AttestationStation'
  )

  await deploy({
    hre,
    name: 'AttestationStationProxy',
    contract: 'Proxy',
    args: [deployer],
    postDeployAction: async (contract) => {
      await assertContractVariable(contract, 'admin', deployer)
    },
  })

  const Deployment__AttestationStationProxy = await hre.deployments.get(
    'AttestationStationProxy'
  )
  const addr = Deployment__AttestationStationProxy.address
  console.log(`AttestationStationProxy deployed to ${addr}`)
  console.log(
    `Using AttestationStation implementation at ${Deployment__AttestationStation.address}`
  )

  const Proxy = await hre.ethers.getContractAt('Proxy', addr)

  const implementation = await Proxy.callStatic.implementation()
  console.log(`implementation is set to ${implementation}`)
  if (
    getAddress(implementation) !==
    getAddress(Deployment__AttestationStation.address)
  ) {
    console.log('implementation not set to AttestationStation contract')
    console.log(
      `Setting implementation to ${Deployment__AttestationStation.address}`
    )

    const tx = await Proxy.upgradeTo(Deployment__AttestationStation.address)
    const receipt = await tx.wait()
    console.log(`implementation set in tx ${receipt.transactionHash}`)
  } else {
    console.log('implementation already set to AttestationStation contract')
  }

  const l2ProxyOwnerAddress = hre.deployConfig.l2ProxyOwnerAddress
  const admin = await Proxy.callStatic.admin()
  console.log(`admin is set to ${admin}`)
  if (getAddress(admin) !== getAddress(l2ProxyOwnerAddress)) {
    console.log('admin not set correctly')
    console.log(`Setting admin to ${l2ProxyOwnerAddress}`)

    const tx = await Proxy.changeAdmin(l2ProxyOwnerAddress)
    const receipt = await tx.wait()
    console.log(`admin set in ${receipt.transactionHash}`)
  } else {
    console.log('admin already set to L2 Proxy Owner Address')
  }
  console.log('Contract deployment complete')
}

deployFn.tags = ['AttestationStationProxy']
deployFn.dependencies = ['AttestationStation']

export default deployFn
