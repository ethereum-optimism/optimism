import assert from 'assert'

import { ethers } from 'ethers'
import { DeployFunction } from 'hardhat-deploy/dist/types'
import { awaitCondition } from '@eth-optimism/core-utils'
import '@eth-optimism/hardhat-deploy-config'
import 'hardhat-deploy'
import '@nomiclabs/hardhat-ethers'

import {
  assertContractVariable,
  getContractsFromArtifacts,
  getDeploymentAddress,
  doStep,
  jsonifyTransaction,
} from '../src/deploy-utils'

const deployFn: DeployFunction = async (hre) => {
  const { deployer } = await hre.getNamedAccounts()

  // Set up required contract references.
  const [
    SystemDictator,
    ProxyAdmin,
    AddressManager,
    L1StandardBridgeProxy,
    L1StandardBridgeProxyWithSigner,
    L1ERC721BridgeProxy,
    L1ERC721BridgeProxyWithSigner,
    SystemConfigProxy,
  ] = await getContractsFromArtifacts(hre, [
    {
      name: 'SystemDictatorProxy',
      iface: 'SystemDictator',
      signerOrProvider: deployer,
    },
    {
      name: 'ProxyAdmin',
      signerOrProvider: deployer,
    },
    {
      name: 'Lib_AddressManager',
      signerOrProvider: deployer,
    },
    {
      name: 'Proxy__OVM_L1StandardBridge',
    },
    {
      name: 'Proxy__OVM_L1StandardBridge',
      signerOrProvider: deployer,
    },
    {
      name: 'L1ERC721BridgeProxy',
    },
    {
      name: 'L1ERC721BridgeProxy',
      signerOrProvider: deployer,
    },
    {
      name: 'SystemConfigProxy',
      iface: 'SystemConfig',
      signerOrProvider: deployer,
    },
  ])

  // If we have the key for the controller then we don't need to wait for external txns.
  const isLiveDeployer =
    deployer.toLowerCase() === hre.deployConfig.controller.toLowerCase()

  // Transfer ownership of the ProxyAdmin to the SystemDictator.
  if ((await ProxyAdmin.owner()) !== SystemDictator.address) {
    console.log(`Setting ProxyAdmin owner to MSD`)
    await ProxyAdmin.transferOwnership(SystemDictator.address)
  } else {
    console.log(`Proxy admin already owned by MSD`)
  }

  // We don't need to transfer proxy addresses if we're already beyond the proxy transfer step.
  const needsProxyTransfer =
    (await SystemDictator.currentStep()) <=
    (await SystemDictator.PROXY_TRANSFER_STEP())

  // Transfer ownership of the AddressManager to SystemDictator.
  if (
    needsProxyTransfer &&
    (await AddressManager.owner()) !== SystemDictator.address
  ) {
    if (isLiveDeployer) {
      console.log(`Setting AddressManager owner to MSD`)
      await AddressManager.transferOwnership(SystemDictator.address)
    } else {
      const tx = await AddressManager.populateTransaction.transferOwnership(
        SystemDictator.address
      )
      console.log(`Please transfer AddressManager owner to MSD`)
      console.log(`AddressManager address: ${AddressManager.address}`)
      console.log(`MSD address: ${SystemDictator.address}`)
      console.log(`JSON:`)
      console.log(jsonifyTransaction(tx))
    }

    // Wait for the ownership transfer to complete.
    await awaitCondition(
      async () => {
        const owner = await AddressManager.owner()
        return owner === SystemDictator.address
      },
      30000,
      1000
    )
  } else {
    console.log(`AddressManager already owned by the SystemDictator`)
  }

  // Transfer ownership of the L1StandardBridge (proxy) to SystemDictator.
  if (
    needsProxyTransfer &&
    (await L1StandardBridgeProxy.callStatic.getOwner({
      from: ethers.constants.AddressZero,
    })) !== SystemDictator.address
  ) {
    if (isLiveDeployer) {
      console.log(`Setting L1StandardBridge owner to MSD`)
      await L1StandardBridgeProxyWithSigner.setOwner(SystemDictator.address)
    } else {
      const tx = await L1StandardBridgeProxy.populateTransaction.setOwner(
        SystemDictator.address
      )
      console.log(`Please transfer L1StandardBridge (proxy) owner to MSD`)
      console.log(
        `L1StandardBridgeProxy address: ${L1StandardBridgeProxy.address}`
      )
      console.log(`MSD address: ${SystemDictator.address}`)
      console.log(`JSON:`)
      console.log(jsonifyTransaction(tx))
    }

    // Wait for the ownership transfer to complete.
    await awaitCondition(
      async () => {
        const owner = await L1StandardBridgeProxy.callStatic.getOwner({
          from: ethers.constants.AddressZero,
        })
        return owner === SystemDictator.address
      },
      30000,
      1000
    )
  } else {
    console.log(`L1StandardBridge already owned by MSD`)
  }

  // Transfer ownership of the L1ERC721Bridge (proxy) to SystemDictator.
  if (
    needsProxyTransfer &&
    (await L1ERC721BridgeProxy.callStatic.admin({
      from: ethers.constants.AddressZero,
    })) !== SystemDictator.address
  ) {
    if (isLiveDeployer) {
      console.log(`Setting L1ERC721Bridge owner to MSD`)
      await L1ERC721BridgeProxyWithSigner.changeAdmin(SystemDictator.address)
    } else {
      const tx = await L1ERC721BridgeProxy.populateTransaction.changeAdmin(
        SystemDictator.address
      )
      console.log(`Please transfer L1ERC721Bridge (proxy) owner to MSD`)
      console.log(`L1ERC721BridgeProxy address: ${L1ERC721BridgeProxy.address}`)
      console.log(`MSD address: ${SystemDictator.address}`)
      console.log(`JSON:`)
      console.log(jsonifyTransaction(tx))
    }

    // Wait for the ownership transfer to complete.
    await awaitCondition(
      async () => {
        const owner = await L1ERC721BridgeProxy.callStatic.admin({
          from: ethers.constants.AddressZero,
        })
        return owner === SystemDictator.address
      },
      30000,
      1000
    )
  } else {
    console.log(`L1ERC721Bridge already owned by MSD`)
  }

  // Step 1 is a freebie, it doesn't impact the system.
  await doStep({
    isLiveDeployer,
    SystemDictator,
    step: 1,
    message: `
      Step 1 will configure the ProxyAdmin contract, you can safely execute this step at any time
      without impacting the functionality of the rest of the system.
    `,
    checks: async () => {
      await assertContractVariable(
        ProxyAdmin,
        'addressManager',
        AddressManager.address
      )
      assert(
        (await ProxyAdmin.implementationName(
          getDeploymentAddress(hre, 'Proxy__OVM_L1CrossDomainMessenger')
        )) === 'OVM_L1CrossDomainMessenger'
      )
      assert(
        (await ProxyAdmin.proxyType(
          getDeploymentAddress(hre, 'Proxy__OVM_L1CrossDomainMessenger')
        )) === 2
      )
      assert(
        (await ProxyAdmin.proxyType(
          getDeploymentAddress(hre, 'Proxy__OVM_L1StandardBridge')
        )) === 1
      )

      // Check the SystemConfig was initialized properly.
      await assertContractVariable(
        SystemConfigProxy,
        'owner',
        hre.deployConfig.finalSystemOwner
      )
      await assertContractVariable(
        SystemConfigProxy,
        'overhead',
        hre.deployConfig.gasPriceOracleOverhead
      )
      await assertContractVariable(
        SystemConfigProxy,
        'scalar',
        hre.deployConfig.gasPriceOracleScalar
      )
      await assertContractVariable(
        SystemConfigProxy,
        'batcherHash',
        ethers.utils.hexZeroPad(
          hre.deployConfig.batchSenderAddress.toLowerCase(),
          32
        )
      )
      await assertContractVariable(
        SystemConfigProxy,
        'gasLimit',
        hre.deployConfig.l2GenesisBlockGasLimit
      )
    },
  })

  // Step 2 shuts down the system.
  await doStep({
    isLiveDeployer,
    SystemDictator,
    step: 2,
    message: `
      Step 2 will stop deposits and withdrawals via the L1CrossDomainMessenger and will stop the
      DTL from syncing new deposits via the CTC, effectively shutting down the legacy system. Once
      this step has been executed, you should immediately begin the L2 migration process. If you
      need to restart the system, run exit1() followed by finalize().
    `,
    checks: async () => {
      assert(
        (await AddressManager.getAddress('OVM_L1CrossDomainMessenger')) ===
          ethers.constants.AddressZero
      )
    },
  })
}

deployFn.tags = ['SystemDictatorSteps', 'phase1']

export default deployFn
