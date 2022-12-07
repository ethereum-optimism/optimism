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
} from '../src/deploy-utils'

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
      console.log('using a live deployer')
      isLiveDeployer = true
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

  // Set up required contract references.
  const [
    SystemDictator,
    ProxyAdmin,
    AddressManager,
    L1CrossDomainMessenger,
    L1StandardBridgeProxy,
    L1StandardBridgeProxyWithSigner,
    L1StandardBridge,
    L2OutputOracle,
    OptimismPortal,
    OptimismMintableERC20Factory,
    L1ERC721BridgeProxy,
    L1ERC721BridgeProxyWithSigner,
    L1ERC721Bridge,
  ] = await getContractsFromArtifacts(hre, [
    {
      name: 'SystemDictator',
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
      name: 'Proxy__OVM_L1CrossDomainMessenger',
      iface: 'L1CrossDomainMessenger',
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
      name: 'Proxy__OVM_L1StandardBridge',
      iface: 'L1StandardBridge',
      signerOrProvider: deployer,
    },
    {
      name: 'L2OutputOracleProxy',
      iface: 'L2OutputOracle',
      signerOrProvider: deployer,
    },
    {
      name: 'OptimismPortalProxy',
      iface: 'OptimismPortal',
      signerOrProvider: deployer,
    },
    {
      name: 'OptimismMintableERC20FactoryProxy',
      iface: 'OptimismMintableERC20Factory',
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
      name: 'L1ERC721BridgeProxy',
      iface: 'L1ERC721Bridge',
      signerOrProvider: deployer,
    },
  ])

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
      console.log(`Please transfer AddressManager owner to MSD`)
      console.log(`MSD address: ${SystemDictator.address}`)
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

  // Transfer ownership of the L1CrossDomainMessenger to SystemDictator.
  if (
    needsProxyTransfer &&
    (await AddressManager.getAddress('OVM_L1CrossDomainMessenger')) !==
      ethers.constants.AddressZero &&
    (await L1CrossDomainMessenger.owner()) !== SystemDictator.address
  ) {
    if (isLiveDeployer) {
      console.log(`Setting L1CrossDomainMessenger owner to MSD`)
      await L1CrossDomainMessenger.transferOwnership(SystemDictator.address)
    } else {
      console.log(`Please transfer L1CrossDomainMessenger owner to MSD`)
      console.log(`MSD address: ${SystemDictator.address}`)
    }

    // Wait for the ownership transfer to complete.
    await awaitCondition(
      async () => {
        const owner = await L1CrossDomainMessenger.owner()
        return owner === SystemDictator.address
      },
      30000,
      1000
    )
  } else {
    console.log(`L1CrossDomainMessenger already owned by MSD`)
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
      console.log(`Please transfer L1StandardBridge (proxy) owner to MSD`)
      console.log(`MSD address: ${SystemDictator.address}`)
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
      console.log(`Please transfer L1ERC721Bridge (proxy) owner to MSD`)
      console.log(`MSD address: ${SystemDictator.address}`)
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

  const checks = {
    1: async () => {
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
    },
    2: async () => {
      assert(
        (await AddressManager.getAddress('OVM_L1CrossDomainMessenger')) ===
          ethers.constants.AddressZero
      )
    },
    3: async () => {
      const deads = [
        'OVM_CanonicalTransactionChain',
        'OVM_L2CrossDomainMessenger',
        'OVM_DecompressionPrecompileAddress',
        'OVM_Sequencer',
        'OVM_Proposer',
        'OVM_ChainStorageContainer-CTC-batches',
        'OVM_ChainStorageContainer-CTC-queue',
        'OVM_CanonicalTransactionChain',
        'OVM_StateCommitmentChain',
        'OVM_BondManager',
        'OVM_ExecutionManager',
        'OVM_FraudVerifier',
        'OVM_StateManagerFactory',
        'OVM_StateTransitionerFactory',
        'OVM_SafetyChecker',
        'OVM_L1MultiMessageRelayer',
        'BondManager',
      ]
      for (const dead of deads) {
        assert(
          (await AddressManager.getAddress(dead)) ===
            ethers.constants.AddressZero
        )
      }
    },
    4: async () => {
      await assertContractVariable(AddressManager, 'owner', ProxyAdmin.address)
      assert(
        (await L1StandardBridgeProxy.callStatic.getOwner({
          from: ethers.constants.AddressZero,
        })) === ProxyAdmin.address
      )
    },
    5: async () => {
      // Check L2OutputOracle was initialized properly.
      await assertContractVariable(
        L2OutputOracle,
        'latestBlockNumber',
        hre.deployConfig.l2OutputOracleStartingBlockNumber
      )

      // Check OptimismPortal was initialized properly.
      await assertContractVariable(
        OptimismPortal,
        'l2Sender',
        '0x000000000000000000000000000000000000dEaD'
      )
      const resourceParams = await OptimismPortal.params()
      assert(
        resourceParams.prevBaseFee.eq(await OptimismPortal.INITIAL_BASE_FEE()),
        `OptimismPortal was not initialized with the correct initial base fee`
      )
      assert(
        resourceParams.prevBoughtGas.eq(0),
        `OptimismPortal was not initialized with the correct initial bought gas`
      )
      assert(
        !resourceParams.prevBlockNum.eq(0),
        `OptimismPortal was not initialized with the correct initial block number`
      )
      assert(
        (await hre.ethers.provider.getBalance(L1StandardBridge.address)).eq(0)
      )

      // Check L1CrossDomainMessenger was initialized properly.
      await assertContractVariable(L1CrossDomainMessenger, 'paused', true)
      try {
        await L1CrossDomainMessenger.xDomainMessageSender()
        assert(false, `L1CrossDomainMessenger was not initialized properly`)
      } catch (err) {
        // Expected.
      }
      await assertContractVariable(
        L1CrossDomainMessenger,
        'owner',
        SystemDictator.address
      )

      // Check L1StandardBridge was initialized properly.
      await assertContractVariable(
        L1StandardBridge,
        'messenger',
        L1CrossDomainMessenger.address
      )
      assert(
        (await hre.ethers.provider.getBalance(L1StandardBridge.address)).eq(0)
      )

      // Check OptimismMintableERC20Factory was initialized properly.
      await assertContractVariable(
        OptimismMintableERC20Factory,
        'bridge',
        L1StandardBridge.address
      )

      // Check L1ERC721Bridge was initialized properly.
      await assertContractVariable(
        L1ERC721Bridge,
        'messenger',
        L1CrossDomainMessenger.address
      )
    },
    6: async () => {
      await assertContractVariable(L1CrossDomainMessenger, 'paused', false)
    },
  }

  for (let i = 1; i <= 6; i++) {
    const currentStep = await SystemDictator.currentStep()
    if (currentStep === i) {
      if (
        currentStep > (await SystemDictator.PROXY_TRANSFER_STEP()) &&
        !(await SystemDictator.dynamicConfigSet())
      ) {
        if (isLiveDeployer) {
          console.log(`Updating dynamic oracle config...`)

          // Use default starting time if not provided
          let deployL2StartingTimestamp =
            hre.deployConfig.l2OutputOracleStartingTimestamp
          if (deployL2StartingTimestamp < 0) {
            const l1StartingBlock = await hre.ethers.provider.getBlock(
              hre.deployConfig.l1StartingBlockTag
            )
            if (l1StartingBlock === null) {
              throw new Error(
                `Cannot fetch block tag ${hre.deployConfig.l1StartingBlockTag}`
              )
            }
            deployL2StartingTimestamp = l1StartingBlock.timestamp
          }

          await SystemDictator.updateL2OutputOracleDynamicConfig({
            l2OutputOracleStartingBlockNumber:
              hre.deployConfig.l2OutputOracleStartingBlockNumber,
            l2OutputOracleStartingTimestamp: deployL2StartingTimestamp,
          })
        } else {
          console.log(`Please update dynamic oracle config...`)
        }

        await awaitCondition(
          async () => {
            return SystemDictator.dynamicConfigSet()
          },
          30000,
          1000
        )
      }

      if (isLiveDeployer) {
        console.log(`Executing step ${i}...`)
        await SystemDictator[`step${i}`]()
      } else {
        console.log(`Please execute step ${i}...`)
      }

      await awaitCondition(
        async () => {
          const step = await SystemDictator.currentStep()
          return step === i + 1
        },
        30000,
        1000
      )

      // Run post step checks
      await checks[i]()
    } else {
      console.log(`Step ${i} executed`)
    }
  }

  if ((await SystemDictator.currentStep()) === 7) {
    if (isLiveDeployer) {
      console.log(`Finalizing deployment...`)
      await SystemDictator.finalize()
    } else {
      console.log(`Please finalize deployment...`)
    }

    await awaitCondition(
      async () => {
        return SystemDictator.finalized()
      },
      30000,
      1000
    )

    await assertContractVariable(L1CrossDomainMessenger, 'owner', finalOwner)
    await assertContractVariable(ProxyAdmin, 'owner', finalOwner)
  }
}

deployFn.tags = ['SystemDictatorSteps']

export default deployFn
