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
  doOwnershipTransfer,
  doPhase,
} from '../src/deploy-utils'

const uint128Max = ethers.BigNumber.from('0xffffffffffffffffffffffffffffffff')

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
    await doOwnershipTransfer({
      isLiveDeployer,
      proxy: ProxyAdmin,
      name: 'ProxyAdmin',
      transferFunc: 'transferOwnership',
      dictator: SystemDictator,
    })
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
    await doOwnershipTransfer({
      isLiveDeployer,
      proxy: AddressManager,
      name: 'AddressManager',
      transferFunc: 'transferOwnership',
      dictator: SystemDictator,
    })
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
    await doOwnershipTransfer({
      isLiveDeployer,
      proxy: L1StandardBridgeProxyWithSigner,
      name: 'L1StandardBridgeProxy',
      transferFunc: 'setOwner',
      dictator: SystemDictator,
    })
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
    await doOwnershipTransfer({
      isLiveDeployer,
      proxy: L1ERC721BridgeProxyWithSigner,
      name: 'L1ERC721BridgeProxy',
      transferFunc: 'changeAdmin',
      dictator: SystemDictator,
    })
  } else {
    console.log(`L1ERC721Bridge already owned by MSD`)
  }

  // Wait for the ownership transfers to complete before continuing.
  await awaitCondition(
    async (): Promise<boolean> => {
      const proxyAdminOwner = await ProxyAdmin.owner()
      const addressManagerOwner = await AddressManager.owner()
      const l1StandardBridgeOwner =
        await L1StandardBridgeProxy.callStatic.getOwner({
          from: ethers.constants.AddressZero,
        })
      const l1Erc721BridgeOwner = await L1ERC721BridgeProxy.callStatic.admin({
        from: ethers.constants.AddressZero,
      })

      return (
        proxyAdminOwner === SystemDictator.address &&
        addressManagerOwner === SystemDictator.address &&
        l1StandardBridgeOwner === SystemDictator.address &&
        l1Erc721BridgeOwner === SystemDictator.address
      )
    },
    5000,
    1000
  )

  await doPhase({
    isLiveDeployer,
    SystemDictator,
    phase: 1,
    message: `
      Phase 1 includes the following steps:

      Step 1 will configure the ProxyAdmin contract, you can safely execute this step at any time
      without impacting the functionality of the rest of the system.

      Step 2 will stop deposits and withdrawals via the L1CrossDomainMessenger and will stop the
      DTL from syncing new deposits via the CTC, effectively shutting down the legacy system. Once
      this step has been executed, you should immediately begin the L2 migration process. If you
      need to restart the system, run exit1() followed by finalize().
    `,
    checks: async () => {
      // Step 1 checks
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

      const config = await SystemConfigProxy.resourceConfig()
      assert(config.maxResourceLimit === 20_000_000)
      assert(config.elasticityMultiplier === 10)
      assert(config.baseFeeMaxChangeDenominator === 8)
      assert(config.systemTxMaxGas === 1_000_000)
      assert(ethers.utils.parseUnits('1', 'gwei').eq(config.minimumBaseFee))
      assert(config.maximumBaseFee.eq(uint128Max))

      // Step 2 checks
      const messenger = await AddressManager.getAddress(
        'OVM_L1CrossDomainMessenger'
      )
      assert(messenger === ethers.constants.AddressZero)
    },
  })
}

deployFn.tags = ['SystemDictatorSteps', 'phase1', 'l1']

export default deployFn
