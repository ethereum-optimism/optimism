/* Imports: External */
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

const deployFn: DeployFunction = async (hre: HardhatRuntimeEnvironment) => {
  const { deployer } = await hre.getNamedAccounts()

  const Deployment__Optimist = await hre.deployments.get('Optimist')

  console.log(`Deploying OptimistProxy with ${deployer}`)

  await deploy({
    hre,
    name: 'OptimistProxy',
    contract: 'Proxy',
    args: [deployer],
    postDeployAction: async (contract) => {
      await assertContractVariable(contract, 'admin', deployer)
    },
  })

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

  const implementation = await Proxy.callStatic.implementation()
  console.log(`implementation set to ${implementation}`)
  if (getAddress(implementation) !== getAddress(Deployment__Optimist.address)) {
    console.log('implementation not set to Optimist contract')
    console.log(`Setting implementation to ${Deployment__Optimist.address}`)

    // Create the calldata for the call to `initialize()`
    const name = hre.deployConfig.optimistName
    const symbol = hre.deployConfig.optimistSymbol
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

  const l2ProxyOwnerAddress = hre.deployConfig.l2ProxyOwnerAddress
  const admin = await Proxy.callStatic.admin()
  console.log(`admin set to ${admin}`)
  if (getAddress(admin) === getAddress(l2ProxyOwnerAddress)) {
    console.log('admin not set correctly')
    console.log(`Setting admin to ${l2ProxyOwnerAddress}`)

    const tx = await Proxy.changeAdmin(l2ProxyOwnerAddress)
    const receipt = await tx.wait()
    console.log(`admin set in ${receipt.transactionHash}`)
  } else {
    console.log('admin already set to L2 multisig')
  }
}

deployFn.tags = ['OptimistProxy']
deployFn.dependencies = ['Optimist']

export default deployFn
