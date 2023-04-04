import { DeployFunction } from 'hardhat-deploy/dist/types'
import { HardhatRuntimeEnvironment } from 'hardhat/types'
import '@eth-optimism/hardhat-deploy-config'
import '@nomiclabs/hardhat-ethers'
import 'hardhat-deploy'
import { assertContractVariable } from '@eth-optimism/contracts-bedrock/src/deploy-utils'
import { ethers, utils } from 'ethers'

import type { DeployConfig } from '../../src'

const { getAddress } = utils

/**
 * Deploys the AttestationStationProxy
 */
const deployFn: DeployFunction = async (hre: HardhatRuntimeEnvironment) => {
  const deployConfig = hre.deployConfig as DeployConfig

  const { deployer } = await hre.getNamedAccounts()
  const ddd = hre.deployConfig.ddd

  if (getAddress(deployer) !== getAddress(ddd)) {
    throw new Error('Must deploy with the ddd')
  }

  console.log(`Deploying AttestationStationProxy with ${deployer}`)

  const Deployment__AttestationStation = await hre.deployments.get(
    'AttestationStation'
  )

  const { deploy } = await hre.deployments.deterministic(
    'AttestationStationProxy',
    {
      salt: hre.ethers.utils.solidityKeccak256(
        ['string'],
        ['AttestationStationProxy']
      ),
      contract: 'Proxy',
      from: deployer,
      args: [deployer],
      log: true,
    }
  )

  await deploy()

  const Deployment__AttestationStationProxy = await hre.deployments.get(
    'AttestationStationProxy'
  )
  const addr = Deployment__AttestationStationProxy.address
  console.log(`AttestationStationProxy deployed to ${addr}`)
  console.log(
    `Using AttestationStation implementation at ${Deployment__AttestationStation.address}`
  )

  const Proxy = await hre.ethers.getContractAt('Proxy', addr)

  const AttestationStation = await hre.ethers.getContractAt(
    'AttestationStation',
    addr
  )

  const implementation = await Proxy.connect(
    ethers.constants.AddressZero
  ).callStatic.implementation()
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

  const l2ProxyOwnerAddress = deployConfig.l2ProxyOwnerAddress
  const admin = await Proxy.connect(
    ethers.constants.AddressZero
  ).callStatic.admin()
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

  await assertContractVariable(Proxy, 'admin', l2ProxyOwnerAddress)
  await assertContractVariable(AttestationStation, 'version', '1.1.0')
}

deployFn.tags = ['AttestationStationProxy', 'OptimistEnvironment']
deployFn.dependencies = ['AttestationStation']

export default deployFn
