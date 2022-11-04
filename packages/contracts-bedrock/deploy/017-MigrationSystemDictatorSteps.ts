import { ethers } from 'ethers'
import { DeployFunction } from 'hardhat-deploy/dist/types'
import '@eth-optimism/hardhat-deploy-config'
import 'hardhat-deploy'
import { awaitCondition } from '@eth-optimism/core-utils'

import { getContractFromArtifact } from '../src/deploy-utils'

const deployFn: DeployFunction = async (hre) => {
  const { deployer } = await hre.getNamedAccounts()

  let isLiveDeployer = false
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
      isLiveDeployer = true
    } else {
      throw new Error(
        `controller address MUST NOT be the deployer on live networks`
      )
    }
  }

  const MigrationSystemDictator = await getContractFromArtifact(
    hre,
    'MigrationSystemDictator',
    {
      signerOrProvider: deployer,
    }
  )

  // Transfer ownership of the ProxyAdmin to the MigrationSystemDictator
  const ProxyAdmin = await getContractFromArtifact(hre, 'ProxyAdmin', {
    signerOrProvider: deployer,
  })
  if ((await ProxyAdmin.owner()) !== MigrationSystemDictator.address) {
    console.log(
      `Transferring proxy admin ownership to the MigrationSystemDictator`
    )
    await ProxyAdmin.setOwner(MigrationSystemDictator.address)
  } else {
    console.log(`Proxy admin already owned by the MigrationSystemDictator`)
  }

  // Transfer ownership of the AddressManager to MigrationSystemDictator.
  const AddressManager = await getContractFromArtifact(
    hre,
    'Lib_AddressManager',
    {
      signerOrProvider: deployer,
    }
  )
  if ((await AddressManager.owner()) !== MigrationSystemDictator.address) {
    if (isLiveDeployer) {
      console.log(
        `Transferring ownership of AddressManager to the MigrationSystemDictator...`
      )
      await AddressManager.transferOwnership(MigrationSystemDictator.address)
    } else {
      console.log(
        `Please transfer ownership of the AddressManager to the MigrationSystemDictator located at: ${MigrationSystemDictator.address}`
      )
    }
    await awaitCondition(async () => {
      const owner = await AddressManager.owner()
      return owner === MigrationSystemDictator.address
    })
  } else {
    console.log(`AddressManager already owned by the MigrationSystemDictator`)
  }

  // Transfer ownership of the L1CrossDomainMessenger to MigrationSystemDictator.
  const L1CrossDomainMessenger = await getContractFromArtifact(
    hre,
    'Proxy__OVM_L1CrossDomainMessenger',
    {
      iface: 'L1CrossDomainMessenger',
      signerOrProvider: deployer,
    }
  )
  if (
    (await L1CrossDomainMessenger.owner()) !== MigrationSystemDictator.address
  ) {
    if (isLiveDeployer) {
      console.log(
        `Transferring ownership of L1CrossDomainMessenger to the MigrationSystemDictator...`
      )
      await L1CrossDomainMessenger.transferOwnership(
        MigrationSystemDictator.address
      )
    } else {
      console.log(
        `Please transfer ownership of the L1CrossDomainMessenger to the MigrationSystemDictator located at: ${MigrationSystemDictator.address}`
      )
    }
    await awaitCondition(async () => {
      const owner = await L1CrossDomainMessenger.owner()
      return owner === MigrationSystemDictator.address
    })
  } else {
    console.log(
      `L1CrossDomainMessenger already owned by the MigrationSystemDictator`
    )
  }

  // Transfer ownership of the L1StandardBridge (proxy) to MigrationSystemDictator.
  const L1StandardBridge = await getContractFromArtifact(
    hre,
    'Proxy__OVM_L1StandardBridge'
  )
  if ((await L1StandardBridge.owner()) !== MigrationSystemDictator.address) {
    if (isLiveDeployer) {
      console.log(
        `Transferring ownership of L1StandardBridge to the MigrationSystemDictator...`
      )
      const L1StandardBridgeWithSigner = await getContractFromArtifact(
        hre,
        'Proxy__OVM_L1StandardBridge',
        {
          signerOrProvider: deployer,
        }
      )
      await L1StandardBridgeWithSigner.setOwner(MigrationSystemDictator.address)
    } else {
      console.log(
        `Please transfer ownership of the L1StandardBridge (proxy) to the MigrationSystemDictator located at: ${MigrationSystemDictator.address}`
      )
    }
    await awaitCondition(async () => {
      const owner = await L1StandardBridge.callStatic.getOwner()
      return owner === MigrationSystemDictator.address
    })
  } else {
    console.log(`L1StandardBridge already owned by the MigrationSystemDictator`)
  }

  for (let i = 1; i <= 6; i++) {
    if (isLiveDeployer) {
      console.log(`Executing step ${i}...`)
      await MigrationSystemDictator[`step${i}`]()
    } else {
      console.log(`Please execute step ${i}...`)
    }

    await awaitCondition(async () => {
      const step = await MigrationSystemDictator.currentStep()
      return step.toNumber() === i + 1
    })
  }
}

deployFn.tags = ['MigrationSystemDictatorSteps', 'migration']

export default deployFn
