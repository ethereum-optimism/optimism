import { ethers } from 'ethers'
import { DeployFunction } from 'hardhat-deploy/dist/types'
import { awaitCondition } from '@eth-optimism/core-utils'
import '@eth-optimism/hardhat-deploy-config'
import 'hardhat-deploy'

import {
  assertDictatorConfig,
  makeDictatorConfig,
  getContractsFromArtifacts,
} from '../src/deploy-utils'

const deployFn: DeployFunction = async (hre) => {
  const { deployer } = await hre.getNamedAccounts()

  let controller = hre.deployConfig.controller
  if (controller === ethers.constants.AddressZero) {
    if (hre.network.config.live === false) {
      console.log(`WARNING!!!`)
      console.log(`WARNING!!!`)
      console.log(`WARNING!!!`)
      console.log(`WARNING!!! A controller address was not provided.`)
      console.log(
        `WARNING!!! Make sure you are ONLY doing this on a test network.`
      )
      controller = deployer
    } else {
      throw new Error(
        `controller address MUST NOT be the deployer on live networks`
      )
    }
  }

  let finalOwner = hre.deployConfig.finalSystemOwner
  if (finalOwner === ethers.constants.AddressZero) {
    if (hre.network.config.live === false) {
      console.log(`WARNING!!!`)
      console.log(`WARNING!!!`)
      console.log(`WARNING!!!`)
      console.log(`WARNING!!! A proxy admin owner address was not provided.`)
      console.log(
        `WARNING!!! Make sure you are ONLY doing this on a test network.`
      )
      finalOwner = deployer
    } else {
      throw new Error(`must specify the finalSystemOwner on live networks`)
    }
  }

  // Load the contracts we need to interact with.
  const [
    SystemDictator,
    SystemDictatorProxy,
    SystemDictatorProxyWithSigner,
    SystemDictatorImpl,
  ] = await getContractsFromArtifacts(hre, [
    {
      name: 'SystemDictatorProxy',
      iface: 'SystemDictator',
      signerOrProvider: deployer,
    },
    {
      name: 'SystemDictatorProxy',
    },
    {
      name: 'SystemDictatorProxy',
      signerOrProvider: deployer,
    },
    {
      name: 'SystemDictator',
      signerOrProvider: deployer,
    },
  ])

  // Load the dictator configuration.
  const config = await makeDictatorConfig(hre, controller, finalOwner, false)

  // Update the implementation if necessary.
  if (
    (await SystemDictatorProxy.callStatic.implementation({
      from: ethers.constants.AddressZero,
    })) !== SystemDictatorImpl.address
  ) {
    console.log('Upgrading the SystemDictator proxy...')

    // Upgrade and initialize the proxy.
    await SystemDictatorProxyWithSigner.upgradeToAndCall(
      SystemDictatorImpl.address,
      SystemDictatorImpl.interface.encodeFunctionData('initialize', [config])
    )

    // Wait for the transaction to execute properly.
    await awaitCondition(
      async () => {
        return (
          (await SystemDictatorProxy.callStatic.implementation({
            from: ethers.constants.AddressZero,
          })) === SystemDictatorImpl.address
        )
      },
      30000,
      1000
    )

    // Verify that the contract was initialized correctly.
    await assertDictatorConfig(SystemDictator, config)
  }

  // Update the owner if necessary.
  if (
    (await SystemDictatorProxy.callStatic.admin({
      from: ethers.constants.AddressZero,
    })) !== controller
  ) {
    console.log('Transferring ownership of the SystemDictator proxy...')

    // Transfer ownership to the controller address.
    await SystemDictatorProxyWithSigner.transferOwnership(controller)

    // Wait for the transaction to execute properly.
    await awaitCondition(
      async () => {
        return (
          (await SystemDictatorProxy.callStatic.admin({
            from: ethers.constants.AddressZero,
          })) === controller
        )
      },
      30000,
      1000
    )
  }
}

deployFn.tags = ['SystemDictatorImpl']

export default deployFn
