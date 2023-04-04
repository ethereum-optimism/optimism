/* Imports: External */
import { DeployFunction } from 'hardhat-deploy/dist/types'
import { HardhatRuntimeEnvironment } from 'hardhat/types'
import '@eth-optimism/hardhat-deploy-config'
import '@nomiclabs/hardhat-ethers'
import 'hardhat-deploy'
import { assertContractVariable } from '@eth-optimism/contracts-bedrock/src/deploy-utils'
import { ethers, utils } from 'ethers'

import type { DeployConfig } from '../../src'

const { getAddress } = utils

const deployFn: DeployFunction = async (hre: HardhatRuntimeEnvironment) => {
  const deployConfig = hre.deployConfig as DeployConfig

  const { deployer } = await hre.getNamedAccounts()
  const ddd = deployConfig.ddd

  if (getAddress(deployer) !== getAddress(ddd)) {
    throw new Error('Must deploy with the ddd')
  }

  const Deployment__Optimist = await hre.deployments.get('Optimist')

  console.log(`Deploying OptimistProxy with ${deployer}`)

  const { deploy } = await hre.deployments.deterministic('OptimistProxy', {
    salt: hre.ethers.utils.solidityKeccak256(['string'], ['OptimistProxy']),
    contract: 'Proxy',
    from: deployer,
    args: [deployer],
    log: true,
  })

  await deploy()

  const Deployment__OptimistProxy = await hre.deployments.get('OptimistProxy')
  console.log(`OptimistProxy deployed to ${Deployment__OptimistProxy.address}`)

  const Proxy = await hre.ethers.getContractAt(
    'Proxy',
    Deployment__OptimistProxy.address
  )

  const Optimist = await hre.ethers.getContractAt(
    'Optimist',
    Deployment__OptimistProxy.address
  )

  const implementation = await Proxy.connect(
    ethers.constants.AddressZero
  ).callStatic.implementation()
  console.log(`implementation set to ${implementation}`)
  if (getAddress(implementation) !== getAddress(Deployment__Optimist.address)) {
    console.log('implementation not set to Optimist contract')
    console.log(`Setting implementation to ${Deployment__Optimist.address}`)

    // Create the calldata for the call to `initialize()`
    const name = deployConfig.optimistName
    const symbol = deployConfig.optimistSymbol
    const calldata = Optimist.interface.encodeFunctionData('initialize', [
      name,
      symbol,
    ])

    const tx = await Proxy.upgradeToAndCall(
      Deployment__Optimist.address,
      calldata
    )
    const receipt = await tx.wait()
    console.log(`implementation set in ${receipt.transactionHash}`)
  } else {
    console.log('implementation already set to Optimist contract')
  }

  const l2ProxyOwnerAddress = deployConfig.l2ProxyOwnerAddress
  const admin = await Proxy.connect(
    ethers.constants.AddressZero
  ).callStatic.admin()
  console.log(`admin set to ${admin}`)
  if (getAddress(admin) !== getAddress(l2ProxyOwnerAddress)) {
    console.log('detected admin is not set')
    console.log(`Setting admin to ${l2ProxyOwnerAddress}`)

    const tx = await Proxy.changeAdmin(l2ProxyOwnerAddress)
    const receipt = await tx.wait()
    console.log(`admin set in ${receipt.transactionHash}`)
  } else {
    console.log('admin already set to proxy owner address')
  }

  const Deployment__AttestationStation = await hre.deployments.get(
    'AttestationStationProxy'
  )

  await assertContractVariable(Proxy, 'admin', l2ProxyOwnerAddress)
  await assertContractVariable(Optimist, 'name', deployConfig.optimistName)
  await assertContractVariable(Optimist, 'version', '1.0.0')
  await assertContractVariable(Optimist, 'symbol', deployConfig.optimistSymbol)
  await assertContractVariable(
    Optimist,
    'ATTESTOR',
    deployConfig.attestorAddress
  )
  await assertContractVariable(
    Optimist,
    'ATTESTATION_STATION',
    Deployment__AttestationStation.address
  )
}

deployFn.tags = ['OptimistProxy', 'OptimistEnvironment']
deployFn.dependencies = ['AttestationStationProxy', 'Optimist']

export default deployFn
